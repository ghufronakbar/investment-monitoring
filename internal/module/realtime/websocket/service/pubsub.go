package websocket_service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"

	"github.com/redis/go-redis/v9"
)

type redisPubSub struct {
	client  *redis.Client
	channel string
	cancel  context.CancelFunc
	sub     *redis.PubSub
	wg      sync.WaitGroup
}

func newRedisPubSub(client *redis.Client, channel string) *redisPubSub {
	if client == nil || channel == "" {
		return nil
	}
	return &redisPubSub{
		client:  client,
		channel: channel,
	}
}

func (r *redisPubSub) publish(ctx context.Context, envelope Envelope) error {
	if r == nil || r.client == nil {
		return errors.New("REDIS_PUBSUB_UNAVAILABLE")
	}

	payload, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	return r.client.Publish(ctx, r.channel, payload).Err()
}

func (r *redisPubSub) start(ctx context.Context, deliver func(Envelope)) {
	if r == nil || r.client == nil || deliver == nil {
		return
	}
	if r.cancel != nil {
		return
	}

	subCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	pubsub := r.client.Subscribe(subCtx, r.channel)
	r.sub = pubsub

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer pubsub.Close()

		if _, err := pubsub.Receive(subCtx); err != nil {
			log.Println("websocket redis subscriber unavailable:", err)
			return
		}

		for {
			msg, err := pubsub.ReceiveMessage(subCtx)
			if err != nil {
				if subCtx.Err() != nil {
					return
				}
				log.Println("websocket redis subscriber error:", err)
				return
			}

			var envelope Envelope
			if err := json.Unmarshal([]byte(msg.Payload), &envelope); err != nil {
				log.Println("websocket redis invalid payload:", err)
				continue
			}
			deliver(envelope)
		}
	}()
}

func (r *redisPubSub) shutdown() {
	if r == nil {
		return
	}
	if r.cancel != nil {
		r.cancel()
	}
	// A blocked ReceiveMessage does not observe context cancellation, so the
	// subscription must be closed to let the receive loop return.
	if r.sub != nil {
		_ = r.sub.Close()
	}
	r.wg.Wait()
	log.Printf("websocket redis fanout subscriber stopped channel=%s", r.channel)
}
