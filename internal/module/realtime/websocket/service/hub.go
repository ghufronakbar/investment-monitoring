package websocket_service

import "sync"

type Hub struct {
	mu      sync.RWMutex
	clients map[string]*Client
	topics  map[string]map[string]*Client
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]*Client),
		topics:  make(map[string]map[string]*Client),
	}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client.ID] = client
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.clients, client.ID)
	for topic, clients := range h.topics {
		delete(clients, client.ID)
		if len(clients) == 0 {
			delete(h.topics, topic)
		}
	}
	client.Close()
}

func (h *Hub) Subscribe(client *Client, topic string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.topics[topic] == nil {
		h.topics[topic] = make(map[string]*Client)
	}
	h.topics[topic][client.ID] = client
	client.AddTopic(topic)
}

func (h *Hub) Unsubscribe(client *Client, topic string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients := h.topics[topic]; clients != nil {
		delete(clients, client.ID)
		if len(clients) == 0 {
			delete(h.topics, topic)
		}
	}
	client.RemoveTopic(topic)
}

func (h *Hub) Publish(envelope Envelope) {
	h.mu.RLock()
	topicClients := h.topics[envelope.Topic]
	clients := make([]*Client, 0, len(topicClients))
	for _, client := range topicClients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		if ok := client.Enqueue(envelope); !ok {
			h.Unregister(client)
		}
	}
}

func (h *Hub) Shutdown() {
	h.mu.Lock()
	clients := make([]*Client, 0, len(h.clients))
	for _, client := range h.clients {
		clients = append(clients, client)
	}
	h.clients = make(map[string]*Client)
	h.topics = make(map[string]map[string]*Client)
	h.mu.Unlock()

	for _, client := range clients {
		client.Close()
	}
}
