package websocket_service

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventRealtimeConnected           = "realtime.connected"
	EventRealtimePong                = "realtime.pong"
	EventRealtimeSubscriptionUpdated = "realtime.subscription_updated"
	EventRealtimeError               = "realtime.error"

	TopicRealtimeSystem = "realtime.system"
)

type Envelope struct {
	Type   string    `json:"type"`
	Topic  string    `json:"topic"`
	Data   any       `json:"data"`
	SentAt time.Time `json:"sentAt"`
}

type ConnectedData struct {
	ClientID string    `json:"clientId"`
	UserID   uuid.UUID `json:"userId"`
	Topics   []string  `json:"topics"`
}

type PongData struct {
	ReceivedAt time.Time `json:"receivedAt"`
}

type SubscriptionUpdatedData struct {
	Subscribed   []string `json:"subscribed"`
	Unsubscribed []string `json:"unsubscribed"`
	Rejected     []string `json:"rejected"`
}

type ErrorData struct {
	Message string `json:"message"`
}

func NewSystemEnvelope(eventType string, data any) Envelope {
	return Envelope{
		Type:   eventType,
		Topic:  TopicRealtimeSystem,
		Data:   data,
		SentAt: time.Now().UTC(),
	}
}
