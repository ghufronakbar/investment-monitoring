package dashboard_service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"

	"github.com/google/uuid"
)

type fakeStore struct {
	entity.Store

	settings     entity.GetProjectSettingsForDashboardRow
	settingsErr  error
	tasks        []entity.GetTaskItemsWithDeadlinesRow
	tasksErr     error
	progress     []entity.GetAcceptedProgressDataRow
	progressErr  error
	disbursed    int64
	disbursedErr error
	ticketStats  entity.GetTicketStatsByProjectIDRow
	ticketErr    error
}

func (f *fakeStore) GetProjectSettingsForDashboard(ctx context.Context, projectID uuid.UUID) (entity.GetProjectSettingsForDashboardRow, error) {
	if f.settingsErr != nil {
		return entity.GetProjectSettingsForDashboardRow{}, f.settingsErr
	}
	return f.settings, nil
}

func (f *fakeStore) GetTaskItemsWithDeadlines(ctx context.Context, projectID uuid.UUID) ([]entity.GetTaskItemsWithDeadlinesRow, error) {
	if f.tasksErr != nil {
		return nil, f.tasksErr
	}
	return f.tasks, nil
}

func (f *fakeStore) GetAcceptedProgressData(ctx context.Context, projectID uuid.UUID) ([]entity.GetAcceptedProgressDataRow, error) {
	if f.progressErr != nil {
		return nil, f.progressErr
	}
	return f.progress, nil
}

func (f *fakeStore) GetDisbursedAmountByProjectID(ctx context.Context, projectID uuid.UUID) (int64, error) {
	if f.disbursedErr != nil {
		return 0, f.disbursedErr
	}
	return f.disbursed, nil
}

func (f *fakeStore) GetTicketStatsByProjectID(ctx context.Context, projectID uuid.UUID) (entity.GetTicketStatsByProjectIDRow, error) {
	if f.ticketErr != nil {
		return entity.GetTicketStatsByProjectIDRow{}, f.ticketErr
	}
	return f.ticketStats, nil
}

func TestGetSCurveGeneratesPlanAndActualDataPoints(t *testing.T) {
	projectID := uuid.New()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 14)
	taskA := uuid.New()
	taskB := uuid.New()
	store := &fakeStore{
		settings: entity.GetProjectSettingsForDashboardRow{
			ProjectID:        projectID,
			ProjectName:      "Project",
			StartDate:        start,
			BaselineDeadline: end,
			TotalBudget:      1000,
			Currency:         "IDR",
		},
		tasks: []entity.GetTaskItemsWithDeadlinesRow{
			{ID: taskA, Title: "A", Deadline: start.AddDate(0, 0, 7), Weight: 40},
			{ID: taskB, Title: "B", Deadline: end, Weight: 60},
		},
		progress: []entity.GetAcceptedProgressDataRow{
			{TaskItemID: taskA, SnapshotWeight: 40, RealProgressPercentage: 50, ValidationDate: start.AddDate(0, 0, 7)},
			{TaskItemID: taskB, SnapshotWeight: 60, RealProgressPercentage: 100, ValidationDate: end},
		},
	}

	got, err := NewService(store, nil, nil).GetSCurve(context.Background(), GetSCurveParams{ProjectID: projectID})
	if err != nil {
		t.Fatalf("GetSCurve returned error: %v", err)
	}
	if got.ProjectName != "Project" || !got.StartDate.Equal(start) || !got.BaselineDeadline.Equal(end) {
		t.Fatalf("metadata = %+v, want project settings", got)
	}
	if len(got.DataPoints) != 3 {
		t.Fatalf("data point len = %d, want 3", len(got.DataPoints))
	}
	if got.DataPoints[1].Plan != 40 || got.DataPoints[1].Actual != 20 {
		t.Fatalf("week 1 point = %+v, want plan 40 actual 20", got.DataPoints[1])
	}
	if got.DataPoints[2].Plan != 100 || got.DataPoints[2].Actual != 80 {
		t.Fatalf("final point = %+v, want plan 100 actual 80", got.DataPoints[2])
	}
}

func TestGetStatsMapsTicketStatsAndProgress(t *testing.T) {
	projectID := uuid.New()
	start := time.Now().AddDate(0, -1, 0)
	taskID := uuid.New()
	store := &fakeStore{
		settings: entity.GetProjectSettingsForDashboardRow{
			ProjectID:        projectID,
			ProjectName:      "Project",
			StartDate:        start,
			BaselineDeadline: start.AddDate(0, 1, 0),
			TotalBudget:      5000,
			Currency:         "IDR",
		},
		disbursed: 2500,
		ticketStats: entity.GetTicketStatsByProjectIDRow{
			OpenCount:       1,
			InProgressCount: 2,
			ResolvedCount:   3,
			ClosedCount:     4,
			TotalCount:      10,
		},
		tasks: []entity.GetTaskItemsWithDeadlinesRow{
			{ID: taskID, Deadline: start, Weight: 100},
		},
		progress: []entity.GetAcceptedProgressDataRow{
			{TaskItemID: taskID, SnapshotWeight: 100, RealProgressPercentage: 75, ValidationDate: time.Now().Add(-time.Hour)},
		},
	}

	got, err := NewService(store, nil, nil).GetStats(context.Background(), GetStatsParams{ProjectID: projectID})
	if err != nil {
		t.Fatalf("GetStats returned error: %v", err)
	}
	if got.ProjectName != "Project" || got.TotalBudget != "5000" || got.DisbursedAmount != "2500" || got.Currency != "IDR" {
		t.Fatalf("stats = %+v, want mapped financial fields", got)
	}
	if got.PhysicalProgress != 75 {
		t.Fatalf("physical progress = %v, want 75", got.PhysicalProgress)
	}
	if got.TicketStats.Total != 10 || got.TicketStats.Closed != 4 {
		t.Fatalf("ticket stats = %+v, want mapped stats", got.TicketStats)
	}
}

func TestGetDashboardReturnsDomainErrors(t *testing.T) {
	t.Run("s-curve missing settings", func(t *testing.T) {
		_, err := NewService(&fakeStore{settingsErr: sql.ErrNoRows}, nil, nil).GetSCurve(context.Background(), GetSCurveParams{ProjectID: uuid.New()})
		assertAppError(t, err, "PROJECT_SETTINGS_NOT_FOUND")
	})

	t.Run("stats store error", func(t *testing.T) {
		_, err := NewService(&fakeStore{settingsErr: errors.New("db down")}, nil, nil).GetStats(context.Background(), GetStatsParams{ProjectID: uuid.New()})
		assertAppError(t, err, "INTERNAL_SERVER_ERROR")
	})
}

func assertAppError(t *testing.T, err error, wantMessage string) {
	t.Helper()

	if err == nil {
		t.Fatalf("err = nil, want %s", wantMessage)
	}
	var appErr *errs.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("err = %T %v, want AppError", err, err)
	}
	if appErr.Message != wantMessage {
		t.Fatalf("message = %s, want %s", appErr.Message, wantMessage)
	}
}
