package platform

// Bounded background work: start something that takes a while, watch how far it has got, and be able to
// stop it.
//
// WHY IT EXISTS. Three things this tree does are slow enough that running them on the UI thread freezes a
// window: importing a large image (a 20-megapixel PNG took over a second), encoding a video, and
// generating a proxy or a thumbnail. Each of those wants three things the toolkit does not otherwise have —
// a way to say how far along it is, a way to cancel it, and a way to hand the RESULT back to the thread
// allowed to touch widgets. This is that, and nothing more: it is deliberately NOT a scheduler, a pool, a
// priority queue or a persistence layer.
//
// THE TWO RULES, which are the same two the audio ring is built on and for different reasons:
//
//   - A Job's WORK runs on its own goroutine and must not touch the widget tree. Its RESULT is handed back
//     through the task queue, so it lands on the UI thread. A job that mutated a widget directly would
//     reintroduce exactly the defect the task queue exists to fix.
//   - A Job's PROGRESS is read from any thread and is safe to read at any time, including after the job
//     has finished or been cancelled. A UI that polls progress every frame must never block on the work.
//
// CANCELLATION IS COOPERATIVE, and that is not a limitation to work around later — it is the only kind
// there is. `Cancel` sets a flag the work checks; it does not kill a goroutine, because a goroutine that
// cannot be stopped cleanly is a goroutine that leaks a half-written file. A job that ignores its context
// will run to completion, and the docs say so rather than implying otherwise.

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// JobState is where a job is in its life.
type JobState int32

const (
	// JobPending is queued but not started.
	JobPending JobState = iota
	// JobRunning means the work function is executing.
	JobRunning
	// JobDone means it finished successfully.
	JobDone
	// JobFailed means it returned an error, or panicked.
	JobFailed
	// JobCancelled means it stopped because it was cancelled.
	JobCancelled
)

// String names the state, for logs and status lines.
func (s JobState) String() string {
	switch s {
	case JobPending:
		return "pending"
	case JobRunning:
		return "running"
	case JobDone:
		return "done"
	case JobFailed:
		return "failed"
	case JobCancelled:
		return "cancelled"
	}
	return "unknown"
}

// Job is one unit of bounded background work.
//
// T is whatever the work produces, so a job is not a "return nothing" abstraction: a thumbnail job yields
// an image, an import job yields a document, a proxy job yields a path.
type Job[T any] struct {
	// Name is for logs and for a status line. It is not shown unless a caller asks.
	Name string

	state  atomic.Int32
	done   chan struct{}
	result atomic.Pointer[T]
	err    error

	// cctx/cancel are created in NewJob, BEFORE the job is visible to any goroutine.
	//
	// They were created lazily behind a sync.Once, and the race detector found the hole: Once guarantees
	// the write happens-before a LATER call to Do, NOT against a caller already inside Do on another
	// goroutine — so `Cancel` on one goroutine read `j.cancel` while the work goroutine was writing it.
	// A cancel func is safe to CALL concurrently, so the only thing that had to be fixed was making sure
	// nothing writes the field after construction. Creating it up front removes the write entirely, which
	// is simpler than a second lock and cannot rot the same way.
	cctx   context.Context
	cancel context.CancelFunc

	mu      sync.Mutex
	work    int64
	total   int64
	note    string
	started time.Time
	ended   time.Time
}

// NewJob names a job. The work is started by Start.
func NewJob[T any](name string) *Job[T] {
	cctx, cancel := context.WithCancel(context.Background())
	return &Job[T]{Name: name, done: make(chan struct{}), cctx: cctx, cancel: cancel}
}

// Work is what a job does. The context is cancelled when the job is, so the work checks it between steps
// of something long — a decode loop, a row scan, an encode — and returns ctx.Err() when it sees it.
//
// The work is expected to call SetProgress. A job that never does is still correct; it is simply
// indeterminate, which is why the progress bar has an indeterminate mode.
type Work[T any] func(ctx context.Context, j *Job[T]) (T, error)

// Start runs the work on its own goroutine and returns immediately.
//
// The result is NOT returned here: a job is asynchronous by construction, and the caller's next question is
// always "when is it done", which is Wait or OnDone. Returning a value from Start would invite a caller to
// block on it and defeat the reason for having a job.
func (j *Job[T]) Start(w Work[T]) {
	j.setState(JobRunning)
	j.mu.Lock()
	j.started = time.Now()
	j.mu.Unlock()
	go func() {
		defer j.finish()
		// A panic in the work is turned into a failed job rather than a dead process: a thumbnail that
		// hits a malformed file must not take the editor down with it, and the caller needs to be able to
		// SEE that it failed. recover here rather than letting it propagate out of the goroutine, where
		// nothing could catch it anyway.
		var result T
		var err error
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = &PanicError{Job: j.Name, Value: r}
				}
			}()
			result, err = w(j.ctx(), j)
		}()
		if err != nil {
			if err == context.Canceled {
				j.setState(JobCancelled)
			} else {
				j.mu.Lock()
				j.err = err
				j.mu.Unlock()
				j.setState(JobFailed)
			}
			return
		}
		j.result.Store(&result)
		j.setState(JobDone)
	}()
}

// ctx is the job's own cancellable context, created on first use.
func (j *Job[T]) ctx() context.Context { return j.cctx }

func (j *Job[T]) finish() {
	j.mu.Lock()
	j.ended = time.Now()
	j.mu.Unlock()
	close(j.done)
}

func (j *Job[T]) setState(s JobState) { j.state.Store(int32(s)) }

// State is where the job is. Safe from any thread, at any time, including after it finished.
func (j *Job[T]) State() JobState { return JobState(j.state.Load()) }

// Done is closed when the job finishes, fails, or is cancelled. It is closed EXACTLY ONCE, so a job that
// fails and is then cancelled does not panic a second close.
func (j *Job[T]) Done() <-chan struct{} { return j.done }

// Err is the failure, or nil. It is only meaningful once the job is no longer running.
func (j *Job[T]) Err() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.err
}

// Result is what the work produced, or the zero T if the job has not finished or failed. Prefer Wait or
// OnDone: this does not block, so it is the one to call from a poll.
//
// It returns T rather than *T for the same reason Wait does — a `Job[*Document]` should hand back a
// `*Document`, not a `**Document`.
func (j *Job[T]) Result() T {
	if r := j.result.Load(); r != nil {
		return *r
	}
	var zero T
	return zero
}

// Wait blocks until the job finishes and returns its result and error. It is for a caller that genuinely
// has nothing to do until the work is over, which in a UI is rarely; OnDone is usually the right choice.
//
// It returns T, not *T, even though the zero value of T may itself be a pointer. A job that produces a
// document is `Job[*Document]`, and returning `*T` from that is a `**Document` — a level of indirection
// the caller has to see through at every call site for no benefit. The zero T is returned when the job
// failed, and the error says so.
func (j *Job[T]) Wait() (T, error) {
	<-j.done
	if r := j.result.Load(); r != nil {
		return *r, nil
	}
	var zero T
	return zero, j.Err()
}

// Cancel asks the job to stop. It is COOPERATIVE: the work checks its context and returns. A job whose work
// ignores the context will run to completion.
func (j *Job[T]) Cancel() {
	j.cancel() // created in NewJob, so this is a read of a field nothing else writes
}

// Cancelled reports whether Cancel has been called. The work uses this, or its context, to stop early.
func (j *Job[T]) Cancelled() bool {
	if s := j.State(); s == JobCancelled || s == JobDone || s == JobFailed {
		// A finished job is not "cancelled" just because it ended; asking afterwards should describe what
		// happened, not re-decide it.
		return s == JobCancelled
	}
	select {
	case <-j.cctx.Done():
		return true
	default:
		return false
	}
}

// SetProgress records how far the work has got. work is in units of total; a total of 0 means the amount
// is not yet known, which the progress bar shows as indeterminate rather than as a bar stuck at zero.
//
// It is safe from the work's own goroutine and never blocks on the reader.
func (j *Job[T]) SetProgress(work, total int64, note string) {
	j.mu.Lock()
	j.work, j.total, j.note = work, total, note
	j.mu.Unlock()
}

// Progress is the work done, the total, and a note. All three are a consistent snapshot, so a caller
// cannot read a work count from one step and a total from the next.
func (j *Job[T]) Progress() (work, total int64, note string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.work, j.total, j.note
}

// Fraction is progress as 0..1, or -1 when the total is not known. -1 rather than 0 because "nothing done"
// and "we do not know how much there is" are different states, and a bar that shows 0% for a job that
// cannot say how long it is is lying.
func (j *Job[T]) Fraction() float32 {
	work, total, _ := j.Progress()
	if total <= 0 {
		return -1
	}
	f := float32(work) / float32(total)
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// Elapsed is how long the job has been running, or ran for. It is 0 before it starts.
func (j *Job[T]) Elapsed() time.Duration {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.started.IsZero() {
		return 0
	}
	if j.ended.IsZero() {
		return time.Since(j.started)
	}
	return j.ended.Sub(j.started)
}

// PanicError is what a job reports when its work panicked, so a failure has a message and a job name
// rather than arriving as a dead process.
type PanicError struct {
	Job   string
	Value any
}

func (e *PanicError) Error() string {
	return "job " + e.Job + " panicked: " + toString(e.Value)
}

func toString(v any) string {
	switch t := v.(type) {
	case error:
		return t.Error()
	case string:
		return t
	default:
		return "unknown panic"
	}
}
