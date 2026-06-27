package outbox

import "github.com/google/uuid"

const (
	EventProgressSubmitted = "ProgressSubmitted"
	EventProgressValidated = "ProgressValidated"
	EventTicketCreated     = "TicketCreated"
	EventTicketAssigned    = "TicketAssigned"
	EventTicketResolved    = "TicketResolved"
	EventTicketClosed      = "TicketClosed"
	EventPaymentApproved   = "PaymentApproved"
	EventPaymentPaid       = "PaymentPaid"
	EventMemberInvited     = "MemberInvited"
)

const (
	AggregateProgressLog    = "progress_log"
	AggregateTicket         = "ticket"
	AggregatePaymentRequest = "payment_request"
	AggregateUser           = "user"
)

type Event struct {
	EventType     string
	AggregateType string
	AggregateID   uuid.UUID
	Payload       any
}
