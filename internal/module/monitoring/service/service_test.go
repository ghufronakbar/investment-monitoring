package monitoring_service

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

	contract          entity.Contract
	contractErr       error
	tasks             []entity.TaskItem
	tasksErr          error
	leafTasks         []entity.TaskItem
	leafTasksErr      error
	progressByTask    []entity.GetSCurveDataByTaskRow
	progressByTaskErr error
	task              entity.TaskItem
	taskErr           error
	childTasks        []entity.TaskItem
	childTasksErr     error
	deleteErr         error
}

func (f *fakeStore) GetContractByProjectID(ctx context.Context, projectID uuid.UUID) (entity.Contract, error) {
	if f.contractErr != nil {
		return entity.Contract{}, f.contractErr
	}
	return f.contract, nil
}

func (f *fakeStore) GetTasksByContractID(ctx context.Context, contractID uuid.UUID) ([]entity.TaskItem, error) {
	if f.tasksErr != nil {
		return nil, f.tasksErr
	}
	return f.tasks, nil
}

func (f *fakeStore) GetLeafTasksByContractID(ctx context.Context, contractID uuid.UUID) ([]entity.TaskItem, error) {
	if f.leafTasksErr != nil {
		return nil, f.leafTasksErr
	}
	return f.leafTasks, nil
}

func (f *fakeStore) GetSCurveDataByTask(ctx context.Context, contractID uuid.UUID) ([]entity.GetSCurveDataByTaskRow, error) {
	if f.progressByTaskErr != nil {
		return nil, f.progressByTaskErr
	}
	return f.progressByTask, nil
}

func (f *fakeStore) GetTaskByID(ctx context.Context, id uuid.UUID) (entity.TaskItem, error) {
	if f.taskErr != nil {
		return entity.TaskItem{}, f.taskErr
	}
	return f.task, nil
}

func (f *fakeStore) GetChildTasks(ctx context.Context, parentID uuid.NullUUID) ([]entity.TaskItem, error) {
	if f.childTasksErr != nil {
		return nil, f.childTasksErr
	}
	return f.childTasks, nil
}

func (f *fakeStore) DeleteTask(ctx context.Context, id uuid.UUID) error {
	return f.deleteErr
}

func TestValidateWBSSurfacesIntegrityErrors(t *testing.T) {
	projectID := uuid.New()
	contractID := uuid.New()
	rootA := uuid.New()
	rootB := uuid.New()
	store := &fakeStore{
		contract: entity.Contract{ID: contractID, ProjectID: projectID},
		tasks: []entity.TaskItem{
			{ID: rootA, ContractID: contractID, Weight: 60},
			{ID: rootB, ContractID: contractID, Weight: 50},
		},
	}

	got, err := NewService(store, nil, nil).ValidateWBS(context.Background(), ValidateWBSParams{ProjectID: projectID})
	if err != nil {
		t.Fatalf("ValidateWBS returned error: %v", err)
	}
	if got.IsValid {
		t.Fatalf("IsValid = true, want false")
	}
	if got.TotalRootWeight != 110 || got.TotalLeafWeight != 110 {
		t.Fatalf("weights = root %d leaf %d, want 110/110", got.TotalRootWeight, got.TotalLeafWeight)
	}
	if !contains(got.Errors, "TOTAL_ROOT_WEIGHT_EXCEEDS_100") {
		t.Fatalf("errors = %#v, want root weight error", got.Errors)
	}
}

func TestGetLeafTasksMapsProgressAndCompletion(t *testing.T) {
	projectID := uuid.New()
	contractID := uuid.New()
	taskID := uuid.New()
	parentID := uuid.New()
	deadline := time.Now().AddDate(0, 0, 7)
	store := &fakeStore{
		contract: entity.Contract{ID: contractID, ProjectID: projectID},
		leafTasks: []entity.TaskItem{{
			ID:          taskID,
			Title:       "Task",
			Description: "Desc",
			Deadline:    deadline,
			Weight:      30,
			ParentID:    uuid.NullUUID{UUID: parentID, Valid: true},
			ContractID:  contractID,
			CreatedAt:   deadline.AddDate(0, 0, -7),
		}},
		progressByTask: []entity.GetSCurveDataByTaskRow{{
			TaskID:            taskID,
			ClaimedPercentage: 50,
			ActualPercentage:  50,
		}},
	}

	got, err := NewService(store, nil, nil).GetLeafTasks(context.Background(), GetTasksParams{ProjectID: projectID})
	if err != nil {
		t.Fatalf("GetLeafTasks returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	task := got[0]
	if task.ID != taskID || task.Weight != 30 || task.ParentID == nil || *task.ParentID != parentID {
		t.Fatalf("task = %+v, want mapped leaf task", task)
	}
	if !task.IsLeaf || !task.IsCompleted || task.ClaimedPercentage != 50 || task.ActualPercentage != 50 {
		t.Fatalf("progress flags = %+v, want completed leaf with 50/50 progress", task)
	}
}

func TestDeleteTaskRejectsTaskWithChildren(t *testing.T) {
	taskID := uuid.New()
	contractID := uuid.New()
	store := &fakeStore{
		task:       entity.TaskItem{ID: taskID, ContractID: contractID},
		childTasks: []entity.TaskItem{{ID: uuid.New(), ContractID: contractID}},
	}

	err := NewService(store, nil, nil).DeleteTask(context.Background(), DeleteTaskParams{TaskID: taskID})
	testutil.AssertAppError(t, err, "CANNOT_DELETE_TASK_WITH_CHILDREN")
}

func TestValidateTaskMutationRejectsParentErrors(t *testing.T) {
	contractID := uuid.New()
	taskID := uuid.New()
	parentID := uuid.New()
	deadline := time.Now()

	t.Run("task cannot parent itself", func(t *testing.T) {
		err := NewService(&fakeStore{}, nil, nil).validateTaskMutation(context.Background(), contractID, &taskID, &taskID, deadline, 10)
		testutil.AssertAppError(t, err, "TASK_CANNOT_PARENT_ITSELF")
	})

	t.Run("missing parent", func(t *testing.T) {
		store := &fakeStore{taskErr: sql.ErrNoRows}
		err := NewService(store, nil, nil).validateTaskMutation(context.Background(), contractID, &taskID, &parentID, deadline, 10)
		testutil.AssertAppError(t, err, "PARENT_TASK_NOT_FOUND")
	})

	t.Run("parent must be in same contract", func(t *testing.T) {
		store := &fakeStore{task: entity.TaskItem{ID: parentID, ContractID: uuid.New(), Deadline: deadline.Add(time.Hour)}}
		err := NewService(store, nil, nil).validateTaskMutation(context.Background(), contractID, &taskID, &parentID, deadline, 10)
		testutil.AssertAppError(t, err, "PARENT_TASK_NOT_IN_CONTRACT")
	})

	t.Run("child deadline cannot exceed parent", func(t *testing.T) {
		store := &fakeStore{task: entity.TaskItem{ID: parentID, ContractID: contractID, Deadline: deadline.Add(-time.Hour)}}
		err := NewService(store, nil, nil).validateTaskMutation(context.Background(), contractID, &taskID, &parentID, deadline, 10)
		testutil.AssertAppError(t, err, "CHILD_DEADLINE_EXCEEDS_PARENT")
	})
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
