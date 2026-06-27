package workflow

import "fmt"

type State string
type Input string

type Transition struct {
	From  State
	Input Input
	To    State
}

type Machine struct {
	transitions map[State]map[Input]State
}

func NewMachine(transitions []Transition) Machine {
	m := Machine{transitions: make(map[State]map[Input]State)}
	for _, transition := range transitions {
		if _, ok := m.transitions[transition.From]; !ok {
			m.transitions[transition.From] = make(map[Input]State)
		}
		m.transitions[transition.From][transition.Input] = transition.To
	}
	return m
}

func (m Machine) Next(current State, input Input) (State, bool) {
	inputs, ok := m.transitions[current]
	if !ok {
		return "", false
	}
	next, ok := inputs[input]
	return next, ok
}

func (m Machine) MustNext(current State, input Input) (State, error) {
	next, ok := m.Next(current, input)
	if !ok {
		return "", fmt.Errorf("invalid transition: %s + %s", current, input)
	}
	return next, nil
}
