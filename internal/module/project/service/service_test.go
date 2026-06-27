package project_service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/internal/testutil"

	"github.com/google/uuid"
)

type fakeStore struct {
	entity.Store

	projects        []entity.Project
	listProjectsErr error
	project         entity.Project
	projectErr      error
	contract        entity.Contract
	contractErr     error
	settings        entity.ProjectSetting
	settingsErr     error
	organization    entity.Organization
	organizationErr error
}

func (f *fakeStore) ListUserAccessibleProjects(ctx context.Context, userID uuid.UUID) ([]entity.Project, error) {
	if f.listProjectsErr != nil {
		return nil, f.listProjectsErr
	}
	return f.projects, nil
}

func (f *fakeStore) GetProjectByID(ctx context.Context, id uuid.UUID) (entity.Project, error) {
	if f.projectErr != nil {
		return entity.Project{}, f.projectErr
	}
	return f.project, nil
}

func (f *fakeStore) GetContractByProjectID(ctx context.Context, projectID uuid.UUID) (entity.Contract, error) {
	if f.contractErr != nil {
		return entity.Contract{}, f.contractErr
	}
	return f.contract, nil
}

func (f *fakeStore) GetProjectSettingsByProjectID(ctx context.Context, projectID uuid.UUID) (entity.ProjectSetting, error) {
	if f.settingsErr != nil {
		return entity.ProjectSetting{}, f.settingsErr
	}
	return f.settings, nil
}

func (f *fakeStore) GetOrganizationByID(ctx context.Context, id uuid.UUID) (entity.Organization, error) {
	if f.organizationErr != nil {
		return entity.Organization{}, f.organizationErr
	}
	return f.organization, nil
}

func TestListAccessibleMapsProjects(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()
	createdAt := time.Now()
	store := &fakeStore{
		projects: []entity.Project{{
			ID:              projectID,
			Name:            "Project",
			Description:     "Description",
			ContractFileUrl: "https://example.com/contract.pdf",
			ImageUrls:       []string{"https://example.com/image.png"},
			CreatedAt:       createdAt,
		}},
	}

	got, err := NewService(store, nil, nil).ListAccessible(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListAccessible returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].ID != projectID || got[0].Name != "Project" || !got[0].CreatedAt.Equal(createdAt) {
		t.Fatalf("project = %+v, want mapped project", got[0])
	}
}

func TestConfigureSettingsValidatesBeforeStoreMutation(t *testing.T) {
	start := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	deadline := start.AddDate(0, 0, -1)

	_, err := NewService(&fakeStore{}, nil, nil).ConfigureSettings(context.Background(), ConfigureSettingsParams{
		ProjectID:        uuid.New(),
		StartDate:        start,
		BaselineDeadline: deadline,
		TotalBudget:      1000,
		Currency:         "IDR",
	})
	testutil.AssertAppError(t, err, "BASELINE_DEADLINE_BEFORE_START_DATE")
}

func TestAssignVendorPreTransactionValidation(t *testing.T) {
	projectID := uuid.New()
	contractID := uuid.New()
	vendorID := uuid.New()
	params := AssignVendorParams{
		ProjectID:      projectID,
		VendorOrgID:    vendorID,
		ContractValue:  1000,
		ContractDocUrl: "https://example.com/contract.pdf",
	}

	t.Run("requires project settings", func(t *testing.T) {
		store := &fakeStore{
			contract:    entity.Contract{ID: contractID, ProjectID: projectID},
			settingsErr: sql.ErrNoRows,
		}

		_, err := NewService(store, nil, nil).AssignVendor(context.Background(), params)
		testutil.AssertAppError(t, err, "PROJECT_SETTINGS_REQUIRED")
	})

	t.Run("contract value must be below total budget", func(t *testing.T) {
		store := &fakeStore{
			contract: entity.Contract{ID: contractID, ProjectID: projectID},
			settings: entity.ProjectSetting{ProjectID: projectID, TotalBudget: 1000},
		}

		_, err := NewService(store, nil, nil).AssignVendor(context.Background(), params)
		testutil.AssertAppError(t, err, "CONTRACT_VALUE_EXCEEDS_BUDGET")
	})

	t.Run("organization must be contractor", func(t *testing.T) {
		store := &fakeStore{
			contract:     entity.Contract{ID: contractID, ProjectID: projectID},
			settings:     entity.ProjectSetting{ProjectID: projectID, TotalBudget: 2000},
			organization: entity.Organization{ID: vendorID, Role: entity.OrganizationRoleEnumEXECUTINGAGENCY},
		}

		_, err := NewService(store, nil, nil).AssignVendor(context.Background(), params)
		testutil.AssertAppError(t, err, "ORGANIZATION_NOT_CONTRACTOR")
	})
}
