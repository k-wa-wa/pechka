package fallback_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/k-wa-wa/pechka/api/internal/repository/fallback"
)

func TestCircuitBreaker_SuccessClosed(t *testing.T) {
	cb := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{
		Name:             "test",
		Timeout:          100 * time.Millisecond,
		FailureThreshold: 3,
		Cooldown:         10 * time.Second,
	})

	ctx := context.Background()
	primaryCalled := false
	fallbackCalled := false

	res, err := fallback.Execute(
		ctx,
		cb,
		"test_op",
		"mongo",
		func(ctx context.Context) (string, error) {
			primaryCalled = true
			return "primary_data", nil
		},
		func(ctx context.Context) (string, error) {
			fallbackCalled = true
			return "fallback_data", nil
		},
		func(err error) bool { return err != nil },
	)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if res != "primary_data" {
		t.Errorf("expected primary_data, got %s", res)
	}
	if !primaryCalled {
		t.Error("expected primary to be called")
	}
	if fallbackCalled {
		t.Error("fallback should not be called on primary success")
	}
	if cb.State() != fallback.StateClosed {
		t.Errorf("expected StateClosed, got %v", cb.State())
	}
}

func TestCircuitBreaker_SingleFailureFallsBack(t *testing.T) {
	cb := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{
		Name:             "test",
		Timeout:          100 * time.Millisecond,
		FailureThreshold: 3,
		Cooldown:         10 * time.Second,
	})

	ctx := context.Background()
	res, err := fallback.Execute(
		ctx,
		cb,
		"test_op",
		"mongo",
		func(ctx context.Context) (string, error) {
			return "", errors.New("primary down")
		},
		func(ctx context.Context) (string, error) {
			return "fallback_data", nil
		},
		func(err error) bool { return err != nil },
	)

	if err != nil {
		t.Fatalf("expected nil error after fallback, got %v", err)
	}
	if res != "fallback_data" {
		t.Errorf("expected fallback_data, got %s", res)
	}
	// Still closed after 1 failure because threshold is 3
	if cb.State() != fallback.StateClosed {
		t.Errorf("expected StateClosed, got %v", cb.State())
	}
}

func TestCircuitBreaker_TimeoutTriggersFallback(t *testing.T) {
	cb := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{
		Name:             "test",
		Timeout:          50 * time.Millisecond,
		FailureThreshold: 3,
		Cooldown:         10 * time.Second,
	})

	ctx := context.Background()
	start := time.Now()
	res, err := fallback.Execute(
		ctx,
		cb,
		"test_op",
		"mongo",
		func(callCtx context.Context) (string, error) {
			select {
			case <-time.After(500 * time.Millisecond):
				return "slow_data", nil
			case <-callCtx.Done():
				return "", callCtx.Err()
			}
		},
		func(ctx context.Context) (string, error) {
			return "fallback_data", nil
		},
		fallback.IsTimeoutOrContextError,
	)

	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("expected fallback to succeed, got %v", err)
	}
	if res != "fallback_data" {
		t.Errorf("expected fallback_data, got %s", res)
	}
	if elapsed > 250*time.Millisecond {
		t.Errorf("expected fast timeout around 50ms, took %v", elapsed)
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	now := time.Now()
	var stateChanges []string

	cb := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{
		Name:             "test",
		Timeout:          50 * time.Millisecond,
		FailureThreshold: 3,
		Cooldown:         30 * time.Second,
		Now:              func() time.Time { return now },
		OnStateChange: func(name string, from, to fallback.State) {
			stateChanges = append(stateChanges, from.String()+"->"+to.String())
		},
	})

	ctx := context.Background()
	failPrimary := func(ctx context.Context) (string, error) {
		return "", errors.New("connection refused")
	}
	fallbackOk := func(ctx context.Context) (string, error) {
		return "fallback_data", nil
	}
	isFailure := func(err error) bool { return err != nil }

	// Failure 1
	_, _ = fallback.Execute(ctx, cb, "op", "mongo", failPrimary, fallbackOk, isFailure)
	if cb.State() != fallback.StateClosed {
		t.Fatalf("expected closed after 1 failure, got %v", cb.State())
	}

	// Failure 2
	_, _ = fallback.Execute(ctx, cb, "op", "mongo", failPrimary, fallbackOk, isFailure)
	if cb.State() != fallback.StateClosed {
		t.Fatalf("expected closed after 2 failures, got %v", cb.State())
	}

	// Failure 3 -> should open
	_, _ = fallback.Execute(ctx, cb, "op", "mongo", failPrimary, fallbackOk, isFailure)
	if cb.State() != fallback.StateOpen {
		t.Fatalf("expected open after 3 failures, got %v", cb.State())
	}

	if len(stateChanges) != 1 || stateChanges[0] != "CLOSED->OPEN" {
		t.Errorf("unexpected state changes: %v", stateChanges)
	}

	// Next call while OPEN: primary should NOT be called at all
	primaryCalled := false
	res, err := fallback.Execute(
		ctx,
		cb,
		"op",
		"mongo",
		func(ctx context.Context) (string, error) {
			primaryCalled = true
			return "primary", nil
		},
		fallbackOk,
		isFailure,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if primaryCalled {
		t.Error("primary was called while circuit breaker was OPEN")
	}
	if res != "fallback_data" {
		t.Errorf("expected fallback_data, got %s", res)
	}

	// Advance time past cooldown (30s) -> should allow probe (Half-Open)
	now = now.Add(31 * time.Second)
	probePrimaryCalled := false
	probePrimarySuccess := func(ctx context.Context) (string, error) {
		probePrimaryCalled = true
		return "recovered_data", nil
	}

	res, err = fallback.Execute(ctx, cb, "op", "mongo", probePrimarySuccess, fallbackOk, isFailure)
	if err != nil {
		t.Fatalf("unexpected error on probe: %v", err)
	}
	if !probePrimaryCalled {
		t.Error("expected probe call to primary in half-open state")
	}
	if res != "recovered_data" {
		t.Errorf("expected recovered_data, got %s", res)
	}

	// Breaker should now be closed again
	if cb.State() != fallback.StateClosed {
		t.Errorf("expected StateClosed after successful probe, got %v", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenFailsBackToOpen(t *testing.T) {
	now := time.Now()
	cb := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{
		Name:             "test",
		Timeout:          50 * time.Millisecond,
		FailureThreshold: 1, // 1 failure to open
		Cooldown:         10 * time.Second,
		Now:              func() time.Time { return now },
	})

	ctx := context.Background()
	failPrimary := func(ctx context.Context) (string, error) {
		return "", errors.New("fail")
	}
	fallbackOk := func(ctx context.Context) (string, error) {
		return "fallback", nil
	}
	isFailure := func(err error) bool { return err != nil }

	// Failure 1 -> Opens
	_, _ = fallback.Execute(ctx, cb, "op", "mongo", failPrimary, fallbackOk, isFailure)
	if cb.State() != fallback.StateOpen {
		t.Fatalf("expected Open, got %v", cb.State())
	}

	// Advance time past cooldown
	now = now.Add(11 * time.Second)

	// Probe call fails -> should transition back to Open
	_, _ = fallback.Execute(ctx, cb, "op", "mongo", failPrimary, fallbackOk, isFailure)
	if cb.State() != fallback.StateOpen {
		t.Errorf("expected StateOpen after failed probe, got %v", cb.State())
	}
}

func TestCircuitBreaker_NonFailureErrorIgnored(t *testing.T) {
	errNotFound := errors.New("document not found")
	cb := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{
		Name:             "test",
		Timeout:          50 * time.Millisecond,
		FailureThreshold: 2,
		Cooldown:         10 * time.Second,
	})

	ctx := context.Background()
	isPrimaryFailure := func(err error) bool {
		return !errors.Is(err, errNotFound)
	}

	for i := 0; i < 5; i++ {
		fallbackCalled := false
		_, err := fallback.Execute(
			ctx,
			cb,
			"op",
			"mongo",
			func(ctx context.Context) (string, error) {
				return "", errNotFound
			},
			func(ctx context.Context) (string, error) {
				fallbackCalled = true
				return "fallback", nil
			},
			isPrimaryFailure,
		)

		if !errors.Is(err, errNotFound) {
			t.Fatalf("expected errNotFound, got %v", err)
		}
		if fallbackCalled {
			t.Error("fallback should not be called for non-failure error")
		}
		if cb.State() != fallback.StateClosed {
			t.Errorf("expected StateClosed, got %v", cb.State())
		}
	}
}

func TestCircuitBreaker_ConcurrentRequests(t *testing.T) {
	cb := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{
		Name:             "test_concurrent",
		Timeout:          50 * time.Millisecond,
		FailureThreshold: 5,
		Cooldown:         5 * time.Second,
	})

	ctx := context.Background()
	var primaryCount atomic.Int64
	var fallbackCount atomic.Int64

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, _ = fallback.Execute(
				ctx,
				cb,
				"op",
				"mongo",
				func(ctx context.Context) (string, error) {
					primaryCount.Add(1)
					if id%2 == 0 {
						return "", errors.New("intermittent fail")
					}
					return "ok", nil
				},
				func(ctx context.Context) (string, error) {
					fallbackCount.Add(1)
					return "fallback", nil
				},
				func(err error) bool { return err != nil },
			)
		}(i)
	}
	wg.Wait()

	total := primaryCount.Load() + fallbackCount.Load()
	if total < 50 {
		t.Errorf("expected at least 50 total calls, got %d", total)
	}
}

func TestCircuitBreaker_StateTransitionsAll(t *testing.T) {
	now := time.Now()
	var stateChanges []string
	var mu sync.Mutex

	cb := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{
		Name:             "test_transitions",
		Timeout:          50 * time.Millisecond,
		FailureThreshold: 2,
		Cooldown:         10 * time.Second,
		Now:              func() time.Time { return now },
		OnStateChange: func(name string, from, to fallback.State) {
			mu.Lock()
			defer mu.Unlock()
			stateChanges = append(stateChanges, from.String()+"->"+to.String())
		},
	})

	ctx := context.Background()
	failPrimary := func(ctx context.Context) (string, error) {
		return "", errors.New("network failure")
	}
	fallbackOk := func(ctx context.Context) (string, error) {
		return "fallback_ok", nil
	}
	isFailure := func(err error) bool { return err != nil }

	// 1. Fail twice to trigger CLOSED -> OPEN
	_, _ = fallback.Execute(ctx, cb, "op", "mongo", failPrimary, fallbackOk, isFailure)
	_, _ = fallback.Execute(ctx, cb, "op", "mongo", failPrimary, fallbackOk, isFailure)

	// 2. Advance time past cooldown to trigger OPEN -> HALF-OPEN on next query
	now = now.Add(11 * time.Second)
	successPrimary := func(ctx context.Context) (string, error) {
		return "recovered", nil
	}
	res, err := fallback.Execute(ctx, cb, "op", "mongo", successPrimary, fallbackOk, isFailure)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != "recovered" {
		t.Errorf("expected recovered, got %s", res)
	}

	mu.Lock()
	defer mu.Unlock()
	expected := []string{"CLOSED->OPEN", "OPEN->HALF-OPEN", "HALF-OPEN->CLOSED"}
	if len(stateChanges) != len(expected) {
		t.Fatalf("expected state changes %v, got %v", expected, stateChanges)
	}
	for i, exp := range expected {
		if stateChanges[i] != exp {
			t.Errorf("transition %d: expected %s, got %s", i, exp, stateChanges[i])
		}
	}
}

func TestCircuitBreaker_ParentContextCancellation(t *testing.T) {
	cb := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{
		Name:             "test_cancel",
		Timeout:          200 * time.Millisecond,
		FailureThreshold: 1,
		Cooldown:         10 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	primaryCalled := false
	fallbackCalled := false

	// Cancel parent context inside primary execution
	_, err := fallback.Execute(
		ctx,
		cb,
		"op",
		"mongo",
		func(callCtx context.Context) (string, error) {
			primaryCalled = true
			cancel() // simulate client aborting request mid-flight
			return "", context.Canceled
		},
		func(ctx context.Context) (string, error) {
			fallbackCalled = true
			return "fallback", nil
		},
		fallback.IsTimeoutOrContextError,
	)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if !primaryCalled {
		t.Error("expected primary to be called")
	}
	if fallbackCalled {
		t.Error("fallback should not be called when parent context was canceled")
	}
	// Breaker should NOT be open because parent context was canceled, not an infrastructure failure
	if cb.State() != fallback.StateClosed {
		t.Errorf("expected circuit breaker to remain CLOSED, got %v", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenParentContextCancellation(t *testing.T) {
	now := time.Now()
	cb := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{
		Name:             "test_half_open_cancel",
		Timeout:          100 * time.Millisecond,
		FailureThreshold: 1,
		Cooldown:         10 * time.Second,
		Now:              func() time.Time { return now },
	})

	ctx := context.Background()

	// 1. Force breaker to OPEN
	_, _ = fallback.Execute(
		ctx,
		cb,
		"op",
		"mongo",
		func(ctx context.Context) (string, error) {
			return "", errors.New("initial failure")
		},
		func(ctx context.Context) (string, error) {
			return "fallback", nil
		},
		func(err error) bool { return err != nil },
	)

	if cb.State() != fallback.StateOpen {
		t.Fatalf("expected StateOpen, got %v", cb.State())
	}

	// 2. Advance time past cooldown to enter HALF-OPEN on next query
	now = now.Add(11 * time.Second)

	// 3. During HALF-OPEN probe, caller cancels parent context
	canceledCtx, cancel := context.WithCancel(context.Background())
	probeCalled := false
	_, err := fallback.Execute(
		canceledCtx,
		cb,
		"op",
		"mongo",
		func(callCtx context.Context) (string, error) {
			probeCalled = true
			cancel() // simulate client disconnecting during probe
			return "", context.Canceled
		},
		func(ctx context.Context) (string, error) {
			return "fallback", nil
		},
		fallback.IsTimeoutOrContextError,
	)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if !probeCalled {
		t.Fatal("expected probe to be called in HALF-OPEN state")
	}

	// 4. The breaker must NOT be stuck in HALF-OPEN with in-flight blocked.
	// Next normal request must be allowed to probe primary again.
	nextProbeCalled := false
	recoveredRes, err := fallback.Execute(
		context.Background(),
		cb,
		"op",
		"mongo",
		func(callCtx context.Context) (string, error) {
			nextProbeCalled = true
			return "recovered_data", nil
		},
		func(ctx context.Context) (string, error) {
			return "fallback", nil
		},
		func(err error) bool { return err != nil },
	)

	if err != nil {
		t.Fatalf("unexpected error on recovery probe: %v", err)
	}
	if !nextProbeCalled {
		t.Fatal("circuit breaker was stuck! Expected next probe to be allowed to call primary")
	}
	if recoveredRes != "recovered_data" {
		t.Errorf("expected recovered_data, got %s", recoveredRes)
	}
	if cb.State() != fallback.StateClosed {
		t.Errorf("expected StateClosed after recovery, got %v", cb.State())
	}
}
