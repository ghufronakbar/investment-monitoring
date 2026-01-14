// internal/module/example/example_service/params.go
package subexample_service

import "github.com/google/uuid"

type SubExampleGetListParams struct {
	Search  string `json:"search"`
	Limit   int    `json:"limit"`
	Page    int    `json:"page"`
	Offset  int    `json:"offset"`
	SortBy  string `json:"sortBy"`
	SortDir string `json:"sortDir"`
}

var SORT_BY = []string{"name", "created_at", "updated_at", "price"}

type SubExampleCreateParams struct {
	Name   string    `json:"name" validate:"required"`
	Price  int64     `json:"price" validate:"required"`
	UserID uuid.UUID `json:"userId" validate:"required"`
}

type SubExampleEditParams struct {
	ID     uuid.UUID `json:"id" validate:"required"`
	Name   string    `json:"name" validate:"required"`
	Price  int64     `json:"price" validate:"required"`
	UserID uuid.UUID `json:"userId" validate:"required"`
}

type SubExampleDeleteParams struct {
	ID     uuid.UUID `json:"id" validate:"required"`
	UserID uuid.UUID `json:"userId" validate:"required"`
}
