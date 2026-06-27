package audit

import "github.com/google/uuid"

const (
	ActionCreateProject          = "CREATE_PROJECT"
	ActionConfigureProject       = "CONFIGURE_PROJECT"
	ActionAssignExecutingAgency  = "ASSIGN_EXECUTING_AGENCY"
	ActionAssignVendor           = "ASSIGN_VENDOR"
	ActionAssignAuditor          = "ASSIGN_AUDITOR"
	ActionAssignGovPartners      = "ASSIGN_GOV_PARTNERS"
	ActionSubmitProgress         = "SUBMIT_PROGRESS"
	ActionValidateProgress       = "VALIDATE_PROGRESS"
	ActionCreateTicket           = "CREATE_TICKET"
	ActionAssignTicket           = "ASSIGN_TICKET"
	ActionUpdateTicketStatus     = "UPDATE_TICKET_STATUS"
	ActionCloseTicket            = "CLOSE_TICKET"
	ActionCreatePaymentRequest   = "CREATE_PAYMENT_REQUEST"
	ActionApprovePaymentAuditor  = "APPROVE_PAYMENT_AUDITOR"
	ActionApprovePaymentAgency   = "APPROVE_PAYMENT_AGENCY"
	ActionDisbursePayment        = "DISBURSE_PAYMENT"
	ActionCreateOrganization     = "CREATE_ORGANIZATION"
	ActionInviteOrganizationUser = "INVITE_ORGANIZATION_USER"
)

const (
	EntityProject        = "project"
	EntityContract       = "contract"
	EntityProgressLog    = "progress_log"
	EntityTicket         = "ticket"
	EntityPaymentRequest = "payment_request"
	EntityOrganization   = "organization"
	EntityUser           = "user"
)

type RecordParams struct {
	ActorUserID uuid.UUID
	ActorOrgID  uuid.UUID
	Action      string
	EntityType  string
	EntityID    uuid.UUID
	BeforeData  any
	AfterData   any
}

type RequestMetadata struct {
	RequestID string
	IPAddress string
	UserAgent string
}
