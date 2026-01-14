// internal/module/example/example_service/return.go
package subexample_service

import (
	"time"

	"github.com/google/uuid"
)

type SubExampleGetReturn struct {
	ID        uuid.UUID
	Name      string
	Price     int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SubExampleDetailReturn struct {
	ID        uuid.UUID
	Name      string
	Price     int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SubExampleCreateReturn struct {
	ID        uuid.UUID
	Name      string
	Price     int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SubExampleEditReturn struct {
	ID        uuid.UUID
	Name      string
	Price     int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SubExampleDeleteReturn struct {
	ID        uuid.UUID
	Name      string
	Price     int64
	CreatedAt time.Time
	UpdatedAt time.Time
}
