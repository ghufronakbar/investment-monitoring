// sql.go
package utils

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// StringToNullString mengonversi string biasa/pointer ke sql.NullString
func StringToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

// NullStringToString mengonversi sql.NullString ke *string untuk JSON response
func NullStringToString(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

// NullStringToStringVal mengonversi sql.NullString ke string kosong jika null (untuk field non-pointer)
func NullStringToStringVal(ns sql.NullString) string {
	if !ns.Valid {
		return ""
	}
	return ns.String
}

func NullStringToNullTime(s *string) sql.NullTime {
	if s == nil {
		return sql.NullTime{Valid: false}
	}

	val := strings.TrimSpace(*s)
	if val == "" {
		return sql.NullTime{Valid: false}
	}

	// parse date YYYY-MM-DD
	t, err := time.Parse("2006-01-02", val)
	if err != nil {
		return sql.NullTime{Valid: false}
	}

	return sql.NullTime{Time: t, Valid: true}
}

func NullInt64ToNullInt64(i *int64) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: *i, Valid: true}
}

func NullInt32ToNullInt32(i *int32) sql.NullInt32 {
	if i == nil {
		return sql.NullInt32{Valid: false}
	}
	return sql.NullInt32{Int32: *i, Valid: true}
}

func NullBoolPtrToNullBool(b *bool) sql.NullBool {
	if b == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *b, Valid: true}
}

func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
