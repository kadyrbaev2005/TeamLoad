// Package store wraps the sqlc queries with a pgx pool and transactions.
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"teamload/internal/store/db"
	"teamload/migrations"
)

// Store gives access to queries, optionally inside a transaction.
type Store struct {
	Pool *pgxpool.Pool
	*db.Queries
}

// Open connects to Postgres.
func Open(ctx context.Context, url string) (*Store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Store{Pool: pool, Queries: db.New(pool)}, nil
}

// Close closes the pool.
func (s *Store) Close() { s.Pool.Close() }

// Migrate applies all pending goose migrations.
func (s *Store) Migrate(ctx context.Context) error {
	sqlDB := stdlib.OpenDBFromPool(s.Pool)
	defer func() { _ = sqlDB.Close() }()
	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.UpContext(ctx, sqlDB, ".")
}

// Tx runs fn in a transaction, committing when it returns nil.
func (s *Store) Tx(ctx context.Context, fn func(q *db.Queries) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(s.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// IsNotFound reports whether err is "no rows".
func IsNotFound(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// IsUniqueViolation reports whether err violates the named unique constraint
// (any unique constraint when name is empty).
func IsUniqueViolation(err error, name string) bool {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return name == "" || pg.ConstraintName == name
	}
	return false
}
