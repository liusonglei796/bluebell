package sse

import (
	"context"
	"sync"
)

type Hub struct {
	clients map[int64]chan interface{}
	mu      sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[int64]chan interface{}),
	}
}

func (h *Hub) Run(ctx context.Context) {
	<-ctx.Done()
}

func (h *Hub) Subscribe(userID int64) chan interface{} {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan interface{}, 64)
	h.clients[userID] = ch
	return ch
}

func (h *Hub) Unsubscribe(userID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if ch, ok := h.clients[userID]; ok {
		close(ch)
		delete(h.clients, userID)
	}
}

func (h *Hub) Send(userID int64, v interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if ch, ok := h.clients[userID]; ok {
		select {
		case ch <- v:
		default:
		}
	}
}

func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
