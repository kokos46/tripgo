package transaction

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) error
	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row
}

type Manager struct {
	Pool         *pgxpool.Pool
	QueryTimeout time.Duration
}

type contextKey struct{}

type txExecutor struct{ tx pgx.Tx }

func (e txExecutor) Exec(ctx context.Context, sql string, arguments ...any) error {
	_, err := e.tx.Exec(ctx, sql, arguments...)
	return err
}

func (e txExecutor) QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row {
	return e.tx.QueryRow(ctx, sql, arguments...)
}

func (m *Manager) Do(ctx context.Context, fn func(context.Context) error) (err error) {
	if _, ok := ctx.Value(contextKey{}).(Executor); ok {
		return fn(ctx)
	}
	txCtx := ctx
	cancel := func() {}
	if m.QueryTimeout > 0 {
		txCtx, cancel = context.WithTimeout(ctx, m.QueryTimeout)
	}
	defer cancel()

	tx, err := m.Pool.BeginTx(txCtx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	txCtx = context.WithValue(txCtx, contextKey{}, Executor(txExecutor{tx: tx}))
	defer func() {
		if recovered := recover(); recovered != nil {
			rollbackCtx, rollbackCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer rollbackCancel()
			_ = tx.Rollback(rollbackCtx)
			panic(recovered)
		}
		if err != nil {
			rollbackCtx, rollbackCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer rollbackCancel()
			if rollbackErr := tx.Rollback(rollbackCtx); rollbackErr != nil && rollbackErr != pgx.ErrTxClosed {
				err = fmt.Errorf("%w (rollback transaction: %v)", err, rollbackErr)
			}
			return
		}
		if commitErr := tx.Commit(txCtx); commitErr != nil {
			err = fmt.Errorf("commit transaction: %w", commitErr)
		}
	}()
	err = fn(txCtx)
	return err
}

func ExecutorFromContext(ctx context.Context) (Executor, bool) {
	executor, ok := ctx.Value(contextKey{}).(Executor)
	return executor, ok
}
