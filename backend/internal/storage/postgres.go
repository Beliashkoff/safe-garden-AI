// Package storage wires the pgxpool connection and exposes the sqlc-generated
// Queries plus a transactional helper used by the auth flow (refresh-token
// rotation must be atomic).
package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

// Store wraps the pgxpool and embeds the sqlc Queries so callers can invoke
// non-transactional queries directly (store.GetUserByID, ...). For atomic
// multi-statement work use ExecTx.
type Store struct {
	pool *pgxpool.Pool
	*db.Queries
}

// New opens a pgxpool against dsn, verifies connectivity, and returns a Store.
// The pool is closed if the initial Ping fails to avoid leaking sockets.
func New(ctx context.Context, dsn string) (*Store, error) {
	if dsn == "" {
		return nil, errors.New("storage.New: empty dsn")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("storage.New: parse dsn: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("storage.New: pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("storage.New: ping: %w", err)
	}
	return &Store{pool: pool, Queries: db.New(pool)}, nil
}

// Close releases all pooled connections. Safe to call once; subsequent calls
// are no-ops.
func (s *Store) Close() { s.pool.Close() }

// Ping checks DB reachability. Use a bounded context so a stalled DB does not
// hang readiness probes.
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// ExecTx runs fn inside a transaction. The function receives a *db.Queries
// bound to the transaction; on nil error the tx commits, on any error it
// rolls back. The default isolation level (read-committed) is sufficient for
// refresh-token rotation because the UNIQUE index on token_hash serializes
// conflicting inserts.
func (s *Store) ExecTx(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("storage.ExecTx: begin: %w", err)
	}
	defer func() {
		// Rollback is a no-op after a successful Commit.
		_ = tx.Rollback(ctx)
	}()
	if err := fn(s.WithTx(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("storage.ExecTx: commit: %w", err)
	}
	return nil
}
