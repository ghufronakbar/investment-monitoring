package websocket_service

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"investment_monitoring/config"
	token_service "investment_monitoring/internal/module/support/token/service"
	"investment_monitoring/internal/policy"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"

	"github.com/redis/go-redis/v9"
	"nhooyr.io/websocket"
)

type Service struct {
	cfg        *config.Config
	tokenMaker *token_service.TokenMaker
	store      entity.Store
	policy     *policy.Service
	hub        *Hub
	pubsub     *redisPubSub
}

func NewService(cfg *config.Config, tokenMaker *token_service.TokenMaker, store entity.Store, rdb *redis.Client) *Service {
	channel := cfg.WEBSOCKET_REDIS_CHANNEL_PREFIX + ":events"
	return &Service{
		cfg:        cfg,
		tokenMaker: tokenMaker,
		store:      store,
		policy:     newTopicPolicy(store),
		hub:        NewHub(),
		pubsub:     newRedisPubSub(rdb, channel),
	}
}

func (s *Service) Enabled() bool {
	return s.cfg.WEBSOCKET_ENABLED
}

func (s *Service) AllowAllOrigins() bool {
	for _, origin := range s.cfg.WEBSOCKET_ALLOWED_ORIGINS {
		if origin == "*" {
			return true
		}
	}
	return false
}

func (s *Service) OriginPatterns() []string {
	if s.AllowAllOrigins() {
		return nil
	}
	return s.cfg.WEBSOCKET_ALLOWED_ORIGINS
}

func (s *Service) AuthenticateRequest(r *http.Request) (*token_service.AccessTokenClaims, error) {
	token := strings.TrimSpace(r.URL.Query().Get("accessToken"))
	if token == "" {
		token = extractBearerToken(r.Header.Get("Authorization"))
	}
	if token == "" {
		return nil, errs.NewUnauthorized("MISSING_TOKEN")
	}
	return s.tokenMaker.ValidateAccessToken(token)
}

func (s *Service) HandleConnection(ctx context.Context, conn *websocket.Conn, claims *token_service.AccessTokenClaims) {
	conn.SetReadLimit(s.cfg.WEBSOCKET_READ_LIMIT_BYTES)

	client := NewClient(claims.ID, claims.Email, s.cfg.WEBSOCKET_CLIENT_SEND_BUFFER)
	s.hub.Register(client)

	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		s.writeLoop(ctx, conn, client)
	}()

	client.Enqueue(NewSystemEnvelope(EventRealtimeConnected, ConnectedData{
		ClientID: client.ID,
		UserID:   client.UserID,
		Topics:   client.Topics(),
	}))

	s.readLoop(ctx, conn, client)

	s.hub.Unregister(client)
	_ = conn.Close(websocket.StatusNormalClosure, "connection closed")
	<-writeDone
}

func (s *Service) Start(ctx context.Context) error {
	if !s.Enabled() || s.pubsub == nil {
		return nil
	}
	s.pubsub.start(ctx, s.hub.Publish)
	return nil
}

func (s *Service) Shutdown(_ context.Context) error {
	if s.pubsub != nil {
		s.pubsub.shutdown()
	}
	s.hub.Shutdown()
	return nil
}

func (s *Service) readLoop(ctx context.Context, conn *websocket.Conn, client *Client) {
	for {
		readCtx := ctx
		cancel := func() {}
		if s.cfg.WEBSOCKET_IDLE_TIMEOUT > 0 {
			readCtx, cancel = context.WithTimeout(ctx, s.cfg.WEBSOCKET_IDLE_TIMEOUT)
		}

		messageType, data, err := conn.Read(readCtx)
		cancel()
		if err != nil {
			return
		}
		if messageType != websocket.MessageText {
			client.Enqueue(NewSystemEnvelope(EventRealtimeError, ErrorData{Message: "UNSUPPORTED_MESSAGE_TYPE"}))
			continue
		}

		var msg ClientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			client.Enqueue(NewSystemEnvelope(EventRealtimeError, ErrorData{Message: "INVALID_JSON"}))
			continue
		}
		s.handleClientMessage(ctx, client, msg)
	}
}

func (s *Service) writeLoop(ctx context.Context, conn *websocket.Conn, client *Client) {
	for envelope := range client.Send() {
		data, err := json.Marshal(envelope)
		if err != nil {
			continue
		}

		writeCtx := ctx
		cancel := func() {}
		if s.cfg.WEBSOCKET_WRITE_TIMEOUT > 0 {
			writeCtx, cancel = context.WithTimeout(ctx, s.cfg.WEBSOCKET_WRITE_TIMEOUT)
		}

		err = conn.Write(writeCtx, websocket.MessageText, data)
		cancel()
		if err != nil {
			return
		}
	}
}

func (s *Service) handleClientMessage(ctx context.Context, client *Client, msg ClientMessage) {
	switch msg.Type {
	case ClientMessagePing:
		client.Enqueue(NewSystemEnvelope(EventRealtimePong, PongData{
			ReceivedAt: time.Now().UTC(),
		}))
	case ClientMessageSubscribe:
		s.handleSubscribe(ctx, client, msg)
	case ClientMessageUnsubscribe:
		s.handleUnsubscribe(client, msg)
	default:
		client.Enqueue(NewSystemEnvelope(EventRealtimeError, ErrorData{Message: "UNKNOWN_MESSAGE_TYPE"}))
	}
}

func (s *Service) handleSubscribe(ctx context.Context, client *Client, msg ClientMessage) {
	subscribed := []string{}
	rejected := []string{}

	for _, topic := range msg.NormalizedTopics() {
		if err := s.AuthorizeTopic(ctx, client.UserID, topic); err != nil {
			rejected = append(rejected, topic)
			continue
		}
		s.hub.Subscribe(client, topic)
		subscribed = append(subscribed, topic)
	}

	client.Enqueue(NewSystemEnvelope(EventRealtimeSubscriptionUpdated, SubscriptionUpdatedData{
		Subscribed:   subscribed,
		Unsubscribed: []string{},
		Rejected:     rejected,
	}))
}

func (s *Service) handleUnsubscribe(client *Client, msg ClientMessage) {
	unsubscribed := []string{}
	rejected := []string{}

	for _, topic := range msg.NormalizedTopics() {
		if !isValidTopic(topic) {
			rejected = append(rejected, topic)
			continue
		}
		s.hub.Unsubscribe(client, topic)
		unsubscribed = append(unsubscribed, topic)
	}

	client.Enqueue(NewSystemEnvelope(EventRealtimeSubscriptionUpdated, SubscriptionUpdatedData{
		Subscribed:   []string{},
		Unsubscribed: unsubscribed,
		Rejected:     rejected,
	}))
}

func isValidTopic(topic string) bool {
	return strings.TrimSpace(topic) != "" && !strings.ContainsAny(topic, " \t\r\n")
}

func extractBearerToken(header string) string {
	parts := strings.Split(header, " ")
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return ""
}
