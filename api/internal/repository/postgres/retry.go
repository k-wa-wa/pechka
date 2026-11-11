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

// DefaultRetryConfig returns standard retry settings for PostgreSQL failover recovery.
// Designed to cover Patroni failover/switchover periods (typically 1-3 seconds).
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  4,
		InitialDelay: 250 * time.Millisecond,
		MaxDelay:     3 * time.Second,
		Multiplier:   2.0,
		Sleep:        defaultSleep,
	}
}

// IsRetryable determines whether an error indicates a transient connection or failover state
// (e.g. connection broken, operator shutdown, switchover to standby, primary election).
func IsRetryable(err error) bool {
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
		// 53300: too_many_connections
		case pgErr.Code == "53300":
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

// WithRetry executes op, retrying transient PostgreSQL failover errors with exponential backoff.
// On retryable errors, pool.Reset() is called to discard demoted/dead connections from the pool.
func WithRetry[T any](ctx context.Context, pool PoolResetter, op func() (T, error)) (T, error) {
	return WithRetryConfig(ctx, pool, DefaultRetryConfig(), op)
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
		cfg.Multiplier = 2.0
	}
	if cfg.Sleep == nil {
		cfg.Sleep = defaultSleep
	}

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		val, err := op()
		if err == nil {
			return val, nil
		}
		if !IsRetryable(err) || attempt >= cfg.MaxAttempts {
			return val, err
		}
		if pool != nil {
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
	return zero, nil
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
