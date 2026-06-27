package websocket_service

import "context"

type Publisher interface {
	Publish(ctx context.Context, envelope Envelope) error
}

func (s *Service) Publish(ctx context.Context, envelope Envelope) error {
	if s.pubsub != nil {
		if err := s.pubsub.publish(ctx, envelope); err == nil {
			return nil
		}
	}

	s.hub.Publish(envelope)
	return nil
}
