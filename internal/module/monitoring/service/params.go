// internal/module/monitoring/service/params.go
package monitoring_service

import (
	"time"

	"github.com/google/uuid"
)

// ===== TASK ITEMS (WBS) =====

type CreateTaskParams struct {
	ProjectID   uuid.UUID
	Title       string     `json:"title" validate:"required"`
	Description string     `json:"description" validate:"required"`
	Deadline    time.Time  `json:"deadline" validate:"required"`
	Weight      int        `json:"weight" validate:"required,gte=0,lte=100"`
	ParentID    *uuid.UUID `json:"parentId"`
}

type UpdateTaskParams struct {
	TaskID      uuid.UUID
	Title       string     `json:"title" validate:"required"`
	Description string     `json:"description" validate:"required"`
	Deadline    time.Time  `json:"deadline" validate:"required"`
	Weight      int        `json:"weight" validate:"required,gte=0,lte=100"`
	ParentID    *uuid.UUID `json:"parentId"`
}

type GetTasksParams struct {
	ProjectID uuid.UUID
	UserID    uuid.UUID
}

type ValidateWBSParams struct {
	ProjectID uuid.UUID
	UserID    uuid.UUID
}

type DeleteTaskParams struct {
	TaskID uuid.UUID
}

// ===== PROGRESS LOGS =====

type SubmitProgressParams struct {
	TaskID             uuid.UUID
	VendorID           uuid.UUID // From JWT context
	UserID             uuid.UUID // From JWT context
	ProgressPercentage int       `json:"progressPercentage" validate:"required,gte=0,lte=100"`
	Report             string    `json:"report" validate:"required"`
	EvidenceImageUrls  []string  `json:"evidenceImageUrls" validate:"required,min=1,dive,url"`
}

type GetProgressLogsParams struct {
	ProjectID uuid.UUID
	UserID    uuid.UUID
}

// ===== PROGRESS VALIDATIONS =====

type ValidateProgressParams struct {
	ProgressLogID          uuid.UUID
	AuditorID              uuid.UUID // From JWT context
	UserID                 uuid.UUID // From JWT context
	RealProgressPercentage int       `json:"realProgressPercentage" validate:"required,gte=0,lte=100"`
	Comment                *string   `json:"comment"`
}

// ===== S-CURVE =====

type GetSCurveParams struct {
	ProjectID uuid.UUID
	UserID    uuid.UUID
}
