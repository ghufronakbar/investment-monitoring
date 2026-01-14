// pkg/filter/filter.go
package filter

import (
	"regexp"
	"strings"
)

type ReqFilter struct {
	Search    string  `json:"search"`
	Page      int     `json:"page"`
	Limit     int     `json:"limit"`
	Sort      string  `json:"sort"`   // asc|desc
	SortBy    string  `json:"sortBy"` // name|createdAt|updatedAt (input user)
	Category  string  `json:"category"`
	DateStart *string `json:"dateStart"`
	DateEnd   *string `json:"dateEnd"`
}

// Offset adalah nilai turunan (hasil kalkulasi), bukan input user
func (f ReqFilter) Offset() int {
	if f.Page < 1 {
		return 0
	}
	if f.Limit < 1 {
		return 0
	}
	return (f.Page - 1) * f.Limit
}

// SortByDB mengubah input user ke nama kolom yang dipakai SQL
var camelToSnakeRE = regexp.MustCompile(`([a-z0-9])([A-Z])`)

func camelToSnake(s string) string {
	// createdAt -> created_at, updatedAt -> updated_at, Name -> name
	s = camelToSnakeRE.ReplaceAllString(s, `$1_$2`)
	return strings.ToLower(s)
}

func (f ReqFilter) SortByDB() string {
	// Whitelist kolom DB yang valid (snake_case)
	allowed := map[string]struct{}{
		"id":         {},
		"name":       {},
		"created_at": {},
		"updated_at": {},
	}

	// Ubah camelCase input jadi snake_case
	sortBy := camelToSnake(f.SortBy)

	// Validasi hasilnya
	if _, ok := allowed[sortBy]; ok {
		return sortBy
	}
	return "id"
}
