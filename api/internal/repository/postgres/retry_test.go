package postgres

import (
	"context"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type mockPoolResetter struct {
	resetCount atomic.Int32
}

func (m *mockPoolResetter) Reset() {
	m.resetCount.Add(1)
}

func TestIsRetryableRead(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "context canceled",
			err:  context.Canceled,
			want: false,
		},
		{
			name: "context deadline exceeded",
			err:  context.DeadlineExceeded,
			want: false,
		},
		{
			name: "pgx.ErrNoRows",
			err:  pgx.ErrNoRows,
			want: false,
		},
		{
			name: "PgError read_only_sql_transaction (25006)",
			err:  &pgconn.PgError{Code: "25006", Message: "cannot execute in a read-only transaction"},
			want: true,
		},
		{
			name: "PgError admin_shutdown (57P01)",
			err:  &pgconn.PgError{Code: "57P01", Message: "terminating connection due to administrator command"},
			want: true,
		},
		{
			name: "PgError cannot_connect_now (57P03)",
			err:  &pgconn.PgError{Code: "57P03", Message: "the database system is starting up"},
			want: true,
		},
		{
			name: "PgError connection_failure (08006)",
			err:  &pgconn.PgError{Code: "08006", Message: "connection failure"},
			want: true,
		},
		{
			name: "PgError too_many_connections (53300) - not retried to prevent thundering herd",
			err:  &pgconn.PgError{Code: "53300", Message: "too many connections"},
			want: false,
		},
		{
			name: "PgError unique_violation (23505) - not retryable",
			err:  &pgconn.PgError{Code: "23505", Message: "duplicate key value"},
			want: false,
		},
		{
			name: "PgError syntax_error (42601) - not retryable",
			err:  &pgconn.PgError{Code: "42601", Message: "syntax error"},
			want: false,
		},
		{
			name: "ConnectError",
			err:  &pgconn.ConnectError{},
			want: true,
		},
		{
			name: "syscall ECONNRESET",
			err:  syscall.ECONNRESET,
			want: true,
		},
		{
			name: "syscall ECONNREFUSED",
			err:  syscall.ECONNREFUSED,
			want: true,
		},
		{
			name: "syscall EPIPE",
			err:  syscall.EPIPE,
			want: true,
		},
		{
			name: "io.EOF",
			err:  io.EOF,
			want: true,
		},
		{
			name: "io.ErrUnexpectedEOF",
			err:  io.ErrUnexpectedEOF,
			want: true,
		},
		{
			name: "net.OpError",
			err:  &net.OpError{Op: "dial", Net: "tcp"},
			want: true,
		},
		{
			name: "wrapped connection reset message",
			err:  errors.New("read: connection reset by peer"),
			want: true,
		},
		{
			name: "wrapped server closed connection message",
			err:  errors.New("server closed the connection unexpectedly"),
			want: true,
		},
		{
			name: "generic logic error - not retryable",
			err:  errors.New("channel_id is required"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryableRead(tt.err)
			if got != tt.want {
				t.Errorf("IsRetryableRead(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestIsRetryableWrite(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "context canceled",
			err:  context.Canceled,
			want: false,
		},
		{
			name: "ConnectError (before query sent)",
			err:  &pgconn.ConnectError{},
			want: true,
		},
		{
			name: "PgError read_only_sql_transaction (25006, server rejected before mutate)",
			err:  &pgconn.PgError{Code: "25006", Message: "cannot execute in a read-only transaction"},
			want: true,
		},
		{
			name: "PgError cannot_connect_now (57P03, server rejected before mutate)",
			err:  &pgconn.PgError{Code: "57P03", Message: "the database system is starting up"},
			want: true,
		},
		{
			name: "syscall ECONNRESET after data sent (MUST NOT RETRY to prevent duplicate insert)",
			err:  syscall.ECONNRESET,
			want: false,
		},
		{
			name: "io.EOF after data sent (MUST NOT RETRY)",
			err:  io.EOF,
			want: false,
		},
		{
			name: "net.OpError after data sent (MUST NOT RETRY)",
			err:  &net.OpError{Op: "read", Net: "tcp"},
			want: false,
		},
		{
			name: "PgError unique_violation (23505) - not retryable",
			err:  &pgconn.PgError{Code: "23505", Message: "duplicate key value"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryableWrite(tt.err)
			if got != tt.want {
				t.Errorf("IsRetryableWrite(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestShouldResetPool(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "25006 read_only_sql_transaction",
			err:  &pgconn.PgError{Code: "25006"},
			want: true,
		},
		{
			name: "57P01 admin_shutdown",
			err:  &pgconn.PgError{Code: "57P01"},
			want: true,
		},
		{
			name: "08006 connection_failure - single conn drop should NOT reset pool",
			err:  &pgconn.PgError{Code: "08006"},
			want: false,
		},
		{
			name: "53300 too_many_connections - should NOT reset pool",
			err:  &pgconn.PgError{Code: "53300"},
			want: false,
		},
		{
			name: "syscall ECONNRESET - should NOT reset entire pool",
			err:  syscall.ECONNRESET,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldResetPool(tt.err)
			if got != tt.want {
				t.Errorf("ShouldResetPool(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestWithRetry_SuccessOnFirstTry(t *testing.T) {
	pool := &mockPoolResetter{}
	calls := 0
	val, err := WithRetry(context.Background(), pool, func() (string, error) {
		calls++
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "ok" {
		t.Errorf("got %q, want %q", val, "ok")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
	if pool.resetCount.Load() != 0 {
		t.Errorf("pool.Reset called %d times, want 0", pool.resetCount.Load())
	}
}

func TestWithRetry_RecoversAfterTransientErrors(t *testing.T) {
	pool := &mockPoolResetter{}
	calls := 0
	cfg := RetryConfig{
		MaxAttempts:  4,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     5 * time.Millisecond,
		Multiplier:   1.5,
		Sleep: func(ctx context.Context, d time.Duration) error {
			return nil
		},
	}

	val, err := WithRetryConfig(context.Background(), pool, cfg, func() (int, error) {
		calls++
		if calls < 3 {
			return 0, &pgconn.PgError{Code: "25006", Message: "read-only transaction during failover"}
		}
		return 42, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 42 {
		t.Errorf("got %d, want 42", val)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
	if pool.resetCount.Load() != 2 {
		t.Errorf("pool.Reset called %d times, want 2", pool.resetCount.Load())
	}
}

func TestWithRetry_PoolResetOnlyOnServerStateChange(t *testing.T) {
	pool := &mockPoolResetter{}
	calls := 0
	cfg := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond,
		Sleep: func(ctx context.Context, d time.Duration) error {
			return nil
		},
	}

	// For ECONNRESET, retry should happen but pool.Reset() should NOT be called
	val, err := WithRetryConfig(context.Background(), pool, cfg, func() (string, error) {
		calls++
		if calls == 1 {
			return "", syscall.ECONNRESET
		}
		return "recovered", nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "recovered" {
		t.Errorf("got %q, want recovered", val)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
	if pool.resetCount.Load() != 0 {
		t.Errorf("pool.Reset called %d times on socket drop, want 0", pool.resetCount.Load())
	}
}

func TestWithRetryWrite_DoesNotRetryOnSeveredConnection(t *testing.T) {
	pool := &mockPoolResetter{}
	calls := 0
	cfg := DefaultWriteRetryConfig()
	cfg.Sleep = func(ctx context.Context, d time.Duration) error { return nil }

	// If connection severed mid-flight (ECONNRESET), non-idempotent write must NOT retry
	_, err := WithRetryConfig(context.Background(), pool, cfg, func() (string, error) {
		calls++
		return "", syscall.ECONNRESET
	})

	if !errors.Is(err, syscall.ECONNRESET) {
		t.Fatalf("expected ECONNRESET, got %v", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (non-idempotent write should not retry on connection reset)", calls)
	}
}

func TestWithRetryWrite_RetriesOnReadOnlyTransaction(t *testing.T) {
	pool := &mockPoolResetter{}
	calls := 0
	cfg := DefaultWriteRetryConfig()
	cfg.Sleep = func(ctx context.Context, d time.Duration) error { return nil }

	// On 25006, server explicitly rejected without modifying state, so safe to retry
	val, err := WithRetryConfig(context.Background(), pool, cfg, func() (string, error) {
		calls++
		if calls == 1 {
			return "", &pgconn.PgError{Code: "25006", Message: "read-only transaction"}
		}
		return "created", nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "created" {
		t.Errorf("got %q, want created", val)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
	if pool.resetCount.Load() != 1 {
		t.Errorf("pool.Reset called %d times, want 1", pool.resetCount.Load())
	}
}

func TestWithRetry_ExhaustsMaxAttempts(t *testing.T) {
	pool := &mockPoolResetter{}
	calls := 0
	cfg := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     5 * time.Millisecond,
		Multiplier:   1.0,
		Sleep: func(ctx context.Context, d time.Duration) error {
			return nil
		},
	}

	failErr := &pgconn.PgError{Code: "57P01", Message: "admin_shutdown"}
	_, err := WithRetryConfig(context.Background(), pool, cfg, func() (string, error) {
		calls++
		return "", failErr
	})

	if !errors.Is(err, failErr) {
		t.Fatalf("expected failErr, got %v", err)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
	if pool.resetCount.Load() != 2 {
		t.Errorf("pool.Reset called %d times, want 2", pool.resetCount.Load())
	}
}

func TestWithRetry_NonRetryableFailsImmediately(t *testing.T) {
	pool := &mockPoolResetter{}
	calls := 0
	cfg := RetryConfig{
		MaxAttempts:  4,
		InitialDelay: 1 * time.Millisecond,
		Sleep: func(ctx context.Context, d time.Duration) error {
			return nil
		},
	}

	nonRetryErr := errors.New("channel_id is required")
	_, err := WithRetryConfig(context.Background(), pool, cfg, func() (string, error) {
		calls++
		return "", nonRetryErr
	})

	if !errors.Is(err, nonRetryErr) {
		t.Fatalf("expected nonRetryErr, got %v", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
	if pool.resetCount.Load() != 0 {
		t.Errorf("pool.Reset called %d times, want 0", pool.resetCount.Load())
	}
}

func TestWithRetry_ContextCancelledDuringBackoff(t *testing.T) {
	pool := &mockPoolResetter{}
	ctx, cancel := context.WithCancel(context.Background())

	cfg := RetryConfig{
		MaxAttempts:  4,
		InitialDelay: 10 * time.Millisecond,
		Sleep: func(ctx context.Context, d time.Duration) error {
			cancel() // cancel context during sleep
			return ctx.Err()
		},
	}

	failErr := syscall.ECONNRESET
	_, err := WithRetryConfig(ctx, pool, cfg, func() (string, error) {
		return "", failErr
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestWithRetryVoid(t *testing.T) {
	pool := &mockPoolResetter{}
	calls := 0
	cfg := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond,
		Sleep: func(ctx context.Context, d time.Duration) error {
			return nil
		},
	}

	err := WithRetryVoidConfig(context.Background(), pool, cfg, func() error {
		calls++
		if calls < 2 {
			return syscall.ECONNREFUSED
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}
