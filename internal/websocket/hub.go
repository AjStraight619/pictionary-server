package websocket

type GameHandler interface {
	HandlePlayerReconnect(playerId string)
	HandlePlayerDisconnect(playerId string)
}

type Hub struct {
	gameHandler GameHandler
	Clients     map[*Client]bool
	Broadcast   chan []byte
	Register    chan *Client
	Unregister  chan *Client
}

func NewHub(gameHandler GameHandler) *Hub {
	return &Hub{
		gameHandler: gameHandler,
		Broadcast:   make(chan []byte),
		Register:    make(chan *Client),
		Unregister:  make(chan *Client),
		Clients:     make(map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
			h.gameHandler.HandlePlayerReconnect(client.playerId)
		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.send)
				h.gameHandler.HandlePlayerDisconnect(client.playerId)
			}

		case message := <-h.Broadcast:
			for client := range h.Clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.Clients, client)
				}
			}
		}
	}
}
