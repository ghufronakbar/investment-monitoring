// internal/module/monitoring/service/service.go
package monitoring_service

import (
	"context"
	"database/sql"
	"strconv"

	"investment_monitoring/config"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type MonitoringService struct {
	store *entity.SQLStore
	cfg   *config.Config
}

func NewService(store *entity.SQLStore, cfg *config.Config) *MonitoringService {
	return &MonitoringService{
		store: store,
		cfg:   cfg,
	}
}

// ===== TASK ITEMS (WBS) =====

// CreateTask creates a new task item in the WBS
func (s *MonitoringService) CreateTask(ctx context.Context, params CreateTaskParams) (TaskReturn, error) {
	var result TaskReturn

	// Get contract ID from project
	contract, err := s.store.GetContractByProjectID(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return result, errs.NewNotFound("CONTRACT_NOT_FOUND")
	}
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Validate parent deadline if parent exists
	if params.ParentID != nil {
		parent, err := s.store.GetTaskByID(ctx, *params.ParentID)
		if err == sql.ErrNoRows {
			return result, errs.NewNotFound("PARENT_TASK_NOT_FOUND")
		}
		if err != nil {
			return result, errs.NewInternalServerError(err)
		}
		// Child deadline cannot exceed parent deadline
		if params.Deadline.After(parent.Deadline) {
			return result, errs.NewBadRequest("CHILD_DEADLINE_EXCEEDS_PARENT")
		}
	}

	// Create task
	createParams := entity.CreateTaskParams{
		Title:       params.Title,
		Description: params.Description,
		Deadline:    params.Deadline,
		Weight:      int32(params.Weight),
		ContractID:  contract.ID,
	}
	if params.ParentID != nil {
		createParams.ParentID = uuid.NullUUID{UUID: *params.ParentID, Valid: true}
	}

	task, err := s.store.CreateTask(ctx, createParams)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	result = TaskReturn{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Deadline:    task.Deadline,
		Weight:      int(task.Weight),
		ContractID:  task.ContractID,
		IsLeaf:      true, // New task has no children
		CreatedAt:   task.CreatedAt,
	}
	if task.ParentID.Valid {
		result.ParentID = &task.ParentID.UUID
	}

	return result, nil
}

// GetTasks retrieves all tasks for a project with hierarchy
func (s *MonitoringService) GetTasks(ctx context.Context, params GetTasksParams) (TaskListReturn, error) {
	var result TaskListReturn

	// Get contract ID from project
	contract, err := s.store.GetContractByProjectID(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return result, errs.NewNotFound("CONTRACT_NOT_FOUND")
	}
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Get all tasks
	tasks, err := s.store.GetTasksByContractID(ctx, contract.ID)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Get leaf tasks for marking
	leafTasks, err := s.store.GetLeafTasksByContractID(ctx, contract.ID)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}
	leafMap := make(map[uuid.UUID]bool)
	for _, lt := range leafTasks {
		leafMap[lt.ID] = true
	}

	// Build task map
	taskMap := make(map[uuid.UUID]*TaskReturn)
	var rootTasks []TaskReturn

	for _, t := range tasks {
		tr := TaskReturn{
			ID:          t.ID,
			Title:       t.Title,
			Description: t.Description,
			Deadline:    t.Deadline,
			Weight:      int(t.Weight),
			ContractID:  t.ContractID,
			IsLeaf:      leafMap[t.ID],
			CreatedAt:   t.CreatedAt,
			Children:    []TaskReturn{},
		}
		if t.ParentID.Valid {
			tr.ParentID = &t.ParentID.UUID
		}
		taskMap[t.ID] = &tr
	}

	// Build hierarchy
	for _, t := range tasks {
		tr := taskMap[t.ID]
		if t.ParentID.Valid {
			if parent, ok := taskMap[t.ParentID.UUID]; ok {
				parent.Children = append(parent.Children, *tr)
			}
		} else {
			rootTasks = append(rootTasks, *tr)
		}
	}

	// Get total root weight
	totalWeight, err := s.store.GetTotalRootWeight(ctx, contract.ID)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	result = TaskListReturn{
		Tasks:       rootTasks,
		TotalWeight: int(totalWeight),
	}

	return result, nil
}

// GetLeafTasks retrieves only leaf tasks for dropdown
func (s *MonitoringService) GetLeafTasks(ctx context.Context, params GetTasksParams) ([]TaskReturn, error) {
	// Get contract ID from project
	contract, err := s.store.GetContractByProjectID(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return nil, errs.NewNotFound("CONTRACT_NOT_FOUND")
	}
	if err != nil {
		return nil, errs.NewInternalServerError(err)
	}

	leafTasks, err := s.store.GetLeafTasksByContractID(ctx, contract.ID)
	if err != nil {
		return nil, errs.NewInternalServerError(err)
	}

	result := make([]TaskReturn, 0, len(leafTasks))
	for _, t := range leafTasks {
		tr := TaskReturn{
			ID:          t.ID,
			Title:       t.Title,
			Description: t.Description,
			Deadline:    t.Deadline,
			Weight:      int(t.Weight),
			ContractID:  t.ContractID,
			IsLeaf:      true,
			CreatedAt:   t.CreatedAt,
		}
		if t.ParentID.Valid {
			tr.ParentID = &t.ParentID.UUID
		}
		result = append(result, tr)
	}

	return result, nil
}

// UpdateTask updates an existing task
func (s *MonitoringService) UpdateTask(ctx context.Context, params UpdateTaskParams) (TaskReturn, error) {
	var result TaskReturn

	// Get existing task
	existing, err := s.store.GetTaskByID(ctx, params.TaskID)
	if err == sql.ErrNoRows {
		return result, errs.NewNotFound("TASK_NOT_FOUND")
	}
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Validate parent deadline if parent exists
	if params.ParentID != nil {
		parent, err := s.store.GetTaskByID(ctx, *params.ParentID)
		if err == sql.ErrNoRows {
			return result, errs.NewNotFound("PARENT_TASK_NOT_FOUND")
		}
		if err != nil {
			return result, errs.NewInternalServerError(err)
		}
		if params.Deadline.After(parent.Deadline) {
			return result, errs.NewBadRequest("CHILD_DEADLINE_EXCEEDS_PARENT")
		}
	}

	updateParams := entity.UpdateTaskParams{
		ID:          params.TaskID,
		Title:       params.Title,
		Description: params.Description,
		Deadline:    params.Deadline,
		Weight:      int32(params.Weight),
	}
	if params.ParentID != nil {
		updateParams.ParentID = uuid.NullUUID{UUID: *params.ParentID, Valid: true}
	}

	task, err := s.store.UpdateTask(ctx, updateParams)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Check if leaf
	children, _ := s.store.GetChildTasks(ctx, uuid.NullUUID{UUID: task.ID, Valid: true})

	result = TaskReturn{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Deadline:    task.Deadline,
		Weight:      int(task.Weight),
		ContractID:  existing.ContractID,
		IsLeaf:      len(children) == 0,
		CreatedAt:   task.CreatedAt,
	}
	if task.ParentID.Valid {
		result.ParentID = &task.ParentID.UUID
	}

	return result, nil
}

// DeleteTask soft deletes a task
func (s *MonitoringService) DeleteTask(ctx context.Context, params DeleteTaskParams) error {
	// Check if task exists
	_, err := s.store.GetTaskByID(ctx, params.TaskID)
	if err == sql.ErrNoRows {
		return errs.NewNotFound("TASK_NOT_FOUND")
	}
	if err != nil {
		return errs.NewInternalServerError(err)
	}

	// Check if task has children
	children, err := s.store.GetChildTasks(ctx, uuid.NullUUID{UUID: params.TaskID, Valid: true})
	if err != nil {
		return errs.NewInternalServerError(err)
	}
	if len(children) > 0 {
		return errs.NewBadRequest("CANNOT_DELETE_TASK_WITH_CHILDREN")
	}

	err = s.store.DeleteTask(ctx, params.TaskID)
	if err != nil {
		return errs.NewInternalServerError(err)
	}

	return nil
}

// ===== PROGRESS LOGS =====

// SubmitProgress creates a new progress log with snapshot
func (s *MonitoringService) SubmitProgress(ctx context.Context, params SubmitProgressParams) (ProgressLogReturn, error) {
	var result ProgressLogReturn

	// Verify task exists and is a leaf
	task, err := s.store.GetTaskByID(ctx, params.TaskID)
	if err == sql.ErrNoRows {
		return result, errs.NewNotFound("TASK_NOT_FOUND")
	}
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Check if task is a leaf node
	children, err := s.store.GetChildTasks(ctx, uuid.NullUUID{UUID: params.TaskID, Valid: true})
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}
	if len(children) > 0 {
		return result, errs.NewBadRequest("CANNOT_REPORT_NON_LEAF_TASK")
	}

	// Create progress log with snapshot (INSERT...SELECT in SQL)
	log, err := s.store.CreateProgressLog(ctx, entity.CreateProgressLogParams{
		Report:             params.Report,
		ProgressPercentage: int32(params.ProgressPercentage),
		EvidenceImageUrls:  params.EvidenceImageUrls,
		VendorID:           params.VendorID,
		ID:                 params.TaskID, // $5 = task_item_id for SELECT
	})
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	result = ProgressLogReturn{
		ID:                  log.ID,
		Report:              log.Report,
		ProgressPercentage:  int(log.ProgressPercentage),
		EvidenceImageUrls:   log.EvidenceImageUrls,
		SnapshotTitle:       log.SnapshotTitle,
		SnapshotDescription: log.SnapshotDescription,
		SnapshotWeight:      int(log.SnapshotWeight),
		TaskItemID:          log.TaskItemID,
		TaskTitle:           task.Title,
		VendorID:            log.VendorID,
		CreatedAt:           log.CreatedAt,
	}

	return result, nil
}

// GetProgressLogs retrieves all progress logs for a project
func (s *MonitoringService) GetProgressLogs(ctx context.Context, params GetProgressLogsParams) ([]ProgressLogReturn, error) {
	// Get contract ID from project
	contract, err := s.store.GetContractByProjectID(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return nil, errs.NewNotFound("CONTRACT_NOT_FOUND")
	}
	if err != nil {
		return nil, errs.NewInternalServerError(err)
	}

	logs, err := s.store.GetAllProgressLogsByContractID(ctx, contract.ID)
	if err != nil {
		return nil, errs.NewInternalServerError(err)
	}

	result := make([]ProgressLogReturn, 0, len(logs))
	for _, l := range logs {
		lr := ProgressLogReturn{
			ID:                  l.ID,
			Report:              l.Report,
			ProgressPercentage:  int(l.ProgressPercentage),
			EvidenceImageUrls:   l.EvidenceImageUrls,
			SnapshotTitle:       l.SnapshotTitle,
			SnapshotDescription: l.SnapshotDescription,
			SnapshotWeight:      int(l.SnapshotWeight),
			TaskItemID:          l.TaskItemID,
			TaskTitle:           l.TaskTitle,
			VendorID:            l.VendorID,
			VendorName:          l.VendorName,
			CreatedAt:           l.CreatedAt,
		}

		if l.ValidationID.Valid {
			var comment *string
			if l.ValidationComment.Valid {
				comment = &l.ValidationComment.String
			}
			lr.Validation = &ValidationReturn{
				ID:                     l.ValidationID.UUID,
				IsAccepted:             l.IsAccepted.Bool,
				RealProgressPercentage: int(l.RealProgressPercentage.Int32),
				Comment:                comment,
			}
		}

		result = append(result, lr)
	}

	return result, nil
}

// GetPendingValidations retrieves logs that need validation
func (s *MonitoringService) GetPendingValidations(ctx context.Context, params GetProgressLogsParams) ([]ProgressLogReturn, error) {
	// Get contract ID from project
	contract, err := s.store.GetContractByProjectID(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return nil, errs.NewNotFound("CONTRACT_NOT_FOUND")
	}
	if err != nil {
		return nil, errs.NewInternalServerError(err)
	}

	logs, err := s.store.GetPendingValidationLogs(ctx, contract.ID)
	if err != nil {
		return nil, errs.NewInternalServerError(err)
	}

	result := make([]ProgressLogReturn, 0, len(logs))
	for _, l := range logs {
		lr := ProgressLogReturn{
			ID:                  l.ID,
			Report:              l.Report,
			ProgressPercentage:  int(l.ProgressPercentage),
			EvidenceImageUrls:   l.EvidenceImageUrls,
			SnapshotTitle:       l.SnapshotTitle,
			SnapshotDescription: l.SnapshotDescription,
			SnapshotWeight:      int(l.SnapshotWeight),
			TaskItemID:          l.TaskItemID,
			VendorID:            l.VendorID,
			CreatedAt:           l.CreatedAt,
		}
		result = append(result, lr)
	}

	return result, nil
}

// ===== PROGRESS VALIDATIONS =====

// ValidateProgress validates a progress log (approve/reject)
func (s *MonitoringService) ValidateProgress(ctx context.Context, params ValidateProgressParams) (ValidationReturn, error) {
	var result ValidationReturn

	// Check if log exists
	log, err := s.store.GetProgressLogByID(ctx, params.ProgressLogID)
	if err == sql.ErrNoRows {
		return result, errs.NewNotFound("PROGRESS_LOG_NOT_FOUND")
	}
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Check if already validated
	isValidated, err := s.store.CheckProgressLogValidated(ctx, params.ProgressLogID)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}
	if isValidated {
		return result, errs.NewBadRequest("PROGRESS_ALREADY_VALIDATED")
	}

	// If rejected, comment is required
	if !params.IsAccepted && (params.Comment == nil || *params.Comment == "") {
		return result, errs.NewBadRequest("COMMENT_REQUIRED_FOR_REJECTION")
	}

	var commentStr sql.NullString
	if params.Comment != nil {
		commentStr = sql.NullString{String: *params.Comment, Valid: true}
	}

	validation, err := s.store.CreateProgressValidation(ctx, entity.CreateProgressValidationParams{
		IsAccepted:             params.IsAccepted,
		Comment:                commentStr,
		RealProgressPercentage: int32(params.RealProgressPercentage),
		ProgressLogID:          log.ID,
		AuditorID:              params.AuditorID,
	})
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	result = ValidationReturn{
		ID:                     validation.ID,
		IsAccepted:             validation.IsAccepted,
		RealProgressPercentage: int(validation.RealProgressPercentage),
		AuditorID:              validation.AuditorID,
		CreatedAt:              validation.CreatedAt,
	}
	if validation.Comment.Valid {
		result.Comment = &validation.Comment.String
	}

	return result, nil
}

// ===== S-CURVE =====

// GetSCurve retrieves S-Curve data for a project
func (s *MonitoringService) GetSCurve(ctx context.Context, params GetSCurveParams) (SCurveReturn, error) {
	var result SCurveReturn

	// Get contract ID from project
	contract, err := s.store.GetContractByProjectID(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return result, errs.NewNotFound("CONTRACT_NOT_FOUND")
	}
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Get aggregated S-Curve data
	scurve, err := s.store.GetSCurveData(ctx, contract.ID)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Get per-task details
	taskDetails, err := s.store.GetSCurveDataByTask(ctx, contract.ID)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Get total root weight for planned progress
	totalWeight, err := s.store.GetTotalRootWeight(ctx, contract.ID)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	result = SCurveReturn{
		TotalSnapshotWeight: int(scurve.TotalSnapshotWeight),
		TotalLogs:           int(scurve.TotalLogs),
		ApprovedLogs:        int(scurve.ApprovedLogs),
		RejectedLogs:        int(scurve.RejectedLogs),
		TaskDetails:         make([]TaskProgressReturn, 0, len(taskDetails)),
	}

	// Parse actual progress (comes as string from DECIMAL)
	actualProgressStr := scurve.ActualProgress
	if apFloat, err := strconv.ParseFloat(actualProgressStr, 64); err == nil {
		result.ActualProgress = apFloat
	}

	// Calculate planned progress (simplified - would need timeline logic for proper S-Curve)
	if totalWeight > 0 {
		result.PlannedProgress = float64(totalWeight)
	}

	for _, td := range taskDetails {
		result.TaskDetails = append(result.TaskDetails, TaskProgressReturn{
			TaskID:            td.TaskID,
			TaskTitle:         td.TaskTitle,
			PlannedWeight:     int(td.PlannedWeight),
			Deadline:          td.Deadline,
			ActualPercentage:  int(td.ActualPercentage),
			ClaimedPercentage: int(td.ClaimedPercentage),
		})
	}

	return result, nil
}

// Ensure pq is imported for array handling
var _ = pq.Array
