// internal/module/monitoring/service/service.go
package monitoring_service

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"investment_monitoring/config"
	"investment_monitoring/internal/audit"
	dashboard_cache "investment_monitoring/internal/module/support/dashboard/cache"
	"investment_monitoring/internal/outbox"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/internal/workflow"
	"investment_monitoring/pkg/errs"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

type MonitoringService struct {
	store  entity.Store
	audit  *audit.Service
	cache  *dashboard_cache.Cache
	outbox *outbox.Service
	cfg    *config.Config
}

func NewService(store entity.Store, rdb *redis.Client, cfg *config.Config) *MonitoringService {
	return &MonitoringService{
		store:  store,
		audit:  audit.NewService(store),
		cache:  dashboard_cache.New(rdb, cfg),
		outbox: outbox.NewService(store),
		cfg:    cfg,
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

	if err := s.validateTaskMutation(ctx, contract.ID, nil, params.ParentID, params.Deadline, params.Weight); err != nil {
		return result, err
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
	_ = s.cache.InvalidateProject(ctx, contract.ProjectID)

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

	// Get progress data per task
	progressData, err := s.store.GetSCurveDataByTask(ctx, contract.ID)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}
	progressMap := make(map[uuid.UUID]struct {
		Claimed int
		Actual  int
	})
	for _, pd := range progressData {
		progressMap[pd.TaskID] = struct {
			Claimed int
			Actual  int
		}{
			Claimed: int(pd.ClaimedPercentage),
			Actual:  int(pd.ActualPercentage),
		}
	}

	// Build task map with pointers
	taskMap := make(map[uuid.UUID]*TaskReturn)
	var rootTaskIDs []uuid.UUID

	for _, t := range tasks {
		children := []TaskReturn{}

		// Get progress for this task
		progress := progressMap[t.ID]

		tr := &TaskReturn{
			ID:                t.ID,
			Title:             t.Title,
			Description:       t.Description,
			Deadline:          t.Deadline,
			Weight:            int(t.Weight),
			ContractID:        t.ContractID,
			IsLeaf:            leafMap[t.ID],
			ClaimedPercentage: progress.Claimed,
			ActualPercentage:  progress.Actual,
			IsCompleted:       progress.Actual >= progress.Claimed && progress.Claimed > 0,
			CreatedAt:         t.CreatedAt,
			Children:          &children,
		}
		if t.ParentID.Valid {
			tr.ParentID = &t.ParentID.UUID
		} else {
			// Track root task IDs
			rootTaskIDs = append(rootTaskIDs, t.ID)
		}
		taskMap[t.ID] = tr
	}

	// Build hierarchy - add children to parents
	for _, t := range tasks {
		if t.ParentID.Valid {
			if parent, ok := taskMap[t.ParentID.UUID]; ok {
				child := taskMap[t.ID]
				*parent.Children = append(*parent.Children, *child)
			}
		}
	}

	// Collect root tasks AFTER hierarchy is built
	rootTasks := make([]TaskReturn, 0, len(rootTaskIDs))
	for _, id := range rootTaskIDs {
		if tr, ok := taskMap[id]; ok {
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

	// Get progress data per task
	progressData, err := s.store.GetSCurveDataByTask(ctx, contract.ID)
	if err != nil {
		return nil, errs.NewInternalServerError(err)
	}
	progressMap := make(map[uuid.UUID]struct {
		Claimed int
		Actual  int
	})
	for _, pd := range progressData {
		progressMap[pd.TaskID] = struct {
			Claimed int
			Actual  int
		}{
			Claimed: int(pd.ClaimedPercentage),
			Actual:  int(pd.ActualPercentage),
		}
	}

	result := make([]TaskReturn, 0, len(leafTasks))
	for _, t := range leafTasks {
		progress := progressMap[t.ID]
		tr := TaskReturn{
			ID:                t.ID,
			Title:             t.Title,
			Description:       t.Description,
			Deadline:          t.Deadline,
			Weight:            int(t.Weight),
			ContractID:        t.ContractID,
			IsLeaf:            true,
			ClaimedPercentage: progress.Claimed,
			ActualPercentage:  progress.Actual,
			IsCompleted:       progress.Actual >= progress.Claimed && progress.Claimed > 0,
			CreatedAt:         t.CreatedAt,
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

	if err := s.validateTaskMutation(ctx, existing.ContractID, &params.TaskID, params.ParentID, params.Deadline, params.Weight); err != nil {
		return result, err
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
	_ = s.invalidateDashboardByContractID(ctx, existing.ContractID)

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
	task, err := s.store.GetTaskByID(ctx, params.TaskID)
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
	_ = s.invalidateDashboardByContractID(ctx, task.ContractID)

	return nil
}

func (s *MonitoringService) ValidateWBS(ctx context.Context, params ValidateWBSParams) (WBSValidationReturn, error) {
	contract, err := s.store.GetContractByProjectID(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return WBSValidationReturn{}, errs.NewNotFound("CONTRACT_NOT_FOUND")
	}
	if err != nil {
		return WBSValidationReturn{}, errs.NewInternalServerError(err)
	}

	tasks, err := s.store.GetTasksByContractID(ctx, contract.ID)
	if err != nil {
		return WBSValidationReturn{}, errs.NewInternalServerError(err)
	}

	validation := validateWBSNodes(mapTaskRowsToWBSNodes(tasks))
	return WBSValidationReturn{
		IsValid:         len(validation.Errors) == 0,
		TotalRootWeight: validation.TotalRootWeight,
		TotalLeafWeight: validation.TotalLeafWeight,
		Errors:          validation.Errors,
	}, nil
}

type wbsNode struct {
	ID       uuid.UUID
	ParentID uuid.NullUUID
	Weight   int
}

type wbsValidation struct {
	TotalRootWeight int
	TotalLeafWeight int
	Errors          []string
}

func (s *MonitoringService) validateTaskMutation(ctx context.Context, contractID uuid.UUID, taskID *uuid.UUID, parentID *uuid.UUID, deadline time.Time, weight int) error {
	if parentID != nil {
		if taskID != nil && *parentID == *taskID {
			return errs.NewBadRequest("TASK_CANNOT_PARENT_ITSELF")
		}

		parent, err := s.store.GetTaskByID(ctx, *parentID)
		if err == sql.ErrNoRows {
			return errs.NewNotFound("PARENT_TASK_NOT_FOUND")
		}
		if err != nil {
			return errs.NewInternalServerError(err)
		}
		if parent.ContractID != contractID {
			return errs.NewBadRequest("PARENT_TASK_NOT_IN_CONTRACT")
		}
		if deadline.After(parent.Deadline) {
			return errs.NewBadRequest("CHILD_DEADLINE_EXCEEDS_PARENT")
		}
		if taskID != nil {
			hasCycle, err := s.wouldCreateCycle(ctx, *taskID, *parentID)
			if err != nil {
				return err
			}
			if hasCycle {
				return errs.NewBadRequest("WBS_CYCLE_DETECTED")
			}
		}
	}

	if taskID != nil {
		children, err := s.store.GetChildTasks(ctx, uuid.NullUUID{UUID: *taskID, Valid: true})
		if err != nil {
			return errs.NewInternalServerError(err)
		}
		for _, child := range children {
			if child.Deadline.After(deadline) {
				return errs.NewBadRequest("CHILD_DEADLINE_EXCEEDS_PARENT")
			}
		}
	}

	tasks, err := s.store.GetTasksByContractID(ctx, contractID)
	if err != nil {
		return errs.NewInternalServerError(err)
	}

	nodes := mapTaskRowsToWBSNodes(tasks)
	mutatedID := uuid.New()
	if taskID != nil {
		mutatedID = *taskID
	}

	parent := uuid.NullUUID{}
	if parentID != nil {
		parent = uuid.NullUUID{UUID: *parentID, Valid: true}
	}
	nodes = upsertWBSNode(nodes, wbsNode{
		ID:       mutatedID,
		ParentID: parent,
		Weight:   weight,
	})

	validation := validateWBSNodes(nodes)
	for _, code := range validation.Errors {
		switch code {
		case "TOTAL_ROOT_WEIGHT_EXCEEDS_100", "TOTAL_LEAF_WEIGHT_EXCEEDS_100", "SIBLING_WEIGHT_EXCEEDS_100", "WBS_CYCLE_DETECTED", "ORPHAN_TASK":
			return errs.NewBadRequest(code)
		}
	}
	return nil
}

func (s *MonitoringService) wouldCreateCycle(ctx context.Context, taskID, parentID uuid.UUID) (bool, error) {
	currentID := parentID
	visited := make(map[uuid.UUID]bool)

	for {
		if currentID == taskID {
			return true, nil
		}
		if visited[currentID] {
			return true, nil
		}
		visited[currentID] = true

		current, err := s.store.GetTaskByID(ctx, currentID)
		if err == sql.ErrNoRows {
			return false, errs.NewNotFound("PARENT_TASK_NOT_FOUND")
		}
		if err != nil {
			return false, errs.NewInternalServerError(err)
		}
		if !current.ParentID.Valid {
			return false, nil
		}
		currentID = current.ParentID.UUID
	}
}

func mapTaskRowsToWBSNodes(tasks []entity.TaskItem) []wbsNode {
	nodes := make([]wbsNode, 0, len(tasks))
	for _, task := range tasks {
		nodes = append(nodes, wbsNode{
			ID:       task.ID,
			ParentID: task.ParentID,
			Weight:   int(task.Weight),
		})
	}
	return nodes
}

func upsertWBSNode(nodes []wbsNode, node wbsNode) []wbsNode {
	for i := range nodes {
		if nodes[i].ID == node.ID {
			nodes[i] = node
			return nodes
		}
	}
	return append(nodes, node)
}

func validateWBSNodes(nodes []wbsNode) wbsValidation {
	validation := wbsValidation{Errors: []string{}}
	if len(nodes) == 0 {
		validation.Errors = append(validation.Errors, "WBS_EMPTY")
		return validation
	}

	nodeByID := make(map[uuid.UUID]wbsNode, len(nodes))
	childCount := make(map[uuid.UUID]int)
	siblingWeight := make(map[uuid.UUID]int)
	rootGroupID := uuid.Nil

	for _, node := range nodes {
		nodeByID[node.ID] = node
	}

	for _, node := range nodes {
		if node.ParentID.Valid {
			if _, ok := nodeByID[node.ParentID.UUID]; !ok {
				validation.Errors = appendUnique(validation.Errors, "ORPHAN_TASK")
				continue
			}
			childCount[node.ParentID.UUID]++
			siblingWeight[node.ParentID.UUID] += node.Weight
			continue
		}
		validation.TotalRootWeight += node.Weight
		siblingWeight[rootGroupID] += node.Weight
	}

	for _, node := range nodes {
		if childCount[node.ID] == 0 {
			validation.TotalLeafWeight += node.Weight
		}
	}

	if validation.TotalRootWeight != 100 {
		validation.Errors = appendUnique(validation.Errors, "TOTAL_ROOT_WEIGHT_NOT_100")
	}
	if validation.TotalRootWeight > 100 {
		validation.Errors = appendUnique(validation.Errors, "TOTAL_ROOT_WEIGHT_EXCEEDS_100")
	}
	if validation.TotalLeafWeight != 100 {
		validation.Errors = appendUnique(validation.Errors, "TOTAL_LEAF_WEIGHT_NOT_100")
	}
	if validation.TotalLeafWeight > 100 {
		validation.Errors = appendUnique(validation.Errors, "TOTAL_LEAF_WEIGHT_EXCEEDS_100")
	}
	for _, total := range siblingWeight {
		if total > 100 {
			validation.Errors = appendUnique(validation.Errors, "SIBLING_WEIGHT_EXCEEDS_100")
			break
		}
	}
	if hasWBSCycle(nodes) {
		validation.Errors = appendUnique(validation.Errors, "WBS_CYCLE_DETECTED")
	}

	return validation
}

func hasWBSCycle(nodes []wbsNode) bool {
	nodeByID := make(map[uuid.UUID]wbsNode, len(nodes))
	for _, node := range nodes {
		nodeByID[node.ID] = node
	}

	visited := make(map[uuid.UUID]bool)
	visiting := make(map[uuid.UUID]bool)

	var visit func(uuid.UUID) bool
	visit = func(id uuid.UUID) bool {
		if visiting[id] {
			return true
		}
		if visited[id] {
			return false
		}
		node, ok := nodeByID[id]
		if !ok {
			return false
		}
		visiting[id] = true
		if node.ParentID.Valid && visit(node.ParentID.UUID) {
			return true
		}
		visiting[id] = false
		visited[id] = true
		return false
	}

	for _, node := range nodes {
		if visit(node.ID) {
			return true
		}
	}
	return false
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
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

	// Check if task has pending log (not yet validated)
	hasPending, err := s.store.CheckTaskHasPendingLog(ctx, params.TaskID)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}
	if hasPending {
		return result, errs.NewBadRequest("TASK_HAS_PENDING_LOG")
	}

	var log entity.ProgressLog
	err = s.store.ExecTx(ctx, func(q *entity.Queries) error {
		var txErr error
		// Create progress log with snapshot (INSERT...SELECT in SQL)
		log, txErr = q.CreateProgressLog(ctx, entity.CreateProgressLogParams{
			Report:             params.Report,
			ProgressPercentage: int32(params.ProgressPercentage),
			EvidenceImageUrls:  params.EvidenceImageUrls,
			VendorID:           params.VendorID,
			ID:                 params.TaskID, // $5 = task_item_id for SELECT
		})
		if txErr != nil {
			return txErr
		}

		if err := s.audit.RecordTx(ctx, q, audit.RecordParams{
			ActorUserID: params.UserID,
			ActorOrgID:  params.VendorID,
			Action:      audit.ActionSubmitProgress,
			EntityType:  audit.EntityProgressLog,
			EntityID:    log.ID,
			AfterData: map[string]any{
				"progressLogId":      log.ID,
				"taskItemId":         params.TaskID,
				"taskTitle":          task.Title,
				"progressPercentage": params.ProgressPercentage,
				"vendorId":           params.VendorID,
			},
		}); err != nil {
			return err
		}

		return s.outbox.PublishTx(ctx, q, outbox.Event{
			EventType:     outbox.EventProgressSubmitted,
			AggregateType: outbox.AggregateProgressLog,
			AggregateID:   log.ID,
			Payload: map[string]any{
				"progressLogId":      log.ID,
				"taskItemId":         params.TaskID,
				"taskTitle":          task.Title,
				"progressPercentage": params.ProgressPercentage,
				"vendorId":           params.VendorID,
			},
		})
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
			// Recalculate isAccepted based on new logic: real >= claimed -> true
			claimedPct := int(l.ProgressPercentage)
			realPct := int(l.RealProgressPercentage.Int32)
			isAccepted := realPct >= claimedPct

			lr.Validation = &ValidationReturn{
				ID:                     l.ValidationID.UUID,
				IsAccepted:             isAccepted,
				ClaimedPercentage:      claimedPct,
				RealProgressPercentage: realPct,
				Comment:                comment,
				AuditorID:              l.AuditorID.UUID,
				CreatedAt:              l.ValidationCreatedAt.Time,
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
// isAccepted is calculated: real == claimed -> true, real < claimed -> false, real > claimed -> error
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

	claimedPercentage := int(log.ProgressPercentage)
	transitionInput, err := workflow.MonitoringValidationInput(claimedPercentage, params.RealProgressPercentage)
	if err != nil {
		return result, errs.NewBadRequest("AUDITOR_CANNOT_EXCEED_CLAIMED_PERCENTAGE")
	}

	nextState, ok := workflow.MonitoringNextValidationState(workflow.MonitoringStatePending, transitionInput)
	if !ok {
		return result, errs.NewBadRequest("INVALID_STATUS_TRANSITION")
	}

	isAccepted := nextState == workflow.MonitoringStateAccepted

	// If rejected, comment is required
	if !isAccepted && (params.Comment == nil || *params.Comment == "") {
		return result, errs.NewBadRequest("COMMENT_REQUIRED_FOR_REJECTION")
	}

	var commentStr sql.NullString
	if params.Comment != nil {
		commentStr = sql.NullString{String: *params.Comment, Valid: true}
	}

	var validation entity.ProgressValidation
	err = s.store.ExecTx(ctx, func(q *entity.Queries) error {
		var txErr error
		validation, txErr = q.CreateProgressValidation(ctx, entity.CreateProgressValidationParams{
			IsAccepted:             isAccepted,
			Comment:                commentStr,
			RealProgressPercentage: int32(params.RealProgressPercentage),
			ProgressLogID:          log.ID,
			AuditorID:              params.AuditorID,
		})
		if txErr != nil {
			return txErr
		}

		if err := s.audit.RecordTx(ctx, q, audit.RecordParams{
			ActorUserID: params.UserID,
			ActorOrgID:  params.AuditorID,
			Action:      audit.ActionValidateProgress,
			EntityType:  audit.EntityProgressLog,
			EntityID:    log.ID,
			BeforeData: map[string]any{
				"progressLogId":      log.ID,
				"claimedPercentage":  claimedPercentage,
				"validationState":    workflow.MonitoringStatePending,
				"progressReportDate": log.CreatedAt,
			},
			AfterData: map[string]any{
				"validationId":           validation.ID,
				"isAccepted":             validation.IsAccepted,
				"claimedPercentage":      claimedPercentage,
				"realProgressPercentage": validation.RealProgressPercentage,
				"validationState":        nextState,
				"auditorId":              params.AuditorID,
			},
		}); err != nil {
			return err
		}

		return s.outbox.PublishTx(ctx, q, outbox.Event{
			EventType:     outbox.EventProgressValidated,
			AggregateType: outbox.AggregateProgressLog,
			AggregateID:   log.ID,
			Payload: map[string]any{
				"validationId":           validation.ID,
				"progressLogId":          log.ID,
				"isAccepted":             validation.IsAccepted,
				"claimedPercentage":      claimedPercentage,
				"realProgressPercentage": validation.RealProgressPercentage,
				"validationState":        nextState,
				"auditorId":              params.AuditorID,
			},
		})
	})
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}
	_ = s.invalidateDashboardByTaskID(ctx, log.TaskItemID)

	result = ValidationReturn{
		ID:                     validation.ID,
		IsAccepted:             validation.IsAccepted,
		ClaimedPercentage:      claimedPercentage,
		RealProgressPercentage: int(validation.RealProgressPercentage),
		AuditorID:              validation.AuditorID,
		CreatedAt:              validation.CreatedAt,
	}
	if validation.Comment.Valid {
		result.Comment = &validation.Comment.String
	}

	return result, nil
}

func (s *MonitoringService) invalidateDashboardByTaskID(ctx context.Context, taskID uuid.UUID) error {
	task, err := s.store.GetTaskByID(ctx, taskID)
	if err != nil {
		return err
	}
	return s.invalidateDashboardByContractID(ctx, task.ContractID)
}

func (s *MonitoringService) invalidateDashboardByContractID(ctx context.Context, contractID uuid.UUID) error {
	contract, err := s.store.GetContractByID(ctx, contractID)
	if err != nil {
		return err
	}
	return s.cache.InvalidateProject(ctx, contract.ProjectID)
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
