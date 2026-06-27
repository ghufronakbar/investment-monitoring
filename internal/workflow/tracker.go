package workflow

import "investment_monitoring/internal/repository/entity"

const (
	TrackerInputAssign      Input = "ASSIGN"
	TrackerInputComment     Input = "COMMENT"
	TrackerInputSetOpen     Input = "SET_OPEN"
	TrackerInputSetAssigned Input = "SET_ASSIGNED"
	TrackerInputSetProgress Input = "SET_IN_PROGRESS"
	TrackerInputSetResolved Input = "SET_RESOLVED"
	TrackerInputClose       Input = "CLOSE"
)

var trackerMachine = NewMachine([]Transition{
	{From: State(entity.TicketStatusEnumOPEN), Input: TrackerInputAssign, To: State(entity.TicketStatusEnumASSIGNED)},
	{From: State(entity.TicketStatusEnumASSIGNED), Input: TrackerInputAssign, To: State(entity.TicketStatusEnumASSIGNED)},
	{From: State(entity.TicketStatusEnumINPROGRESS), Input: TrackerInputAssign, To: State(entity.TicketStatusEnumASSIGNED)},
	{From: State(entity.TicketStatusEnumRESOLVED), Input: TrackerInputAssign, To: State(entity.TicketStatusEnumASSIGNED)},

	{From: State(entity.TicketStatusEnumOPEN), Input: TrackerInputComment, To: State(entity.TicketStatusEnumOPEN)},
	{From: State(entity.TicketStatusEnumASSIGNED), Input: TrackerInputComment, To: State(entity.TicketStatusEnumASSIGNED)},
	{From: State(entity.TicketStatusEnumINPROGRESS), Input: TrackerInputComment, To: State(entity.TicketStatusEnumINPROGRESS)},
	{From: State(entity.TicketStatusEnumRESOLVED), Input: TrackerInputComment, To: State(entity.TicketStatusEnumRESOLVED)},

	{From: State(entity.TicketStatusEnumOPEN), Input: TrackerInputSetOpen, To: State(entity.TicketStatusEnumOPEN)},
	{From: State(entity.TicketStatusEnumASSIGNED), Input: TrackerInputSetOpen, To: State(entity.TicketStatusEnumOPEN)},
	{From: State(entity.TicketStatusEnumINPROGRESS), Input: TrackerInputSetOpen, To: State(entity.TicketStatusEnumOPEN)},
	{From: State(entity.TicketStatusEnumRESOLVED), Input: TrackerInputSetOpen, To: State(entity.TicketStatusEnumOPEN)},

	{From: State(entity.TicketStatusEnumOPEN), Input: TrackerInputSetAssigned, To: State(entity.TicketStatusEnumASSIGNED)},
	{From: State(entity.TicketStatusEnumASSIGNED), Input: TrackerInputSetAssigned, To: State(entity.TicketStatusEnumASSIGNED)},
	{From: State(entity.TicketStatusEnumINPROGRESS), Input: TrackerInputSetAssigned, To: State(entity.TicketStatusEnumASSIGNED)},
	{From: State(entity.TicketStatusEnumRESOLVED), Input: TrackerInputSetAssigned, To: State(entity.TicketStatusEnumASSIGNED)},

	{From: State(entity.TicketStatusEnumOPEN), Input: TrackerInputSetProgress, To: State(entity.TicketStatusEnumINPROGRESS)},
	{From: State(entity.TicketStatusEnumASSIGNED), Input: TrackerInputSetProgress, To: State(entity.TicketStatusEnumINPROGRESS)},
	{From: State(entity.TicketStatusEnumINPROGRESS), Input: TrackerInputSetProgress, To: State(entity.TicketStatusEnumINPROGRESS)},
	{From: State(entity.TicketStatusEnumRESOLVED), Input: TrackerInputSetProgress, To: State(entity.TicketStatusEnumINPROGRESS)},

	{From: State(entity.TicketStatusEnumOPEN), Input: TrackerInputSetResolved, To: State(entity.TicketStatusEnumRESOLVED)},
	{From: State(entity.TicketStatusEnumASSIGNED), Input: TrackerInputSetResolved, To: State(entity.TicketStatusEnumRESOLVED)},
	{From: State(entity.TicketStatusEnumINPROGRESS), Input: TrackerInputSetResolved, To: State(entity.TicketStatusEnumRESOLVED)},
	{From: State(entity.TicketStatusEnumRESOLVED), Input: TrackerInputSetResolved, To: State(entity.TicketStatusEnumRESOLVED)},

	{From: State(entity.TicketStatusEnumOPEN), Input: TrackerInputClose, To: State(entity.TicketStatusEnumCLOSED)},
	{From: State(entity.TicketStatusEnumASSIGNED), Input: TrackerInputClose, To: State(entity.TicketStatusEnumCLOSED)},
	{From: State(entity.TicketStatusEnumINPROGRESS), Input: TrackerInputClose, To: State(entity.TicketStatusEnumCLOSED)},
	{From: State(entity.TicketStatusEnumRESOLVED), Input: TrackerInputClose, To: State(entity.TicketStatusEnumCLOSED)},
})

func TrackerNextStatus(current entity.TicketStatusEnum, input Input) (entity.TicketStatusEnum, bool) {
	next, ok := trackerMachine.Next(State(current), input)
	if !ok {
		return "", false
	}
	return entity.TicketStatusEnum(next), true
}

func TrackerInputForStatus(status entity.TicketStatusEnum) (Input, bool) {
	switch status {
	case entity.TicketStatusEnumOPEN:
		return TrackerInputSetOpen, true
	case entity.TicketStatusEnumASSIGNED:
		return TrackerInputSetAssigned, true
	case entity.TicketStatusEnumINPROGRESS:
		return TrackerInputSetProgress, true
	case entity.TicketStatusEnumRESOLVED:
		return TrackerInputSetResolved, true
	case entity.TicketStatusEnumCLOSED:
		return TrackerInputClose, true
	default:
		return "", false
	}
}
