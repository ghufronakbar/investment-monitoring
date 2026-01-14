// internal/module/monitoring/service/return.go
package monitoring_service

import (
	"time"

	"github.com/google/uuid"
)

// ===== TASK ITEMS =====

type TaskReturn struct {
	ID                uuid.UUID     `json:"id"`
	Title             string        `json:"title"`
	Description       string        `json:"description"`
	Deadline          time.Time     `json:"deadline"`
	Weight            int           `json:"weight"`
	ParentID          *uuid.UUID    `json:"parentId"`
	ContractID        uuid.UUID     `json:"contractId"`
	Children          *[]TaskReturn `json:"children"`
	IsLeaf            bool          `json:"isLeaf"`
	ClaimedPercentage int           `json:"claimedPercentage"` // Latest contractor claim
	ActualPercentage  int           `json:"actualPercentage"`  // Latest auditor-approved
	IsCompleted       bool          `json:"isCompleted"`       // actual >= claimed
	CreatedAt         time.Time     `json:"createdAt"`
}

type TaskListReturn struct {
	Tasks       []TaskReturn `json:"tasks"`
	TotalWeight int          `json:"totalWeight"`
}

// ===== PROGRESS LOGS =====

type ProgressLogReturn struct {
	ID                  uuid.UUID         `json:"id"`
	Report              string            `json:"report"`
	ProgressPercentage  int               `json:"progressPercentage"`
	EvidenceImageUrls   []string          `json:"evidenceImageUrls"`
	SnapshotTitle       string            `json:"snapshotTitle"`
	SnapshotDescription string            `json:"snapshotDescription"`
	SnapshotWeight      int               `json:"snapshotWeight"`
	TaskItemID          uuid.UUID         `json:"taskItemId"`
	TaskTitle           string            `json:"taskTitle,omitempty"`
	VendorID            uuid.UUID         `json:"vendorId"`
	VendorName          string            `json:"vendorName,omitempty"`
	Validation          *ValidationReturn `json:"validation"`
	CreatedAt           time.Time         `json:"createdAt"`
}

type ValidationReturn struct {
	ID                     uuid.UUID `json:"id"`
	IsAccepted             bool      `json:"isAccepted"`
	ClaimedPercentage      int       `json:"claimedPercentage"`      // Original contractor claim
	RealProgressPercentage int       `json:"realProgressPercentage"` // Auditor's assessed value
	Comment                *string   `json:"comment,omitempty"`
	AuditorID              uuid.UUID `json:"auditorId"`
	CreatedAt              time.Time `json:"createdAt"`
}

// ===== S-CURVE =====

type SCurveReturn struct {
	ActualProgress      float64              `json:"actualProgress"`
	PlannedProgress     float64              `json:"plannedProgress"`
	TotalSnapshotWeight int                  `json:"totalSnapshotWeight"`
	TotalLogs           int                  `json:"totalLogs"`
	ApprovedLogs        int                  `json:"approvedLogs"`
	RejectedLogs        int                  `json:"rejectedLogs"`
	TaskDetails         []TaskProgressReturn `json:"taskDetails"`
}

type TaskProgressReturn struct {
	TaskID            uuid.UUID `json:"taskId"`
	TaskTitle         string    `json:"taskTitle"`
	PlannedWeight     int       `json:"plannedWeight"`
	Deadline          time.Time `json:"deadline"`
	ActualPercentage  int       `json:"actualPercentage"`
	ClaimedPercentage int       `json:"claimedPercentage"`
}
