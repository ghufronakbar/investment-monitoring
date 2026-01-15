// internal/module/support/dashboard/service/return.go
package dashboard_service

import "time"

// ===== S-CURVE =====

type SCurveDataPoint struct {
	Date   string  `json:"date"`   // YYYY-MM-DD format
	Plan   float64 `json:"plan"`   // Cumulative planned progress %
	Actual float64 `json:"actual"` // Cumulative actual progress %
}

type SCurveReturn struct {
	ProjectName      string            `json:"projectName"`
	StartDate        time.Time         `json:"startDate"`
	BaselineDeadline time.Time         `json:"baselineDeadline"`
	DataPoints       []SCurveDataPoint `json:"dataPoints"`
}

// ===== STATS =====

type StatsReturn struct {
	ProjectName      string      `json:"projectName"`
	TotalBudget      string      `json:"totalBudget"`     // String for BigInt
	DisbursedAmount  string      `json:"disbursedAmount"` // String for BigInt
	Currency         string      `json:"currency"`
	PhysicalProgress float64     `json:"physicalProgress"` // Current actual progress %
	TicketStats      TicketStats `json:"ticketStats"`
}

type TicketStats struct {
	Open       int64 `json:"open"`
	InProgress int64 `json:"inProgress"`
	Resolved   int64 `json:"resolved"`
	Closed     int64 `json:"closed"`
	Total      int64 `json:"total"`
}
