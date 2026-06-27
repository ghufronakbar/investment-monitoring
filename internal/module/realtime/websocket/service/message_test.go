package websocket_service

import (
	"reflect"
	"testing"
)

func TestClientMessageNormalizedTopics(t *testing.T) {
	msg := ClientMessage{
		Topic: "project.1",
		Topics: []string{
			"project.1",
			" ticket.2 ",
			"",
			"ticket.2",
		},
	}

	got := msg.NormalizedTopics()
	want := []string{"project.1", "ticket.2"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizedTopics() = %#v, want %#v", got, want)
	}
}

func TestIsValidTopic(t *testing.T) {
	tests := []struct {
		name  string
		topic string
		want  bool
	}{
		{name: "valid", topic: "project.1.monitoring", want: true},
		{name: "empty", topic: "", want: false},
		{name: "space", topic: "project.1 monitoring", want: false},
		{name: "newline", topic: "project.1\nmonitoring", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidTopic(tt.topic); got != tt.want {
				t.Fatalf("isValidTopic(%q) = %v, want %v", tt.topic, got, tt.want)
			}
		})
	}
}
