// internal/repository/entity/store.go
package entity

import (
	"context"
	"database/sql"
	"fmt"
)

// Store menyediakan semua fungsi Query generated + kemampuan Transaction
type Store interface {
	Querier // Embed interface Querier agar Store juga bisa melakukan query biasa
	ExecTx(ctx context.Context, fn func(*Queries) error) error
}

// SQLStore adalah implementasi konkret dari Store
type SQLStore struct {
	*Queries // Embed struct Queries generated
	db       *sql.DB
}

// NewStore menggantikan New() biasa
func NewStore(db *sql.DB) *SQLStore {
	return &SQLStore{
		db:      db,
		Queries: New(db),
	}
}

// ExecTx mengeksekusi function di dalam database transaction
func (store *SQLStore) ExecTx(ctx context.Context, fn func(*Queries) error) error {
	// 1. Mulai Transaksi
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// 2. Buat instance Queries baru yang menggunakan Transaksi ini
	q := New(tx)

	// 3. Jalankan logic bisnis (function callback)
	err = fn(q)

	// 4. Handle Error: Rollback atau Commit
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}
