package outbox

import (
	"context"
	"encoding/json"
	"fmt"

	websocket_service "investment_monitoring/internal/module/realtime/websocket/service"
	"investment_monitoring/internal/repository/entity"

	"github.com/google/uuid"
)

const (
	RealtimeProgressSubmitted = "progress.submitted"
	RealtimeProgressValidated = "progress.validated"
	RealtimeTicketCreated     = "ticket.created"
	RealtimeTicketAssigned    = "ticket.assigned"
	RealtimeTicketResolved    = "ticket.resolved"
	RealtimeTicketClosed      = "ticket.closed"
	RealtimePaymentApproved   = "payment.approved"
	RealtimePaymentPaid       = "payment.paid"
	RealtimeMemberInvited     = "organization.member.invited"
)

type RealtimePublisher interface {
	Publish(ctx context.Context, envelope websocket_service.Envelope) error
}

type RealtimeMapper struct {
	store entity.Store
}

func NewRealtimeMapper(store entity.Store) *RealtimeMapper {
	return &RealtimeMapper{store: store}
}

func (m *RealtimeMapper) Map(ctx context.Context, event entity.OutboxEvent) ([]websocket_service.Envelope, error) {
	switch event.EventType {
	case EventProgressSubmitted:
		return m.mapProgressEvent(ctx, event, RealtimeProgressSubmitted)
	case EventProgressValidated:
		return m.mapProgressEvent(ctx, event, RealtimeProgressValidated)
	case EventTicketCreated:
		return m.mapTicketEvent(ctx, event, RealtimeTicketCreated)
	case EventTicketAssigned:
		return m.mapTicketEvent(ctx, event, RealtimeTicketAssigned)
	case EventTicketResolved:
		return m.mapTicketEvent(ctx, event, RealtimeTicketResolved)
	case EventTicketClosed:
		return m.mapTicketEvent(ctx, event, RealtimeTicketClosed)
	case EventPaymentApproved:
		return m.mapPaymentEvent(ctx, event, RealtimePaymentApproved)
	case EventPaymentPaid:
		return m.mapPaymentEvent(ctx, event, RealtimePaymentPaid)
	case EventMemberInvited:
		return m.mapMemberEvent(event, RealtimeMemberInvited)
	default:
		return nil, nil
	}
}

func (m *RealtimeMapper) mapProgressEvent(ctx context.Context, event entity.OutboxEvent, eventType string) ([]websocket_service.Envelope, error) {
	projectID, ok := payloadUUID(event.Payload, "projectId")
	if !ok {
		if m.store == nil {
			return nil, fmt.Errorf("progress event %s missing projectId and store unavailable", event.ID)
		}
		progressLog, err := m.store.GetProgressLogWithDetails(ctx, event.AggregateID)
		if err != nil {
			return nil, err
		}
		contract, err := m.store.GetContractByID(ctx, progressLog.ContractID)
		if err != nil {
			return nil, err
		}
		projectID = contract.ProjectID
	}

	return []websocket_service.Envelope{
		newDomainEnvelope(event, eventType, fmt.Sprintf("project.%s.monitoring", projectID)),
	}, nil
}

func (m *RealtimeMapper) mapTicketEvent(ctx context.Context, event entity.OutboxEvent, eventType string) ([]websocket_service.Envelope, error) {
	projectID, ok := payloadUUID(event.Payload, "projectId")
	if !ok {
		if m.store == nil {
			return nil, fmt.Errorf("ticket event %s missing projectId and store unavailable", event.ID)
		}
		ticket, err := m.store.GetTicketWithContract(ctx, event.AggregateID)
		if err != nil {
			return nil, err
		}
		contract, err := m.store.GetContractByID(ctx, ticket.ContractID)
		if err != nil {
			return nil, err
		}
		projectID = contract.ProjectID
	}

	ticketID := event.AggregateID
	if id, ok := payloadUUID(event.Payload, "ticketId"); ok {
		ticketID = id
	}

	return []websocket_service.Envelope{
		newDomainEnvelope(event, eventType, fmt.Sprintf("project.%s.ticket", projectID)),
		newDomainEnvelope(event, eventType, fmt.Sprintf("ticket.%s", ticketID)),
	}, nil
}

func (m *RealtimeMapper) mapPaymentEvent(ctx context.Context, event entity.OutboxEvent, eventType string) ([]websocket_service.Envelope, error) {
	projectID, ok := payloadUUID(event.Payload, "projectId")
	if !ok {
		if m.store == nil {
			return nil, fmt.Errorf("payment event %s missing projectId and store unavailable", event.ID)
		}
		payment, err := m.store.GetPaymentRequestWithContract(ctx, event.AggregateID)
		if err != nil {
			return nil, err
		}
		projectID = payment.ProjectID
	}

	paymentID := event.AggregateID
	if id, ok := payloadUUID(event.Payload, "paymentRequestId"); ok {
		paymentID = id
	}

	return []websocket_service.Envelope{
		newDomainEnvelope(event, eventType, fmt.Sprintf("project.%s.finance", projectID)),
		newDomainEnvelope(event, eventType, fmt.Sprintf("payment.%s", paymentID)),
	}, nil
}

func (m *RealtimeMapper) mapMemberEvent(event entity.OutboxEvent, eventType string) ([]websocket_service.Envelope, error) {
	organizationID, hasOrganizationID := payloadUUID(event.Payload, "organizationId")
	userID, hasUserID := payloadUUID(event.Payload, "userId")
	if !hasUserID {
		userID = event.AggregateID
		hasUserID = true
	}

	envelopes := []websocket_service.Envelope{}
	if hasOrganizationID {
		envelopes = append(envelopes, newDomainEnvelope(event, eventType, fmt.Sprintf("organization.%s", organizationID)))
	}
	if hasUserID {
		envelopes = append(envelopes, newDomainEnvelope(event, eventType, fmt.Sprintf("user.%s", userID)))
	}
	return envelopes, nil
}

func newDomainEnvelope(event entity.OutboxEvent, eventType, topic string) websocket_service.Envelope {
	return websocket_service.Envelope{
		Type:  eventType,
		Topic: topic,
		Data: websocket_service.DomainEventData{
			OutboxEventID: event.ID,
			EventType:     event.EventType,
			AggregateType: event.AggregateType,
			AggregateID:   event.AggregateID,
			Payload:       event.Payload,
		},
		SentAt: event.CreatedAt,
	}
}

func payloadUUID(payload json.RawMessage, key string) (uuid.UUID, bool) {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(payload, &values); err != nil {
		return uuid.Nil, false
	}

	raw, ok := values[key]
	if !ok {
		return uuid.Nil, false
	}

	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return uuid.Nil, false
	}

	id, err := uuid.Parse(text)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}
