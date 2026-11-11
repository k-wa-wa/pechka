package postgres

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// PoolResetter represents a connection pool capable of discarding existing connections.
// *pgxpool.Pool satisfies this interface.
type PoolResetter interface {
	Reset()
}

// RetryConfig holds settings for transient error retries.
type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Sleep        func(ctx context.Context, d time.Duration) error
	IsRetryable  func(err error) bool
}

func defaultSleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// DefaultRetryConfig returns standard retry settings for PostgreSQL failover recovery on read / idempotent operations.
// MaxAttempts and backoff are designed to cover both planned switchovers (~1-3s) and extended failovers (~15s+).
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  8,
		InitialDelay: 250 * time.Millisecond,
		MaxDelay:     4 * time.Second,
		Multiplier:   1.6,
		Sleep:        defaultSleep,
		IsRetryable:  IsRetryableRead,
	}
}

// DefaultWriteRetryConfig returns retry settings for non-idempotent write operations.
// It uses IsRetryableWrite to ensure requests with unconfirmed execution are never duplicated.
func DefaultWriteRetryConfig() RetryConfig {
	cfg := DefaultRetryConfig()
	cfg.IsRetryable = IsRetryableWrite
	return cfg
}

// ShouldResetPool determines whether the connection pool should close all connections.
// Restricted to server state changes (demoted old primary 25006, or operator shutdown 57xxx)
// to avoid thundering-herd reconnect storms on single connection drops.
func ShouldResetPool(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "25006" || strings.HasPrefix(pgErr.Code, "57") {
			return true
		}
	}
	return false
}

// IsRetryable determines whether an error indicates a transient connection or failover state.
// Alias for IsRetryableRead for backward compatibility.
func IsRetryable(err error) bool {
	return IsRetryableRead(err)
}

// IsRetryableRead checks if an error is safe to retry for read or idempotent operations.
func IsRetryableRead(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return false
	}

	// PostgreSQL wire protocol errors (PgError)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch {
		// 25006: read_only_sql_transaction (query directed to old primary demoted to standby)
		case pgErr.Code == "25006":
			return true
		// 57xxx: Class 57 — Operator Intervention (admin_shutdown, crash_shutdown, cannot_connect_now)
		case strings.HasPrefix(pgErr.Code, "57"):
			return true
		// 08xxx: Class 08 — Connection Exception (connection_exception, connection_failure, etc.)
		case strings.HasPrefix(pgErr.Code, "08"):
			return true
		default:
			return false
		}
	}

	// ConnectError indicates failure while opening a new connection
	var connErr *pgconn.ConnectError
	if errors.As(err, &connErr) {
		return true
	}

	// Network / Socket errors
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	if errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ETIMEDOUT) {
		return true
	}

	var netOpErr *net.OpError
	if errors.As(err, &netOpErr) {
		return true
	}

	// Fallback text check for wrapped / unexported driver errors
	msg := strings.ToLower(err.Error())
	for _, substr := range []string{
		"connection reset",
		"broken pipe",
		"connection refused",
		"server closed the connection",
		"conn closed",
		"read-only transaction",
		"the database system is shutting down",
		"the database system is starting up",
		"failed to connect to",
		"driver: bad connection",
	} {
		if strings.Contains(msg, substr) {
			return true
		}
	}

	return false
}

// IsRetryableWrite checks if an error is safe to retry for NON-IDEMPOTENT write operations.
// Only returns true if the error is guaranteed to have occurred before data reached the server,
// or if the server explicitly rejected the statement before modifying state (e.g. read-only demotion).
// If a connection severed midway through or after sending data, it returns false to prevent duplicates.
func IsRetryableWrite(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return false
	}

	// If pgconn explicitly guarantees that no query bytes were sent to server
	if pgconn.SafeToRetry(err) {
		return true
	}

	// Failure when establishing new connection
	var connErr *pgconn.ConnectError
	if errors.As(err, &connErr) {
		return true
	}

	// Explicit server-side rejection where server guaranteed NO modification occurred
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "25006": // read_only_sql_transaction (rejected as standby, transaction aborted)
			return true
		case "57P03": // cannot_connect_now (rejected during startup/shutdown)
			return true
		default:
			return false
		}
	}

	return false
}

// WithRetry executes op for read or idempotent operations, retrying transient PostgreSQL failover errors.
func WithRetry[T any](ctx context.Context, pool PoolResetter, op func() (T, error)) (T, error) {
	return WithRetryConfig(ctx, pool, DefaultRetryConfig(), op)
}

// WithRetryWrite executes op for NON-IDEMPOTENT write operations (e.g. INSERT).
// Retries only when guaranteed safe to do so without creating duplicate records.
func WithRetryWrite[T any](ctx context.Context, pool PoolResetter, op func() (T, error)) (T, error) {
	return WithRetryConfig(ctx, pool, DefaultWriteRetryConfig(), op)
}

// WithRetryConfig executes op with custom RetryConfig.
func WithRetryConfig[T any](ctx context.Context, pool PoolResetter, cfg RetryConfig, op func() (T, error)) (T, error) {
	var zero T
	delay := cfg.InitialDelay
	if delay <= 0 {
		delay = 250 * time.Millisecond
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 1
	}
	if cfg.Multiplier < 1.0 {
		cfg.Multiplier = 1.6
	}
	if cfg.Sleep == nil {
		cfg.Sleep = defaultSleep
	}
	isRetryable := cfg.IsRetryable
	if isRetryable == nil {
		isRetryable = IsRetryableRead
	}

	var lastErr error
	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		val, err := op()
		if err == nil {
			return val, nil
		}
		lastErr = err
		if !isRetryable(err) || attempt >= cfg.MaxAttempts {
			return val, err
		}
		if pool != nil && ShouldResetPool(err) {
			pool.Reset()
		}
		slog.WarnContext(ctx, "transient postgres error during operation, retrying with backoff",
			"attempt", attempt,
			"max_attempts", cfg.MaxAttempts,
			"error", err,
			"backoff", delay,
		)
		if sleepErr := cfg.Sleep(ctx, delay); sleepErr != nil {
			return zero, sleepErr
		}
		delay = time.Duration(float64(delay) * cfg.Multiplier)
		if cfg.MaxDelay > 0 && delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}
	return zero, lastErr
}

// WithRetryVoid executes op that returns no value with default retry configuration.
func WithRetryVoid(ctx context.Context, pool PoolResetter, op func() error) error {
	return WithRetryVoidConfig(ctx, pool, DefaultRetryConfig(), op)
}

// WithRetryVoidConfig executes op with custom RetryConfig.
func WithRetryVoidConfig(ctx context.Context, pool PoolResetter, cfg RetryConfig, op func() error) error {
	_, err := WithRetryConfig(ctx, pool, cfg, func() (struct{}, error) {
		return struct{}{}, op()
	})
	return err
}

// WithRetryWriteVoid executes a non-idempotent void operation.
func WithRetryWriteVoid(ctx context.Context, pool PoolResetter, op func() error) error {
	return WithRetryVoidConfig(ctx, pool, DefaultWriteRetryConfig(), op)
}
