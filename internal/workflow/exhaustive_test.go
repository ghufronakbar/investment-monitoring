package workflow

import (
	"testing"

	"investment_monitoring/internal/repository/entity"
)

// machineSpec is the transition matrix as specified in the governance design,
// written independently of the machine definitions so the exhaustive test can
// catch both missing and extra transitions.
type machineSpec struct {
	name    string
	machine Machine
	states  []State
	inputs  []Input
	allowed map[State]map[Input]State
}

func fsmSpecs() []machineSpec {
	ticket := func(s entity.TicketStatusEnum) State { return State(s) }
	payment := func(s entity.PaymentRequestStatusEnum) State { return State(s) }

	return []machineSpec{
		{
			name:    "monitoring",
			machine: monitoringMachine,
			states:  []State{MonitoringStatePending, MonitoringStateAccepted, MonitoringStateRejected},
			inputs:  []Input{MonitoringInputAccept, MonitoringInputReject},
			allowed: map[State]map[Input]State{
				MonitoringStatePending: {
					MonitoringInputAccept: MonitoringStateAccepted,
					MonitoringInputReject: MonitoringStateRejected,
				},
			},
		},
		{
			name:    "finance",
			machine: financeMachine,
			states: []State{
				payment(entity.PaymentRequestStatusEnumSUBMITTED),
				payment(entity.PaymentRequestStatusEnumAUDITORAPPROVED),
				payment(entity.PaymentRequestStatusEnumAUDITORREJECTED),
				payment(entity.PaymentRequestStatusEnumAGENCYAPPROVED),
				payment(entity.PaymentRequestStatusEnumAGENCYREJECTED),
				payment(entity.PaymentRequestStatusEnumPAID),
			},
			inputs: []Input{
				FinanceInputAuditorApprove,
				FinanceInputAuditorReject,
				FinanceInputAgencyApprove,
				FinanceInputAgencyReject,
				FinanceInputDisburse,
			},
			allowed: map[State]map[Input]State{
				payment(entity.PaymentRequestStatusEnumSUBMITTED): {
					FinanceInputAuditorApprove: payment(entity.PaymentRequestStatusEnumAUDITORAPPROVED),
					FinanceInputAuditorReject:  payment(entity.PaymentRequestStatusEnumAUDITORREJECTED),
				},
				payment(entity.PaymentRequestStatusEnumAUDITORAPPROVED): {
					FinanceInputAgencyApprove: payment(entity.PaymentRequestStatusEnumAGENCYAPPROVED),
					FinanceInputAgencyReject:  payment(entity.PaymentRequestStatusEnumAGENCYREJECTED),
				},
				payment(entity.PaymentRequestStatusEnumAGENCYAPPROVED): {
					FinanceInputDisburse: payment(entity.PaymentRequestStatusEnumPAID),
				},
			},
		},
		{
			name:    "tracker",
			machine: trackerMachine,
			states: []State{
				ticket(entity.TicketStatusEnumOPEN),
				ticket(entity.TicketStatusEnumASSIGNED),
				ticket(entity.TicketStatusEnumINPROGRESS),
				ticket(entity.TicketStatusEnumRESOLVED),
				ticket(entity.TicketStatusEnumCLOSED),
			},
			inputs: []Input{
				TrackerInputAssign,
				TrackerInputComment,
				TrackerInputSetOpen,
				TrackerInputSetAssigned,
				TrackerInputSetProgress,
				TrackerInputSetResolved,
				TrackerInputClose,
			},
			allowed: map[State]map[Input]State{
				ticket(entity.TicketStatusEnumOPEN): {
					TrackerInputAssign:  ticket(entity.TicketStatusEnumASSIGNED),
					TrackerInputComment: ticket(entity.TicketStatusEnumOPEN),
				},
				ticket(entity.TicketStatusEnumASSIGNED): {
					TrackerInputAssign:      ticket(entity.TicketStatusEnumASSIGNED),
					TrackerInputComment:     ticket(entity.TicketStatusEnumASSIGNED),
					TrackerInputSetProgress: ticket(entity.TicketStatusEnumINPROGRESS),
				},
				ticket(entity.TicketStatusEnumINPROGRESS): {
					TrackerInputAssign:      ticket(entity.TicketStatusEnumASSIGNED),
					TrackerInputComment:     ticket(entity.TicketStatusEnumINPROGRESS),
					TrackerInputSetResolved: ticket(entity.TicketStatusEnumRESOLVED),
				},
				ticket(entity.TicketStatusEnumRESOLVED): {
					TrackerInputComment:     ticket(entity.TicketStatusEnumRESOLVED),
					TrackerInputSetProgress: ticket(entity.TicketStatusEnumINPROGRESS),
					TrackerInputClose:       ticket(entity.TicketStatusEnumCLOSED),
				},
			},
		},
	}
}

// TestFSMExhaustiveStateInputPairs checks every (state, input) pair of every
// workflow machine against the specified matrix: allowed pairs must reach the
// specified next state, all other pairs must be rejected.
func TestFSMExhaustiveStateInputPairs(t *testing.T) {
	totalPairs, totalAllowed := 0, 0

	for _, spec := range fsmSpecs() {
		allowed := 0
		for _, state := range spec.states {
			for _, input := range spec.inputs {
				want, wantOK := spec.allowed[state][input]
				got, ok := spec.machine.Next(state, input)
				if ok != wantOK {
					t.Errorf("%s: %s + %s accepted = %v, want %v", spec.name, state, input, ok, wantOK)
					continue
				}
				if ok {
					allowed++
					if got != want {
						t.Errorf("%s: %s + %s -> %s, want %s", spec.name, state, input, got, want)
					}
				}
			}
		}

		pairs := len(spec.states) * len(spec.inputs)
		totalPairs += pairs
		totalAllowed += allowed
		t.Logf("FSM %-10s states=%d inputs=%d pairs=%d allowed=%d rejected=%d",
			spec.name, len(spec.states), len(spec.inputs), pairs, allowed, pairs-allowed)
	}

	t.Logf("FSM total pairs=%d allowed=%d rejected=%d", totalPairs, totalAllowed, totalPairs-totalAllowed)
}

// TestFSMMachinesHaveNoUnspecifiedStates guards the exhaustive test itself:
// every state that appears in a machine must be part of its spec.
func TestFSMMachinesHaveNoUnspecifiedStates(t *testing.T) {
	for _, spec := range fsmSpecs() {
		known := make(map[State]bool, len(spec.states))
		for _, s := range spec.states {
			known[s] = true
		}
		knownInput := make(map[Input]bool, len(spec.inputs))
		for _, in := range spec.inputs {
			knownInput[in] = true
		}
		for from, inputs := range spec.machine.transitions {
			if !known[from] {
				t.Errorf("%s: machine has unspecified source state %s", spec.name, from)
			}
			for input, to := range inputs {
				if !knownInput[input] {
					t.Errorf("%s: machine has unspecified input %s", spec.name, input)
				}
				if !known[to] {
					t.Errorf("%s: machine has unspecified target state %s", spec.name, to)
				}
			}
		}
	}
}

// TestFSMTerminalStatesHaveNoOutgoingTransitions verifies that the states the
// governance design marks as terminal cannot be left by any input.
func TestFSMTerminalStatesHaveNoOutgoingTransitions(t *testing.T) {
	terminal := map[string][]State{
		"monitoring": {MonitoringStateAccepted, MonitoringStateRejected},
		"finance": {
			State(entity.PaymentRequestStatusEnumAUDITORREJECTED),
			State(entity.PaymentRequestStatusEnumAGENCYREJECTED),
			State(entity.PaymentRequestStatusEnumPAID),
		},
		"tracker": {State(entity.TicketStatusEnumCLOSED)},
	}

	for _, spec := range fsmSpecs() {
		for _, state := range terminal[spec.name] {
			for _, input := range spec.inputs {
				if next, ok := spec.machine.Next(state, input); ok {
					t.Errorf("%s: terminal state %s left via %s to %s", spec.name, state, input, next)
				}
			}
		}
	}
}
