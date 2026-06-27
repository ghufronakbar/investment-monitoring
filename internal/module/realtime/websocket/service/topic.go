package websocket_service

import (
	"context"
	"strings"

	"investment_monitoring/internal/policy"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"

	"github.com/google/uuid"
)

const (
	TopicKindUser         = "user"
	TopicKindProject      = "project"
	TopicKindTicket       = "ticket"
	TopicKindPayment      = "payment"
	TopicKindOrganization = "organization"

	TopicProjectMonitoring = "monitoring"
	TopicProjectTicket     = "ticket"
	TopicProjectFinance    = "finance"
)

type Topic struct {
	Raw      string
	Kind     string
	ID       uuid.UUID
	Subtopic string
}

func ParseTopic(raw string) (Topic, bool) {
	raw = strings.TrimSpace(raw)
	if !isValidTopic(raw) {
		return Topic{}, false
	}

	parts := strings.Split(raw, ".")
	if len(parts) < 2 {
		return Topic{}, false
	}

	id, err := uuid.Parse(parts[1])
	if err != nil {
		return Topic{}, false
	}

	topic := Topic{
		Raw:  raw,
		Kind: parts[0],
		ID:   id,
	}

	switch topic.Kind {
	case TopicKindUser, TopicKindTicket, TopicKindPayment, TopicKindOrganization:
		if len(parts) != 2 {
			return Topic{}, false
		}
	case TopicKindProject:
		if len(parts) > 3 {
			return Topic{}, false
		}
		if len(parts) == 3 {
			switch parts[2] {
			case TopicProjectMonitoring, TopicProjectTicket, TopicProjectFinance:
				topic.Subtopic = parts[2]
			default:
				return Topic{}, false
			}
		}
	default:
		return Topic{}, false
	}

	return topic, true
}

func (s *Service) AuthorizeTopic(ctx context.Context, userID uuid.UUID, rawTopic string) error {
	topic, ok := ParseTopic(rawTopic)
	if !ok {
		return errs.NewBadRequest("INVALID_TOPIC")
	}

	switch topic.Kind {
	case TopicKindUser:
		if topic.ID != userID {
			return errs.NewForbidden("TOPIC_FORBIDDEN")
		}
		return nil
	case TopicKindProject:
		return s.authorizeProjectTopic(ctx, userID, topic)
	case TopicKindTicket:
		if s.policy == nil {
			return errs.NewForbidden("TOPIC_AUTHORIZATION_UNAVAILABLE")
		}
		return s.policy.CanViewTicket(ctx, userID, topic.ID)
	case TopicKindPayment:
		if s.policy == nil {
			return errs.NewForbidden("TOPIC_AUTHORIZATION_UNAVAILABLE")
		}
		return s.policy.CanViewPaymentRequest(ctx, userID, topic.ID)
	case TopicKindOrganization:
		if s.store == nil {
			return errs.NewForbidden("TOPIC_AUTHORIZATION_UNAVAILABLE")
		}
		ok, err := s.store.CheckMembershipExists(ctx, entity.CheckMembershipExistsParams{
			UserID:         userID,
			OrganizationID: topic.ID,
		})
		if err != nil {
			return errs.NewInternalServerError(err)
		}
		if !ok {
			return errs.NewForbidden("TOPIC_FORBIDDEN")
		}
		return nil
	default:
		return errs.NewBadRequest("INVALID_TOPIC")
	}
}

func (s *Service) authorizeProjectTopic(ctx context.Context, userID uuid.UUID, topic Topic) error {
	if s.policy == nil {
		return errs.NewForbidden("TOPIC_AUTHORIZATION_UNAVAILABLE")
	}

	switch topic.Subtopic {
	case "":
		return s.policy.CanViewProject(ctx, userID, topic.ID)
	case TopicProjectMonitoring:
		return s.policy.CanViewMonitoringProject(ctx, userID, topic.ID)
	case TopicProjectTicket:
		return s.policy.CanViewTrackerProject(ctx, userID, topic.ID)
	case TopicProjectFinance:
		return s.policy.CanViewFinanceProject(ctx, userID, topic.ID)
	default:
		return errs.NewBadRequest("INVALID_TOPIC")
	}
}

func newTopicPolicy(store entity.Store) *policy.Service {
	if store == nil {
		return nil
	}
	return policy.NewService(store)
}
