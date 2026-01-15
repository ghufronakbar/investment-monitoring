// internal/module/support/dashboard/service/service.go
package dashboard_service

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"investment_monitoring/config"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"

	"github.com/google/uuid"
)

type DashboardService struct {
	store *entity.SQLStore
	cfg   *config.Config
}

func NewService(store *entity.SQLStore, cfg *config.Config) *DashboardService {
	return &DashboardService{
		store: store,
		cfg:   cfg,
	}
}

// GetSCurve calculates S-Curve data points weekly from start_date to baseline_deadline
func (s *DashboardService) GetSCurve(ctx context.Context, params GetSCurveParams) (*SCurveReturn, error) {
	// Get project settings
	settings, err := s.store.GetProjectSettingsForDashboard(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return nil, errs.NewNotFound("PROJECT_SETTINGS_NOT_FOUND")
	}
	if err != nil {
		return nil, errs.NewInternalServerError(err)
	}

	// Get task items for plan calculation
	tasks, err := s.store.GetTaskItemsWithDeadlines(ctx, params.ProjectID)
	if err != nil {
		return nil, errs.NewInternalServerError(err)
	}

	// Get accepted progress data for actual calculation
	progressData, err := s.store.GetAcceptedProgressData(ctx, params.ProjectID)
	if err != nil {
		return nil, errs.NewInternalServerError(err)
	}

	// Calculate total weight (all leaf nodes)
	leafTasks := filterLeafTasks(tasks)
	totalWeight := 0
	for _, task := range leafTasks {
		totalWeight += int(task.Weight)
	}

	// Generate weekly data points
	dataPoints := generateWeeklyDataPoints(
		settings.StartDate,
		settings.BaselineDeadline,
		leafTasks,
		progressData,
		totalWeight,
	)

	return &SCurveReturn{
		ProjectName:      settings.ProjectName,
		StartDate:        settings.StartDate,
		BaselineDeadline: settings.BaselineDeadline,
		DataPoints:       dataPoints,
	}, nil
}

// GetStats returns dashboard statistics
func (s *DashboardService) GetStats(ctx context.Context, params GetStatsParams) (*StatsReturn, error) {
	// Get project settings
	settings, err := s.store.GetProjectSettingsForDashboard(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return nil, errs.NewNotFound("PROJECT_SETTINGS_NOT_FOUND")
	}
	if err != nil {
		return nil, errs.NewInternalServerError(err)
	}

	// Get disbursed amount
	disbursed, err := s.store.GetDisbursedAmountByProjectID(ctx, params.ProjectID)
	if err != nil {
		return nil, errs.NewInternalServerError(err)
	}

	// Get ticket stats
	ticketStats, err := s.store.GetTicketStatsByProjectID(ctx, params.ProjectID)
	if err != nil {
		return nil, errs.NewInternalServerError(err)
	}

	// Calculate current physical progress
	physicalProgress := s.calculateCurrentProgress(ctx, params.ProjectID)

	return &StatsReturn{
		ProjectName:      settings.ProjectName,
		TotalBudget:      strconv.FormatInt(settings.TotalBudget, 10),
		DisbursedAmount:  strconv.FormatInt(disbursed, 10),
		Currency:         settings.Currency,
		PhysicalProgress: physicalProgress,
		TicketStats: TicketStats{
			Open:       ticketStats.OpenCount,
			InProgress: ticketStats.InProgressCount,
			Resolved:   ticketStats.ResolvedCount,
			Closed:     ticketStats.ClosedCount,
			Total:      ticketStats.TotalCount,
		},
	}, nil
}

// ===== HELPER FUNCTIONS =====

// filterLeafTasks returns only leaf task items (no children)
func filterLeafTasks(tasks []entity.GetTaskItemsWithDeadlinesRow) []entity.GetTaskItemsWithDeadlinesRow {
	// Build set of parent IDs
	parentIDs := make(map[uuid.UUID]bool)
	for _, task := range tasks {
		if task.ParentID.Valid {
			parentIDs[task.ParentID.UUID] = true
		}
	}

	// Filter: task is leaf if it's not a parent of any other task
	var leafTasks []entity.GetTaskItemsWithDeadlinesRow
	for _, task := range tasks {
		if !parentIDs[task.ID] {
			leafTasks = append(leafTasks, task)
		}
	}
	return leafTasks
}

// generateWeeklyDataPoints creates S-Curve data points per week
func generateWeeklyDataPoints(
	startDate, endDate time.Time,
	tasks []entity.GetTaskItemsWithDeadlinesRow,
	progressData []entity.GetAcceptedProgressDataRow,
	totalWeight int,
) []SCurveDataPoint {
	var dataPoints []SCurveDataPoint

	if totalWeight == 0 {
		return dataPoints
	}

	// Iterate weekly
	current := startDate
	for !current.After(endDate) {
		plan := calculatePlanAtDate(current, tasks, totalWeight)
		actual := calculateActualAtDate(current, progressData, totalWeight)

		dataPoints = append(dataPoints, SCurveDataPoint{
			Date:   current.Format("2006-01-02"),
			Plan:   plan,
			Actual: actual,
		})

		current = current.AddDate(0, 0, 7) // Next week
	}

	// Add final point at end date if not already included
	if len(dataPoints) > 0 {
		lastPoint := dataPoints[len(dataPoints)-1]
		if lastPoint.Date != endDate.Format("2006-01-02") {
			plan := calculatePlanAtDate(endDate, tasks, totalWeight)
			actual := calculateActualAtDate(endDate, progressData, totalWeight)
			dataPoints = append(dataPoints, SCurveDataPoint{
				Date:   endDate.Format("2006-01-02"),
				Plan:   plan,
				Actual: actual,
			})
		}
	}

	return dataPoints
}

// calculatePlanAtDate returns cumulative planned progress at a given date
func calculatePlanAtDate(date time.Time, tasks []entity.GetTaskItemsWithDeadlinesRow, totalWeight int) float64 {
	if totalWeight == 0 {
		return 0
	}

	cumulativeWeight := 0
	for _, task := range tasks {
		// If task deadline is on or before this date, count its weight
		if !task.Deadline.After(date) {
			cumulativeWeight += int(task.Weight)
		}
	}

	return float64(cumulativeWeight) / float64(totalWeight) * 100
}

// calculateActualAtDate returns cumulative actual progress at a given date
func calculateActualAtDate(date time.Time, progressData []entity.GetAcceptedProgressDataRow, totalWeight int) float64 {
	if totalWeight == 0 {
		return 0
	}

	// Track latest progress per task
	taskProgress := make(map[uuid.UUID]float64)

	for _, pd := range progressData {
		// Only count if validation happened on or before this date
		if !pd.ValidationDate.After(date) {
			// Calculate contribution: (snapshot_weight * real_progress_percentage / 100)
			contribution := float64(pd.SnapshotWeight) * float64(pd.RealProgressPercentage) / 100
			taskProgress[pd.TaskItemID] = contribution
		}
	}

	// Sum all contributions
	var totalContribution float64
	for _, contribution := range taskProgress {
		totalContribution += contribution
	}

	return totalContribution / float64(totalWeight) * 100
}

// calculateCurrentProgress returns the latest actual progress percentage
func (s *DashboardService) calculateCurrentProgress(ctx context.Context, projectID uuid.UUID) float64 {
	// Get task items
	tasks, err := s.store.GetTaskItemsWithDeadlines(ctx, projectID)
	if err != nil {
		return 0
	}

	// Get progress data
	progressData, err := s.store.GetAcceptedProgressData(ctx, projectID)
	if err != nil {
		return 0
	}

	leafTasks := filterLeafTasks(tasks)
	totalWeight := 0
	for _, task := range leafTasks {
		totalWeight += int(task.Weight)
	}

	if totalWeight == 0 {
		return 0
	}

	// Calculate current (as of now)
	return calculateActualAtDate(time.Now(), progressData, totalWeight)
}
