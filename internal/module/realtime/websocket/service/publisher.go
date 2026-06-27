package websocket_service

import "context"

type Publisher interface {
	Publish(ctx context.Context, envelope Envelope) error
}

func (s *Service) Publish(_ context.Context, envelope Envelope) error {
	s.hub.Publish(envelope)
	return nil
}
