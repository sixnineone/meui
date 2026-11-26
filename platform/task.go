// Package platform is the OS-facing seam: the contract the toolkit uses for
// windows, events, input, clipboard, dialogs, displays and cross-thread work,
// with the renderer/windowing backend supplied behind it.
//
// This file holds the cross-thread task API, which is the part of the contract
// the toolkit depends on most: widget code runs on timer and worker goroutines
// and must marshal mutations onto the UI thread.
//
// THE ONE EXCEPTION is real-time audio (meui-media/audio, MEDIA.md D4). A device
// callback runs on SDL's own audio thread and must not allocate, lock, or hop to
// the UI thread: marshalling an audio callback is how a glitch ships. So audio
// callbacks are the documented exception — they pull from a lock-free ring on the
// audio thread, and only the app-facing state changes are marshalled here. Every
// other callback in the tree follows the UI-thread rule above.
package platform

import (
	"log"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

// TaskQueue is the UI-thread work queue.
//
// Threading contract:
//
//   - The queue has exactly one UI thread: the goroutine that called
//     EnterUIThread (the loop that drives frames).
//   - Post from that goroutine runs the callback inline, immediately.
//   - Post from any other goroutine enqueues it; it runs on the UI thread at
//     the next Drain, in submission order.
//   - PostAfter always enqueues with a due time; it runs at the first Drain
//     after that time.
//   - Drain must only be called on the UI thread (the frame loop calls it at
//     frame start, before layout and paint).
//
// The inline case matters: it keeps the previous synchronous behaviour for
// callers that are already on the UI thread, while fixing the real defect —
// callbacks fired from timers and worker goroutines used to run on those
// goroutines and mutate the widget tree concurrently with paint.
type TaskQueue struct {
	mu     sync.Mutex
	uiGoid int64
	tasks  []queuedTask
	wake   func()
}

type queuedTask struct {
	fn  func()
	due time.Time
}

// NewTaskQueue returns an empty queue. wake, when non-nil, is called after a
// cross-thread enqueue so a backend can poke its event loop; a continuously
// running frame loop needs no wake.
func NewTaskQueue(wake func()) *TaskQueue {
	return &TaskQueue{wake: wake}
}

// EnterUIThread marks the calling goroutine as the queue's UI thread.
func (q *TaskQueue) EnterUIThread() {
	q.mu.Lock()
	q.uiGoid = goid()
	q.mu.Unlock()
}

// OnUIThread reports whether the caller is the queue's UI thread.
func (q *TaskQueue) OnUIThread() bool {
	q.mu.Lock()
	ui := q.uiGoid
	q.mu.Unlock()
	return ui != 0 && goid() == ui
}

// Post runs fn on the UI thread: inline when already there, otherwise queued
// for the next Drain.
func (q *TaskQueue) Post(fn func()) {
	if fn == nil {
		return
	}
	if q.OnUIThread() {
		fn()
		return
	}
	q.enqueue(queuedTask{fn: fn})
}

// PostAfter queues fn to run on the UI thread no earlier than d from now.
func (q *TaskQueue) PostAfter(fn func(), d time.Duration) {
	if fn == nil {
		return
	}
	q.enqueue(queuedTask{fn: fn, due: time.Now().Add(d)})
}

func (q *TaskQueue) enqueue(t queuedTask) {
	q.mu.Lock()
	q.tasks = append(q.tasks, t)
	wake := q.wake
	q.mu.Unlock()
	if wake != nil {
		wake()
	}
}

// Drain runs every task that is due, in submission order, on the UI thread.
func (q *TaskQueue) Drain() {
	q.mu.Lock()
	now := time.Now()
	ready := make([]func(), 0, len(q.tasks))
	pending := q.tasks[:0]
	for _, t := range q.tasks {
		if t.due.IsZero() || !t.due.After(now) {
			ready = append(ready, t.fn)
			continue
		}
		pending = append(pending, t)
	}
	q.tasks = pending
	q.mu.Unlock()

	for _, fn := range ready {
		fn()
	}
	if os.Getenv("TRADEUI_UPD_DEBUG") != "" && len(ready) > 0 {
		log.Printf("[upd] Drain ran %d task(s)", len(ready))
	}
}

// Pending reports how many tasks are waiting (including not-yet-due timers).
func (q *TaskQueue) Pending() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.tasks)
}

// --- default queue --------------------------------------------------------
//
// The toolkit posts through these package-level helpers, so widget code needs
// no queue handle. The backend installs its own queue with SetDefaultQueue.

var (
	defaultMu    sync.RWMutex
	defaultQueue = NewTaskQueue(nil)
)

// SetDefaultQueue replaces the process-wide queue (called by a backend's
// startup).
func SetDefaultQueue(q *TaskQueue) {
	if q == nil {
		return
	}
	defaultMu.Lock()
	defaultQueue = q
	defaultMu.Unlock()
}

// DefaultQueue returns the process-wide queue.
func DefaultQueue() *TaskQueue {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultQueue
}

// EnterUIThread marks the calling goroutine as the default queue's UI thread.
func EnterUIThread() { DefaultQueue().EnterUIThread() }

// OnUIThread reports whether the caller is on the default queue's UI thread.
func OnUIThread() bool { return DefaultQueue().OnUIThread() }

// InvokeTask runs fn on the UI thread (inline when already there).
func InvokeTask(fn func()) { DefaultQueue().Post(fn) }

// InvokeTaskAfter queues fn to run on the UI thread after d.
func InvokeTaskAfter(fn func(), d time.Duration) { DefaultQueue().PostAfter(fn, d) }

// DrainTasks runs the due tasks on the UI thread.
func DrainTasks() { DefaultQueue().Drain() }

// goid returns the current goroutine's id.
//
// Go does not expose goroutine identity, and the queue needs it to decide
// whether a caller is already on the UI thread. Parsing the runtime's own
// header is the standard pure-Go way to get it without cgo, and it is only
// consulted on Post, never on a per-frame path.
func goid() int64 {
	var buf [32]byte
	n := runtime.Stack(buf[:], false)
	// Header is "goroutine <id> [".
	const prefix = "goroutine "
	s := buf[:n]
	if len(s) <= len(prefix) {
		return 0
	}
	s = s[len(prefix):]
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	id, err := strconv.ParseInt(string(s[:i]), 10, 64)
	if err != nil {
		return 0
	}
	return id
}
