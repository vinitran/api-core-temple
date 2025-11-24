package datastore

import (
	"context"

	"github.com/stephenafamo/bob"
)

type TxFunc func(ctx context.Context, exec bob.Executor) error

type TxRunner interface {
	Run(ctx context.Context, fn TxFunc) error
}

type txRunner struct {
	pool PGXPool
}

func NewTxRunner(pool PGXPool) TxRunner {
	return &txRunner{pool: pool}
}

func (r *txRunner) Run(ctx context.Context, fn TxFunc) error {
	return RunInTx(ctx, r.pool, fn)
}

func RunInTx(ctx context.Context, pool PGXPool, fn TxFunc) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}

	exec := NewBobExecutorFromTx(tx)
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	if err := fn(ctx, exec); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	committed = true
	return nil
}
