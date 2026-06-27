package workflow

import "investment_monitoring/internal/repository/entity"

const (
	FinanceInputAuditorApprove Input = "AUDITOR_APPROVE"
	FinanceInputAuditorReject  Input = "AUDITOR_REJECT"
	FinanceInputAgencyApprove  Input = "AGENCY_APPROVE"
	FinanceInputAgencyReject   Input = "AGENCY_REJECT"
	FinanceInputDisburse       Input = "DISBURSE"
)

var financeMachine = NewMachine([]Transition{
	{
		From:  State(entity.PaymentRequestStatusEnumSUBMITTED),
		Input: FinanceInputAuditorApprove,
		To:    State(entity.PaymentRequestStatusEnumAUDITORAPPROVED),
	},
	{
		From:  State(entity.PaymentRequestStatusEnumSUBMITTED),
		Input: FinanceInputAuditorReject,
		To:    State(entity.PaymentRequestStatusEnumAUDITORREJECTED),
	},
	{
		From:  State(entity.PaymentRequestStatusEnumAUDITORAPPROVED),
		Input: FinanceInputAgencyApprove,
		To:    State(entity.PaymentRequestStatusEnumAGENCYAPPROVED),
	},
	{
		From:  State(entity.PaymentRequestStatusEnumAUDITORAPPROVED),
		Input: FinanceInputAgencyReject,
		To:    State(entity.PaymentRequestStatusEnumAGENCYREJECTED),
	},
	{
		From:  State(entity.PaymentRequestStatusEnumAGENCYAPPROVED),
		Input: FinanceInputDisburse,
		To:    State(entity.PaymentRequestStatusEnumPAID),
	},
})

func FinanceNextStatus(current entity.PaymentRequestStatusEnum, input Input) (entity.PaymentRequestStatusEnum, bool) {
	next, ok := financeMachine.Next(State(current), input)
	if !ok {
		return "", false
	}
	return entity.PaymentRequestStatusEnum(next), true
}

func FinanceApprovalInput(action string, auditorStage bool) Input {
	if auditorStage {
		if action == "APPROVE" {
			return FinanceInputAuditorApprove
		}
		return FinanceInputAuditorReject
	}
	if action == "APPROVE" {
		return FinanceInputAgencyApprove
	}
	return FinanceInputAgencyReject
}
