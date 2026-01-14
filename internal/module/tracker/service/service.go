// internal/module/tracker/service/service.go
package tracker_service

import (
	"context"
	"database/sql"
	"fmt"

	"investment_monitoring/config"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"

	"github.com/google/uuid"
)

type TrackerService struct {
	store *entity.SQLStore
	cfg   *config.Config
}

func NewService(store *entity.SQLStore, cfg *config.Config) *TrackerService {
	return &TrackerService{
		store: store,
		cfg:   cfg,
	}
}

// ===== TICKETS =====

// CreateTicket creates a ticket and initial OPEN log atomically
func (s *TrackerService) CreateTicket(ctx context.Context, params CreateTicketParams) (TicketReturn, error) {
	var result TicketReturn
	var ticket entity.Ticket

	// Lookup ContractID from ProjectID
	contract, err := s.store.GetContractByProjectID(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return result, errs.NewNotFound("CONTRACT_NOT_FOUND")
	}
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Prepare task item ID and validate if provided
	var taskItemID uuid.NullUUID
	if params.TaskItemID != nil {
		// Validate that task item exists and belongs to this contract
		taskItem, err := s.store.GetTaskByID(ctx, *params.TaskItemID)
		if err == sql.ErrNoRows {
			return result, errs.NewBadRequest("TASK_ITEM_NOT_FOUND")
		}
		if err != nil {
			return result, errs.NewInternalServerError(err)
		}
		if taskItem.ContractID != contract.ID {
			return result, errs.NewBadRequest("TASK_ITEM_NOT_IN_CONTRACT")
		}
		taskItemID = uuid.NullUUID{UUID: *params.TaskItemID, Valid: true}
	}

	// Execute in transaction using ExecTx
	err = s.store.ExecTx(ctx, func(q *entity.Queries) error {
		var txErr error

		// Create ticket
		ticket, txErr = q.CreateTicket(ctx, entity.CreateTicketParams{
			Title:                      params.Title,
			Description:                params.Description,
			Location:                   params.Location,
			EvidenceImageUrls:          params.EvidenceImageUrls,
			TaskItemID:                 taskItemID,
			ContractID:                 contract.ID,
			TicketAssignerContractorID: params.ContractorID,
			TicketAssignerUserID:       params.UserID,
		})
		if txErr != nil {
			return txErr
		}

		// Create initial OPEN log
		_, txErr = q.CreateTicketLog(ctx, entity.CreateTicketLogParams{
			Action:         entity.TicketLogActionEnumCREATE,
			Description:    "Ticket created",
			NewStatus:      entity.TicketStatusEnumOPEN,
			TicketID:       ticket.ID,
			OrganizationID: params.ContractorID,
			UserID:         params.UserID,
		})
		return txErr
	})
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Get creator org name
	creatorOrg, _ := s.store.GetOrganizationByID(ctx, params.ContractorID)

	result = TicketReturn{
		ID:                ticket.ID,
		Title:             ticket.Title,
		Description:       ticket.Description,
		Location:          ticket.Location,
		EvidenceImageUrls: ticket.EvidenceImageUrls,
		ContractID:        ticket.ContractID,
		CurrentStatus:     string(entity.TicketStatusEnumOPEN),
		Creator: TicketActorReturn{
			OrgID:   params.ContractorID,
			OrgName: creatorOrg.Name,
		},
		CreatedAt: ticket.CreatedAt,
	}
	if ticket.TaskItemID.Valid {
		result.TaskItemID = &ticket.TaskItemID.UUID
	}

	return result, nil
}

// GetTickets retrieves all tickets for a project
func (s *TrackerService) GetTickets(ctx context.Context, params GetTicketsParams) ([]TicketReturn, error) {
	var result []TicketReturn

	// Lookup ContractID from ProjectID
	contract, err := s.store.GetContractByProjectID(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return result, errs.NewNotFound("CONTRACT_NOT_FOUND")
	}
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	var tickets []entity.GetTicketsByContractIDRow
	switch params.StatusFilter {
	case "open":
		openTickets, err := s.store.GetOpenTicketsByContractID(ctx, contract.ID)
		if err != nil {
			return result, errs.NewInternalServerError(err)
		}
		for _, t := range openTickets {
			tickets = append(tickets, entity.GetTicketsByContractIDRow{
				ID:                         t.ID,
				Title:                      t.Title,
				Description:                t.Description,
				Location:                   t.Location,
				EvidenceImageUrls:          t.EvidenceImageUrls,
				TaskItemID:                 t.TaskItemID,
				ContractID:                 t.ContractID,
				TicketAssignerContractorID: t.TicketAssignerContractorID,
				TicketAssignToID:           t.TicketAssignToID,
				CreatedAt:                  t.CreatedAt,
				CreatorOrgName:             t.CreatorOrgName,
				AssigneeOrgName:            t.AssigneeOrgName,
				TaskTitle:                  t.TaskTitle,
				CurrentStatus:              t.CurrentStatus,
				LastLogAt:                  t.LastLogAt,
			})
		}
	case "closed":
		closedTickets, err := s.store.GetClosedTicketsByContractID(ctx, contract.ID)
		if err != nil {
			return result, errs.NewInternalServerError(err)
		}
		for _, t := range closedTickets {
			tickets = append(tickets, entity.GetTicketsByContractIDRow{
				ID:                         t.ID,
				Title:                      t.Title,
				Description:                t.Description,
				Location:                   t.Location,
				EvidenceImageUrls:          t.EvidenceImageUrls,
				TaskItemID:                 t.TaskItemID,
				ContractID:                 t.ContractID,
				TicketAssignerContractorID: t.TicketAssignerContractorID,
				TicketAssignToID:           t.TicketAssignToID,
				CreatedAt:                  t.CreatedAt,
				CreatorOrgName:             t.CreatorOrgName,
				AssigneeOrgName:            t.AssigneeOrgName,
				TaskTitle:                  t.TaskTitle,
				CurrentStatus:              t.CurrentStatus,
				LastLogAt:                  t.LastLogAt,
			})
		}
	default:
		tickets, err = s.store.GetTicketsByContractID(ctx, contract.ID)
		if err != nil {
			return result, errs.NewInternalServerError(err)
		}
	}

	ticketReturns := make([]TicketReturn, 0, len(tickets))
	for _, t := range tickets {
		tr := TicketReturn{
			ID:                t.ID,
			Title:             t.Title,
			Description:       t.Description,
			Location:          t.Location,
			EvidenceImageUrls: t.EvidenceImageUrls,
			ContractID:        t.ContractID,
			CurrentStatus:     string(t.CurrentStatus),
			Creator: TicketActorReturn{
				OrgID:   t.TicketAssignerContractorID,
				OrgName: t.CreatorOrgName,
			},
			CreatedAt: t.CreatedAt,
		}
		if t.TaskItemID.Valid {
			tr.TaskItemID = &t.TaskItemID.UUID
		}
		if t.TaskTitle.Valid {
			tr.TaskTitle = &t.TaskTitle.String
		}
		if t.TicketAssignToID.Valid {
			tr.AssignedTo = &TicketActorReturn{
				OrgID:   t.TicketAssignToID.UUID,
				OrgName: t.AssigneeOrgName.String,
			}
		}
		if !t.LastLogAt.IsZero() {
			tr.LastLogAt = &t.LastLogAt
		}
		ticketReturns = append(ticketReturns, tr)
	}

	return ticketReturns, nil
}

// GetTicketByID retrieves a ticket with logs
func (s *TrackerService) GetTicketByID(ctx context.Context, params GetTicketByIDParams) (TicketReturn, error) {
	var result TicketReturn

	ticket, err := s.store.GetTicketByID(ctx, params.TicketID)
	if err == sql.ErrNoRows {
		return result, errs.NewNotFound("TICKET_NOT_FOUND")
	}
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Get logs
	logs, err := s.store.GetTicketLogsByTicketID(ctx, params.TicketID)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	logReturns := make([]TicketLogReturn, 0, len(logs))
	for _, l := range logs {
		lr := TicketLogReturn{
			ID:          l.ID,
			Action:      string(l.Action),
			Description: l.Description,
			NewStatus:   string(l.NewStatus),
			OrgID:       l.OrganizationID,
			OrgName:     l.OrgName,
			UserID:      l.UserID,
			UserName:    l.UserName,
			CreatedAt:   l.CreatedAt,
		}
		if l.AttachmentUrl.Valid {
			lr.AttachmentUrl = &l.AttachmentUrl.String
		}
		if l.PreviousStatus.Valid {
			prevStatus := string(l.PreviousStatus.TicketStatusEnum)
			lr.PreviousStatus = &prevStatus
		}
		logReturns = append(logReturns, lr)
	}

	result = TicketReturn{
		ID:                ticket.ID,
		Title:             ticket.Title,
		Description:       ticket.Description,
		Location:          ticket.Location,
		EvidenceImageUrls: ticket.EvidenceImageUrls,
		ContractID:        ticket.ContractID,
		CurrentStatus:     string(ticket.CurrentStatus),
		Creator: TicketActorReturn{
			OrgID:   ticket.TicketAssignerContractorID,
			OrgName: ticket.CreatorOrgName,
		},
		Logs:      logReturns,
		CreatedAt: ticket.CreatedAt,
	}
	if ticket.TaskItemID.Valid {
		result.TaskItemID = &ticket.TaskItemID.UUID
	}
	if ticket.TaskTitle.Valid {
		result.TaskTitle = &ticket.TaskTitle.String
	}
	if ticket.TicketAssignToID.Valid {
		result.AssignedTo = &TicketActorReturn{
			OrgID:   ticket.TicketAssignToID.UUID,
			OrgName: ticket.AssigneeOrgName.String,
		}
	}

	return result, nil
}

// AssignTicket assigns ticket to GOV_PARTNER and creates ASSIGNED log atomically
func (s *TrackerService) AssignTicket(ctx context.Context, params AssignTicketParams) (TicketReturn, error) {
	var result TicketReturn

	// Verify ticket exists and get current status
	currentStatus, err := s.store.GetLatestTicketStatus(ctx, params.TicketID)
	if err == sql.ErrNoRows {
		return result, errs.NewNotFound("TICKET_NOT_FOUND")
	}
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Check if ticket is already closed
	if currentStatus == entity.TicketStatusEnumCLOSED {
		return result, errs.NewBadRequest("TICKET_ALREADY_CLOSED")
	}

	// Verify assignee is GOV_PARTNER
	assignee, err := s.store.GetOrganizationByID(ctx, params.AssignToOrgID)
	if err == sql.ErrNoRows {
		return result, errs.NewNotFound("ASSIGNEE_ORG_NOT_FOUND")
	}
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}
	if assignee.Role != entity.OrganizationRoleEnumGOVPARTNER {
		return result, errs.NewBadRequest("ASSIGNEE_MUST_BE_GOV_PARTNER")
	}

	// Verify assignee is a GOV_PARTNER participant in this project's contract
	ticket, err := s.store.GetTicketWithContract(ctx, params.TicketID)
	if err != nil {
		return result, errs.NewNotFound("TICKET_NOT_FOUND")
	}

	// Get contract's gov participants to verify the assignee org is a GOV_PARTNER in the project
	govPartners, err := s.store.GetGovParticipantsByContractID(ctx, ticket.ContractID)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}
	isPartnerInProject := false
	for _, gp := range govPartners {
		if gp.GovParticipantID == params.AssignToOrgID {
			isPartnerInProject = true
			break
		}
	}
	if !isPartnerInProject {
		return result, errs.NewBadRequest("GOV_PARTNER_NOT_IN_PROJECT")
	}

	// Execute in transaction using ExecTx
	err = s.store.ExecTx(ctx, func(q *entity.Queries) error {
		// Update ticket
		_, txErr := q.AssignTicket(ctx, entity.AssignTicketParams{
			ID:               params.TicketID,
			TicketAssignToID: uuid.NullUUID{UUID: params.AssignToOrgID, Valid: true},
		})
		if txErr != nil {
			return txErr
		}

		// Create ASSIGNED log
		_, txErr = q.CreateTicketLog(ctx, entity.CreateTicketLogParams{
			Action:         entity.TicketLogActionEnumASSIGNED,
			Description:    fmt.Sprintf("Assigned to %s", assignee.Name),
			PreviousStatus: entity.NullTicketStatusEnum{TicketStatusEnum: currentStatus, Valid: true},
			NewStatus:      entity.TicketStatusEnumASSIGNED,
			TicketID:       params.TicketID,
			OrganizationID: params.OrgID,
			UserID:         params.UserID,
		})
		return txErr
	})
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Return updated ticket
	return s.GetTicketByID(ctx, GetTicketByIDParams{TicketID: params.TicketID})
}

// AddLog adds a comment or status change log
func (s *TrackerService) AddLog(ctx context.Context, params AddTicketLogParams) (TicketLogReturn, error) {
	var result TicketLogReturn

	// Get current status
	currentStatus, err := s.store.GetLatestTicketStatus(ctx, params.TicketID)
	if err == sql.ErrNoRows {
		return result, errs.NewNotFound("TICKET_NOT_FOUND")
	}
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Check if ticket is already closed
	if currentStatus == entity.TicketStatusEnumCLOSED {
		return result, errs.NewBadRequest("TICKET_ALREADY_CLOSED")
	}

	// Determine action and new status
	action := entity.TicketLogActionEnumCOMMENTED
	newStatus := currentStatus

	if params.NewStatus != nil {
		// Validate enum value
		validStatuses := map[string]bool{
			string(entity.TicketStatusEnumOPEN):       true,
			string(entity.TicketStatusEnumASSIGNED):   true,
			string(entity.TicketStatusEnumINPROGRESS): true,
			string(entity.TicketStatusEnumRESOLVED):   true,
			string(entity.TicketStatusEnumCLOSED):     true,
		}
		if !validStatuses[*params.NewStatus] {
			return result, errs.NewBadRequest("INVALID_STATUS_VALUE")
		}
		action = entity.TicketLogActionEnumSTATUSCHANGED
		newStatus = entity.TicketStatusEnum(*params.NewStatus)
	}

	var attachmentUrl sql.NullString
	if params.AttachmentUrl != nil {
		attachmentUrl = sql.NullString{String: *params.AttachmentUrl, Valid: true}
	}

	log, err := s.store.CreateTicketLog(ctx, entity.CreateTicketLogParams{
		Action:         action,
		Description:    params.Description,
		AttachmentUrl:  attachmentUrl,
		PreviousStatus: entity.NullTicketStatusEnum{TicketStatusEnum: currentStatus, Valid: true},
		NewStatus:      newStatus,
		TicketID:       params.TicketID,
		OrganizationID: params.OrgID,
		UserID:         params.UserID,
	})
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Get org and user names
	org, _ := s.store.GetOrganizationByID(ctx, params.OrgID)
	user, _ := s.store.GetUserByID(ctx, params.UserID)

	result = TicketLogReturn{
		ID:          log.ID,
		Action:      string(log.Action),
		Description: log.Description,
		NewStatus:   string(log.NewStatus),
		OrgID:       log.OrganizationID,
		OrgName:     org.Name,
		UserID:      log.UserID,
		UserName:    user.Name,
		CreatedAt:   log.CreatedAt,
	}
	if log.AttachmentUrl.Valid {
		result.AttachmentUrl = &log.AttachmentUrl.String
	}
	if log.PreviousStatus.Valid {
		prevStatus := string(log.PreviousStatus.TicketStatusEnum)
		result.PreviousStatus = &prevStatus
	}

	return result, nil
}

// CloseTicket closes a ticket (EXECUTING_AGENCY only)
func (s *TrackerService) CloseTicket(ctx context.Context, params CloseTicketParams) (TicketReturn, error) {
	// Get current status
	currentStatus, err := s.store.GetLatestTicketStatus(ctx, params.TicketID)
	if err == sql.ErrNoRows {
		return TicketReturn{}, errs.NewNotFound("TICKET_NOT_FOUND")
	}
	if err != nil {
		return TicketReturn{}, errs.NewInternalServerError(err)
	}

	// Check if already closed
	if currentStatus == entity.TicketStatusEnumCLOSED {
		return TicketReturn{}, errs.NewBadRequest("TICKET_ALREADY_CLOSED")
	}

	// Create CLOSED log
	_, err = s.store.CreateTicketLog(ctx, entity.CreateTicketLogParams{
		Action:         entity.TicketLogActionEnumRESOLVED,
		Description:    "Ticket closed by Project Owner",
		PreviousStatus: entity.NullTicketStatusEnum{TicketStatusEnum: currentStatus, Valid: true},
		NewStatus:      entity.TicketStatusEnumCLOSED,
		TicketID:       params.TicketID,
		OrganizationID: params.OrgID,
		UserID:         params.UserID,
	})
	if err != nil {
		return TicketReturn{}, errs.NewInternalServerError(err)
	}

	// Return updated ticket
	return s.GetTicketByID(ctx, GetTicketByIDParams{TicketID: params.TicketID})
}
