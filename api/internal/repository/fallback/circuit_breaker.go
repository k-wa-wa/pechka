package fallback

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// State represents the state of the circuit breaker.
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF-OPEN"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", s)
	}
}

// CircuitBreakerSettings configures a CircuitBreaker.
type CircuitBreakerSettings struct {
	Name             string
	Timeout          time.Duration
	FailureThreshold int
	Cooldown         time.Duration
	OnStateChange    func(name string, from, to State)
	Now              func() time.Time
}

// CircuitBreaker provides protection against cascading failures with fast-fallback.
type CircuitBreaker struct {
	name             string
	timeout          time.Duration
	failureThreshold int
	cooldown         time.Duration
	onStateChange    func(name string, from, to State)
	now              func() time.Time

	mu                  sync.Mutex
	state               State
	consecutiveFailures int
	openedAt            time.Time
	halfOpenInFlight    bool
}

// NewCircuitBreaker creates a new CircuitBreaker with the given settings.
func NewCircuitBreaker(settings CircuitBreakerSettings) *CircuitBreaker {
	timeout := settings.Timeout
	if timeout <= 0 {
		timeout = 300 * time.Millisecond
	}
	threshold := settings.FailureThreshold
	if threshold <= 0 {
		threshold = 3
	}
	cooldown := settings.Cooldown
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	nowFn := settings.Now
	if nowFn == nil {
		nowFn = time.Now
	}

	return &CircuitBreaker{
		name:             settings.Name,
		timeout:          timeout,
		failureThreshold: threshold,
		cooldown:         cooldown,
		onStateChange:    settings.OnStateChange,
		now:              nowFn,
		state:            StateClosed,
	}
}

// Name returns the name of the circuit breaker.
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// If Open and cooldown elapsed, it can transition to HalfOpen upon next query.
	if cb.state == StateOpen && cb.now().Sub(cb.openedAt) >= cb.cooldown {
		return StateHalfOpen
	}
	return cb.state
}

// beforeCall determines whether the primary datastore can be queried.
func (cb *CircuitBreaker) beforeCall() (allowPrimary bool, state State) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := cb.now()

	switch cb.state {
	case StateClosed:
		return true, StateClosed

	case StateOpen:
		if now.Sub(cb.openedAt) >= cb.cooldown {
			cb.transitionToLocked(StateHalfOpen)
			cb.halfOpenInFlight = true
			return true, StateHalfOpen
		}
		return false, StateOpen

	case StateHalfOpen:
		if !cb.halfOpenInFlight {
			cb.halfOpenInFlight = true
			return true, StateHalfOpen
		}
		return false, StateHalfOpen
	}

	return false, cb.state
}

// reportSuccess reports that a primary call succeeded.
func (cb *CircuitBreaker) reportSuccess(fromState State) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if fromState == StateHalfOpen {
		cb.halfOpenInFlight = false
		cb.transitionToLocked(StateClosed)
	}
	cb.consecutiveFailures = 0
}

// reportFailure reports that a primary call failed.
func (cb *CircuitBreaker) reportFailure(fromState State) (opened bool, consecutiveFailures int) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if fromState == StateHalfOpen {
		cb.halfOpenInFlight = false
		cb.transitionToLocked(StateOpen)
		return true, cb.consecutiveFailures
	}

	cb.consecutiveFailures++
	consecutiveFailures = cb.consecutiveFailures
	if cb.consecutiveFailures >= cb.failureThreshold {
		cb.transitionToLocked(StateOpen)
		opened = true
	}
	return opened, consecutiveFailures
}

func (cb *CircuitBreaker) transitionToLocked(to State) {
	from := cb.state
	if from == to {
		return
	}
	cb.state = to
	if to == StateOpen {
		cb.openedAt = cb.now()
	}
	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, from, to)
	}
}

// Execute runs the primary function with timeout protection and circuit breaking.
// If the breaker is open or primary fails, it transparently falls back to fallbackFn.
func Execute[T any](
	ctx context.Context,
	cb *CircuitBreaker,
	operation string,
	backend string,
	primaryFn func(ctx context.Context) (T, error),
	fallbackFn func(ctx context.Context) (T, error),
	isPrimaryFailure func(error) bool,
) (T, error) {
	allow, state := cb.beforeCall()

	if !allow {
		slog.WarnContext(ctx, "circuit breaker is OPEN; skipping primary and routing directly to postgres",
			"breaker", cb.name,
			"backend", backend,
			"operation", operation,
			"state", state.String(),
		)
		RecordFallback(backend, operation)
		return fallbackFn(ctx)
	}

	callCtx, cancel := context.WithTimeout(ctx, cb.timeout)
	res, err := primaryFn(callCtx)
	cancel()

	if err == nil {
		cb.reportSuccess(state)
		return res, nil
	}

	// Check if this error is an infrastructure failure vs business logic error (e.g. not found)
	if isPrimaryFailure != nil && !isPrimaryFailure(err) {
		cb.reportSuccess(state)
		return res, err
	}

	opened, totalFailures := cb.reportFailure(state)
	slog.WarnContext(ctx, "primary datastore call failed; falling back to postgres",
		"breaker", cb.name,
		"backend", backend,
		"operation", operation,
		"consecutive_failures", totalFailures,
		"error", err,
	)
	if opened {
		slog.WarnContext(ctx, "circuit breaker opened due to consecutive failures",
			"breaker", cb.name,
			"backend", backend,
			"threshold", cb.failureThreshold,
			"cooldown", cb.cooldown.String(),
		)
	}
	RecordFallback(backend, operation)

	return fallbackFn(ctx)
}

// IsTimeoutOrContextError returns true if the error was caused by context deadline exceeded or cancellation.
func IsTimeoutOrContextError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}
