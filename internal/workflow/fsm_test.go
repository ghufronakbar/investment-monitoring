package workflow

import (
	"strings"
	"testing"
)

func TestMachineNextAndMustNext(t *testing.T) {
	machine := NewMachine([]Transition{
		{From: "A", Input: "go", To: "B"},
	})

	next, ok := machine.Next("A", "go")
	if !ok {
		t.Fatal("expected transition to be valid")
	}
	if next != "B" {
		t.Fatalf("next = %s, want B", next)
	}

	if _, ok := machine.Next("B", "go"); ok {
		t.Fatal("expected unknown state transition to be invalid")
	}
	if _, ok := machine.Next("A", "stay"); ok {
		t.Fatal("expected unknown input transition to be invalid")
	}

	if _, err := machine.MustNext("A", "stay"); err == nil || !strings.Contains(err.Error(), "invalid transition") {
		t.Fatalf("MustNext error = %v, want invalid transition", err)
	}
}
