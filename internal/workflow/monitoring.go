package workflow

const (
	MonitoringStatePending  State = "PENDING_VALIDATION"
	MonitoringStateAccepted State = "ACCEPTED"
	MonitoringStateRejected State = "REJECTED"

	MonitoringInputAccept Input = "ACCEPT"
	MonitoringInputReject Input = "REJECT"
)

var monitoringMachine = NewMachine([]Transition{
	{From: MonitoringStatePending, Input: MonitoringInputAccept, To: MonitoringStateAccepted},
	{From: MonitoringStatePending, Input: MonitoringInputReject, To: MonitoringStateRejected},
})

func MonitoringValidationState(isAlreadyValidated bool, isAccepted bool) State {
	if !isAlreadyValidated {
		return MonitoringStatePending
	}
	if isAccepted {
		return MonitoringStateAccepted
	}
	return MonitoringStateRejected
}

func MonitoringValidationInput(claimedPercentage int, realPercentage int) (Input, error) {
	if realPercentage > claimedPercentage {
		return "", ErrInvalidValidationPercentage
	}
	if realPercentage == claimedPercentage {
		return MonitoringInputAccept, nil
	}
	return MonitoringInputReject, nil
}

func MonitoringNextValidationState(current State, input Input) (State, bool) {
	return monitoringMachine.Next(current, input)
}
