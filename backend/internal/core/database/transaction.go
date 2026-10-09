package database

import (
	"context"
	"fmt"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2"
	trmmanager "github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Tx is a PostgreSQL transaction exposed to repositories in an application scenario.
type Tx = pgx.Tx

// TransactionManager runs an application scenario in a single transaction. The
// callback receives a tx-scoped context so decorators (e.g. the realtime emitter)
// can attach per-transaction state.
type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(context.Context, Tx) error) error
}

// PostgresTransactionManager adapts Avito's go-transaction-manager to the current
// service contract that requires an explicit pgx.Tx.
type PostgresTransactionManager struct {
	pool    *pgxpool.Pool
	manager trm.Manager
}

// NewTransactionManager creates a transaction manager for the PostgreSQL pool.
func NewTransactionManager(pool *pgxpool.Pool) *PostgresTransactionManager {
	return &PostgresTransactionManager{
		pool:    pool,
		manager: trmmanager.Must(trmpgx.NewDefaultFactory(pool)),
	}
}

// WithinTransaction runs fn in a transaction created and finished by
// go-transaction-manager. The library also handles rollback, panic, and nested
// transactions correctly.
func (m *PostgresTransactionManager) WithinTransaction(ctx context.Context, fn func(context.Context, Tx) error) error {
	return m.manager.Do(ctx, func(txCtx context.Context) error {
		connection := trmpgx.DefaultCtxGetter.DefaultTrOrDB(txCtx, m.pool)
		tx, ok := connection.(pgx.Tx)
		if !ok {
			return fmt.Errorf("transaction manager did not provide pgx transaction")
		}
		return fn(txCtx, tx)
	})
}

var _ TransactionManager = (*PostgresTransactionManager)(nil)
