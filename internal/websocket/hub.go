package websocket

type GameHandler interface {
	HandlePlayerReconnect(playerId string)
	HandlePlayerDisconnect(playerId string)
	HandleWordSelect(word string)
	HandleTimerStartMessages(payload map[string]interface{})
	HandleTimerStopMessages(payload map[string]interface{})
	HandlePlayerGuess(playerId string, username string, guess string)
	GetGameState() map[string]any
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
			// go h.gameHandler.HandlePlayerReconnect(client.PlayerId)
		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				// go h.gameHandler.HandlePlayerDisconnect(client.PlayerId)
				close(client.Send)
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
		}
	}
}
