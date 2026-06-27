package websocket_service

import "strings"

const (
	ClientMessagePing        = "ping"
	ClientMessageSubscribe   = "subscribe"
	ClientMessageUnsubscribe = "unsubscribe"
)

type ClientMessage struct {
	Type   string   `json:"type"`
	Topic  string   `json:"topic,omitempty"`
	Topics []string `json:"topics,omitempty"`
}

func (m ClientMessage) NormalizedTopics() []string {
	seen := make(map[string]bool)
	topics := make([]string, 0, len(m.Topics)+1)

	if topic := strings.TrimSpace(m.Topic); topic != "" {
		seen[topic] = true
		topics = append(topics, topic)
	}

	for _, topic := range m.Topics {
		topic = strings.TrimSpace(topic)
		if topic == "" || seen[topic] {
			continue
		}
		seen[topic] = true
		topics = append(topics, topic)
	}

	return topics
}
