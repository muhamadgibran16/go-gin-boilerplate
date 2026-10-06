package database

import (
	"context"

	"gorm.io/gorm"
)

type txKey struct{}

// Transactor runs functions inside a database transaction.
//
// The transaction travels in the context, so repositories join it automatically
// as long as they get their handle with database.Conn(ctx, r.db):
//
//	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
//		if err := s.orders.Create(ctx, order); err != nil {
//			return err // rolls back
//		}
//		return s.items.CreateMany(ctx, items)
//	}) // commits when fn returns nil
type Transactor struct {
	db *gorm.DB
}

// NewTransactor creates a Transactor
func NewTransactor(db *gorm.DB) *Transactor {
	return &Transactor{db: db}
}

// WithinTx runs fn in a transaction. It commits if fn returns nil and rolls back if fn
// returns an error or panics. When ctx already carries a transaction, fn joins it.
func (t *Transactor) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return fn(ctx)
	}
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

// Conn returns the transaction carried by ctx, or db bound to ctx when there is none.
// Repositories should use it instead of db.WithContext(ctx).
func Conn(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return db.WithContext(ctx)
}
