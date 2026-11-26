package platform

import (
	"sync"
	"testing"
	"time"
)

// TestPostFromAnotherGoroutineIsQueuedNotRun is the defect this queue exists
// to fix: a callback posted from a worker/timer goroutine used to run on that
// goroutine, concurrently with the UI thread.
func TestPostFromAnotherGoroutineIsQueuedNotRun(t *testing.T) {
	q := NewTaskQueue(nil)
	var ran bool
	done := make(chan struct{})

	go func() {
		defer close(done)
		q.Post(func() { ran = true })
	}()
	<-done

	if ran {
		t.Fatal("callback ran on the posting goroutine; it must wait for Drain")
	}
	if q.Pending() != 1 {
		t.Fatalf("Pending = %d, want 1", q.Pending())
	}

	q.EnterUIThread()
	q.Drain()
	if !ran {
		t.Fatal("Drain did not run the queued callback")
	}
	if q.Pending() != 0 {
		t.Fatalf("Pending after drain = %d, want 0", q.Pending())
	}
}

// TestPostOnUIThreadRunsInline preserves today's behaviour for callers that
// are already on the UI thread.
func TestPostOnUIThreadRunsInline(t *testing.T) {
	q := NewTaskQueue(nil)
	q.EnterUIThread()

	var ran bool
	q.Post(func() { ran = true })
	if !ran {
		t.Fatal("Post on the UI thread should run inline")
	}
	if q.Pending() != 0 {
		t.Fatalf("inline Post left %d queued", q.Pending())
	}
}

// TestOnUIThreadIsThreadSpecific pins that the check is per-goroutine, not a
// global flag.
func TestOnUIThreadIsThreadSpecific(t *testing.T) {
	q := NewTaskQueue(nil)
	q.EnterUIThread()
	if !q.OnUIThread() {
		t.Fatal("OnUIThread false on the thread that entered")
	}
	other := make(chan bool, 1)
	go func() { other <- q.OnUIThread() }()
	if <-other {
		t.Fatal("OnUIThread true on a different goroutine")
	}
}

// TestPostAfterRespectsDueTime pins the timer path (InvokeTaskAfter).
func TestPostAfterRespectsDueTime(t *testing.T) {
	q := NewTaskQueue(nil)
	q.EnterUIThread()

	var ran bool
	q.PostAfter(func() { ran = true }, 30*time.Millisecond)

	q.Drain()
	if ran {
		t.Fatal("PostAfter ran before its due time")
	}
	time.Sleep(60 * time.Millisecond)
	q.Drain()
	if !ran {
		t.Fatal("PostAfter did not run after its due time")
	}
}

// TestDrainRunsInSubmissionOrder pins FIFO ordering across the queue.
func TestDrainRunsInSubmissionOrder(t *testing.T) {
	q := NewTaskQueue(nil)
	var mu sync.Mutex
	var order []int
	done := make(chan struct{})

	go func() {
		defer close(done)
		for i := 0; i < 5; i++ {
			i := i
			q.Post(func() {
				mu.Lock()
				order = append(order, i)
				mu.Unlock()
			})
		}
	}()
	<-done

	q.EnterUIThread()
	q.Drain()

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 5 {
		t.Fatalf("ran %d tasks, want 5", len(order))
	}
	for i, v := range order {
		if v != i {
			t.Fatalf("order = %v, want 0..4", order)
		}
	}
}

// TestDrainIsReentrantSafe pins that a task may post more work: the new work
// runs on the following drain rather than recursively.
func TestDrainIsReentrantSafe(t *testing.T) {
	q := NewTaskQueue(nil)
	var second bool
	done := make(chan struct{})

	// Post from another goroutine so the outer callback really runs inside
	// Drain (a UI-thread Post would run inline and prove nothing here).
	go func() {
		defer close(done)
		q.Post(func() {
			q.PostAfter(func() { second = true }, 0)
		})
	}()
	<-done

	q.EnterUIThread()
	q.Drain()
	if second {
		t.Fatal("work posted during Drain should run on the next drain")
	}
	q.Drain()
	if !second {
		t.Fatal("queued work did not run on the next drain")
	}
}

func TestTaskQueueNilSafe(t *testing.T) {
	q := NewTaskQueue(nil)
	q.EnterUIThread()
	q.Post(nil)
	q.PostAfter(nil, time.Millisecond)
	q.Drain()
	if q.Pending() != 0 {
		t.Fatalf("Pending = %d, want 0", q.Pending())
	}
}

func TestDefaultQueueHelpers(t *testing.T) {
	q := NewTaskQueue(nil)
	SetDefaultQueue(q)
	if DefaultQueue() != q {
		t.Fatal("SetDefaultQueue did not install the queue")
	}
	EnterUIThread()
	var ran bool
	InvokeTask(func() { ran = true })
	if !ran {
		t.Fatal("InvokeTask on the UI thread should run inline")
	}
	InvokeTaskAfter(func() {}, time.Millisecond)
	DrainTasks()
	// Restore a clean default so other packages' tests are unaffected.
	SetDefaultQueue(NewTaskQueue(nil))
}
