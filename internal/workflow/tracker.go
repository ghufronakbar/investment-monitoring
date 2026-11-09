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

// trackerMachine enforces the obstacle-ticket lifecycle
// OPEN -> ASSIGNED -> IN_PROGRESS -> RESOLVED -> CLOSED.
// The only backward edges are explicit: reassignment while work is not yet
// resolved, and reopening a resolved ticket when field verification fails.
// SET_OPEN and SET_ASSIGNED have no transitions; assignment goes through ASSIGN.
var trackerMachine = NewMachine([]Transition{
	{From: State(entity.TicketStatusEnumOPEN), Input: TrackerInputAssign, To: State(entity.TicketStatusEnumASSIGNED)},
	{From: State(entity.TicketStatusEnumASSIGNED), Input: TrackerInputAssign, To: State(entity.TicketStatusEnumASSIGNED)},
	{From: State(entity.TicketStatusEnumINPROGRESS), Input: TrackerInputAssign, To: State(entity.TicketStatusEnumASSIGNED)},

	{From: State(entity.TicketStatusEnumOPEN), Input: TrackerInputComment, To: State(entity.TicketStatusEnumOPEN)},
	{From: State(entity.TicketStatusEnumASSIGNED), Input: TrackerInputComment, To: State(entity.TicketStatusEnumASSIGNED)},
	{From: State(entity.TicketStatusEnumINPROGRESS), Input: TrackerInputComment, To: State(entity.TicketStatusEnumINPROGRESS)},
	{From: State(entity.TicketStatusEnumRESOLVED), Input: TrackerInputComment, To: State(entity.TicketStatusEnumRESOLVED)},

	{From: State(entity.TicketStatusEnumASSIGNED), Input: TrackerInputSetProgress, To: State(entity.TicketStatusEnumINPROGRESS)},
	{From: State(entity.TicketStatusEnumRESOLVED), Input: TrackerInputSetProgress, To: State(entity.TicketStatusEnumINPROGRESS)},

	{From: State(entity.TicketStatusEnumINPROGRESS), Input: TrackerInputSetResolved, To: State(entity.TicketStatusEnumRESOLVED)},

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
