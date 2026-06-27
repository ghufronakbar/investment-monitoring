package websocket_service

import (
	"sort"
	"sync"

	"github.com/google/uuid"
)

type Client struct {
	ID     string
	UserID uuid.UUID
	Email  string

	send      chan Envelope
	topics    map[string]bool
	closeFunc func()

	mu     sync.RWMutex
	closed bool
}

func NewClient(userID uuid.UUID, email string, sendBuffer int) *Client {
	if sendBuffer <= 0 {
		sendBuffer = 64
	}

	return &Client{
		ID:     uuid.NewString(),
		UserID: userID,
		Email:  email,
		send:   make(chan Envelope, sendBuffer),
		topics: make(map[string]bool),
	}
}

func (c *Client) SetCloseFunc(closeFunc func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closeFunc = closeFunc
}

func (c *Client) Enqueue(envelope Envelope) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return false
	}

	select {
	case c.send <- envelope:
		return true
	default:
		return false
	}
}

func (c *Client) Send() <-chan Envelope {
	return c.send
}

func (c *Client) AddTopic(topic string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.topics[topic] = true
}

func (c *Client) RemoveTopic(topic string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.topics, topic)
}

func (c *Client) Topics() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	topics := make([]string, 0, len(c.topics))
	for topic := range c.topics {
		topics = append(topics, topic)
	}
	sort.Strings(topics)
	return topics
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	c.closed = true
	if c.closeFunc != nil {
		c.closeFunc()
	}
	close(c.send)
}
