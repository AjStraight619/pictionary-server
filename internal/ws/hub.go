package ws

import (
	"context"
	"log"
	"sync"

	e "github.com/Ajstraight619/pictionary-server/internal/events"
)

type Hub struct {
	ctx          context.Context
	Broadcast    chan []byte
	GameEvents   chan e.GameEvent
	Clients      map[*Client]bool
	Register     chan *Client
	Unregister   chan *Client
	OnDisconnect func(playerID string)
}

type Hubs struct {
	mu   sync.RWMutex
	Hubs map[string]*Hub
}

func NewHub(ctx context.Context) *Hub {
	return &Hub{
		ctx:        ctx,
		Broadcast:  make(chan []byte),
		GameEvents: make(chan e.GameEvent, 10),
		Clients:    make(map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

func NewHubs() *Hubs {
	return &Hubs{
		Hubs: make(map[string]*Hub),
	}
}

func (h *Hubs) AddHub(gameID string, hub *Hub) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Hubs[gameID] = hub
}

func (h *Hubs) GetHub(gameID string) (*Hub, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	hub, exists := h.Hubs[gameID]
	return hub, exists
}

func (h *Hubs) RemoveHub(gameID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.Hubs, gameID)
}

func (h *Hub) Run() {
	defer h.cleanup()

	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
				// if h.OnDisconnect != nil {
				// 	go h.OnDisconnect(client.PlayerID)
				// }
			}
		case message := <-h.Broadcast:
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
		case <-h.ctx.Done():
			log.Printf("Hub is shutting down...")
			return
		}
	}
}

func (h *Hub) BroadcastMessage(message []byte) {
	h.Broadcast <- message
}

func (h *Hub) SendToPlayer(playerID string, message []byte) {
	for client := range h.Clients {
		if client.PlayerID == playerID {
			client.Send <- message
			break
		}
	}
}

func (h *Hub) GameEventChannel() <-chan e.GameEvent {
	return h.GameEvents
}

func (h *Hub) cleanup() {
	// Close all client connections
	for client := range h.Clients {
		close(client.Send)
		client.Conn.Close()
		delete(h.Clients, client)
	}

	// Clean up channels
	close(h.Broadcast)
	close(h.GameEvents)
}
