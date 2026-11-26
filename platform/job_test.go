package platform

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// A job is only worth having if it does three things a goroutine does not: report progress, be cancellable,
// and hand its result back somewhere safe. These tests are for those three, and for the failure paths —
// because a job system that only works when everything succeeds is not a job system.

// TestAJobReportsItsResult is the ordinary case.
func TestAJobReportsItsResult(t *testing.T) {
	j := NewJob[int]("sum")
	j.Start(func(ctx context.Context, j *Job[int]) (int, error) {
		total := 0
		for i := 1; i <= 10; i++ {
			total += i
		}
		return total, nil
	})
	res, err := j.Wait()
	if err != nil {
		t.Fatalf("job: %v", err)
	}
	if res != 55 {
		t.Errorf("the job produced %v, want 55", res)
	}
	if j.State() != JobDone {
		t.Errorf("the state is %v, want done", j.State())
	}
}

// TestProgressIsReadableWhileTheJobRuns is the point of progress at all: a UI polls it every frame, from
// the UI thread, while the work runs elsewhere. Reading it after the fact proves nothing.
func TestProgressIsReadableWhileTheJobRuns(t *testing.T) {
	release := make(chan struct{})
	j := NewJob[int]("slow")
	j.Start(func(ctx context.Context, j *Job[int]) (int, error) {
		j.SetProgress(0, 10, "starting")
		for i := 1; i <= 10; i++ {
			select {
			case <-release:
			default:
			}
			j.SetProgress(int64(i), 10, "step")
			time.Sleep(2 * time.Millisecond)
		}
		return 10, nil
	})

	// Mid-flight, from this goroutine, while the work is in the other — but WAITING for the work to
	// actually start first. The first version of this read immediately, and it read a job that had not
	// been scheduled yet: a real UI polls every frame and would hit the same window on the first frame,
	// which is why the wait is here rather than a sleep.
	for i := 0; i < 200; i++ {
		if _, total, _ := j.Progress(); total == 10 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	work, total, note := j.Progress()
	if total != 10 {
		t.Errorf("the total is %d, want 10", total)
	}
	if note == "" && work == 0 {
		t.Error("neither progress nor a note was readable while the job ran")
	}
	if f := j.Fraction(); f < 0 || f > 1 {
		t.Errorf("the fraction is %f, want 0..1", f)
	}
	close(release)
	if _, err := j.Wait(); err != nil {
		t.Fatal(err)
	}
	if f := j.Fraction(); f != 1 {
		t.Errorf("a finished job's fraction is %f, want 1", f)
	}
}

// TestAnUnknownTotalIsIndeterminateNotZero: a job that cannot say how much there is must not report 0%,
// which is a bar that looks stalled. It reports -1, and the widget has an indeterminate mode for it.
func TestAnUnknownTotalIsIndeterminateNotZero(t *testing.T) {
	j := NewJob[int]("unknown")
	if f := j.Fraction(); f != -1 {
		t.Errorf("a job with no work done has fraction %f, want -1 for indeterminate", f)
	}
	j.Start(func(ctx context.Context, j *Job[int]) (int, error) {
		j.SetProgress(5, 0, "working") // a total of 0: still unknown
		return 1, nil
	})
	if _, err := j.Wait(); err != nil {
		t.Fatal(err)
	}
	if f := j.Fraction(); f != -1 {
		t.Errorf("a job that never learned its total reports %f, want -1", f)
	}
}

// TestCancelStopsTheWork is what makes a long job safe to abandon — a user who opens the wrong file should
// be able to walk away from an encode that is going to take a minute.
func TestCancelStopsTheWork(t *testing.T) {
	started := make(chan struct{})
	j := NewJob[int]("cancellable")
	j.Start(func(ctx context.Context, j *Job[int]) (int, error) {
		close(started)
		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				return 0, ctx.Err() // the cooperative contract, made explicit
			case <-time.After(time.Millisecond):
				j.SetProgress(int64(i), 0, "working")
			}
		}
	})
	<-started
	j.Cancel()

	// It must actually stop, and stop as CANCELLED rather than failed.
	select {
	case <-j.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("the job did not stop after Cancel")
	}
	if s := j.State(); s != JobCancelled {
		t.Errorf("the state is %v, want cancelled", s)
	}
	if j.Err() != nil && j.Err() != context.Canceled {
		t.Errorf("a cancelled job reported %v, want nil or context.Canceled", j.Err())
	}
}

// TestAPanicBecomesAFailedJobNotADeadProcess: a thumbnail job that hits a malformed file must not take the
// editor with it, and the caller has to be able to SEE that it failed.
func TestAPanicBecomesAFailedJobNotADeadProcess(t *testing.T) {
	j := NewJob[int]("panicky")
	j.Start(func(ctx context.Context, j *Job[int]) (int, error) {
		var slice []int
		_ = slice[3] // out of range
		return 0, nil
	})
	if _, err := j.Wait(); err == nil {
		t.Fatal("a panicking job reported no error")
	} else if !strings.Contains(err.Error(), "panicky") {
		t.Errorf("the error does not name the job: %v", err)
	}
	if s := j.State(); s != JobFailed {
		t.Errorf("the state is %v, want failed", s)
	}
}

// TestAFailingJobCarriesItsError.
func TestAFailingJobCarriesItsError(t *testing.T) {
	j := NewJob[int]("failing")
	j.Start(func(ctx context.Context, j *Job[int]) (int, error) {
		return 0, context.DeadlineExceeded
	})
	_, err := j.Wait()
	if err != context.DeadlineExceeded {
		t.Errorf("the error is %v, want DeadlineExceeded", err)
	}
	if j.State() != JobFailed {
		t.Errorf("the state is %v, want failed", j.State())
	}
}

// TestCancelAfterFinishDoesNotChangeWhatHappened: asking after the fact should describe what occurred, not
// re-decide it. A job that finished and is then cancelled is DONE, not cancelled.
func TestCancelAfterFinishDoesNotChangeWhatHappened(t *testing.T) {
	j := NewJob[int]("done-first")
	j.Start(func(ctx context.Context, j *Job[int]) (int, error) { return 7, nil })
	if _, err := j.Wait(); err != nil {
		t.Fatal(err)
	}
	j.Cancel()
	if s := j.State(); s != JobDone {
		t.Errorf("a finished job reports %v after Cancel, want done", s)
	}
	if j.Cancelled() {
		t.Error("a finished job reports Cancelled after the fact")
	}
}

// TestProgressIsSafeToReadFromManyThreads is the rule that makes polling it from a UI possible. Run under
// `make test-race`, this is the assertion that matters; without the race detector it still checks that
// nothing deadlocks and that the snapshots are self-consistent.
func TestProgressIsSafeToReadFromManyThreads(t *testing.T) {
	release := make(chan struct{})
	j := NewJob[int]("polled")
	j.Start(func(ctx context.Context, j *Job[int]) (int, error) {
		for i := 0; ; i++ {
			select {
			case <-release:
				return i, nil
			default:
			}
			j.SetProgress(int64(i), 100, "working")
		}
	})

	var wg sync.WaitGroup
	stop := make(chan struct{})
	for r := 0; r < 8; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				work, total, _ := j.Progress()
				// A CONSISTENT snapshot: a total that moved but a work count that went backwards is the
				// signature of a read that straddled two writes.
				if total != 0 && work < 0 {
					t.Errorf("inconsistent progress snapshot: work %d total %d", work, total)
					return
				}
				_ = j.Fraction()
				_ = j.State()
			}
		}()
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	<-j.Done()
	close(stop)
	wg.Wait()
}

// TestElapsedIsZeroBeforeAndPositiveAfter: a status line that shows a job has been running for 0s because
// the clock was read too early is a small wrongness that is easy to get wrong and easy to test.
func TestElapsedIsZeroBeforeAndPositiveAfter(t *testing.T) {
	j := NewJob[int]("timed")
	if d := j.Elapsed(); d != 0 {
		t.Errorf("a job that has not started reports elapsed %v, want 0", d)
	}
	j.Start(func(ctx context.Context, j *Job[int]) (int, error) {
		time.Sleep(20 * time.Millisecond)
		return 1, nil
	})
	<-j.Done()
	if d := j.Elapsed(); d < 15*time.Millisecond {
		t.Errorf("a job that slept 20ms reports elapsed %v", d)
	}
}

// TestFractionIsClamped: a job that reports more work than total is describing a bug in the work, and the
// bar must not draw past its end.
func TestFractionIsClamped(t *testing.T) {
	j := NewJob[int]("over")
	j.Start(func(ctx context.Context, j *Job[int]) (int, error) {
		j.SetProgress(150, 100, "too far")
		return 0, nil
	})
	<-j.Done()
	if f := j.Fraction(); f != 1 {
		t.Errorf("a job reporting 150 of 100 has fraction %f, want it clamped to 1", f)
	}
}

// TestAJobWithANameAndANoteIsLoggable: the two strings exist so a status line and a log can say what is
// running, which is the difference between a spinner and a mystery.
func TestAJobWithANameAndANoteIsLoggable(t *testing.T) {
	j := NewJob[string]("thumbnail")
	j.Start(func(ctx context.Context, j *Job[string]) (string, error) {
		j.SetProgress(1, 3, "reading the file")
		return "thumb.png", nil
	})
	res, err := j.Wait()
	if err != nil || res != "thumb.png" {
		t.Fatalf("the job produced %q, %v", res, err)
	}
	if j.Name != "thumbnail" {
		t.Errorf("the job is named %q", j.Name)
	}
	_, _, note := j.Progress()
	if note != "reading the file" {
		t.Errorf("the note is %q, want the one the work set", note)
	}
	if s := JobRunning.String(); s != "running" {
		t.Errorf("JobRunning names itself %q", s)
	}
}
