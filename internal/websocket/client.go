package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type BroadcastMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type PlayerGuessPayload struct {
	PlayerId string `json:"playerId"`
	Username string `json:"username"`
	Guess    string `json:"guess"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	EnableCompression: true,
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.

	maxMessageSize = 20480
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type Client struct {
	gameId   string
	PlayerId string
	conn     *websocket.Conn
	Send     chan []byte
	hub      *Hub
}

func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Unexpected close error: %v", err)
			} else {
				log.Printf("Read error: %v", err)
			}
			break
		}

		// Handle ping messages
		if string(message) == "ping" {
			log.Println("Received ping, responding with pong")
			if err := c.conn.WriteMessage(websocket.PongMessage, nil); err != nil {
				log.Printf("Error sending pong: %v", err)
			}
			continue
		}

		// if string(message) == "ping" {
		// 	log.Println("Received ping, responding with pong")
		// 	if err := c.conn.WriteMessage(websocket.TextMessage, []byte("pong")); err != nil {
		// 		log.Printf("Error sending pong: %v", err)
		// 	}
		// 	continue
		// }

		// Parse the message into the expected structure
		var parsedMessage Message
		if err := json.Unmarshal(message, &parsedMessage); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		// Handle the parsed message
		switch parsedMessage.Type {
		case "start_timer", "stop_timer":
			payloadMap, err := parsePayloadAsMap(parsedMessage.Payload)
			if err != nil {
				log.Println(err)
				break
			}

			if parsedMessage.Type == "start_timer" {
				c.hub.gameHandler.HandleTimerStartMessages(payloadMap)
			} else {
				c.hub.gameHandler.HandleTimerStopMessages(payloadMap)
			}

		case "select_word":
			payload, ok := parsedMessage.Payload.(map[string]interface{})
			if !ok {
				log.Println("Invalid payload: expected an object")
				break
			}

			word, ok := payload["word"].(string)
			if !ok {
				log.Println("Invalid payload: expected 'word' to be a string")
				break
			}

			c.hub.gameHandler.HandleWordSelect(word)

		case "player_guess":
			// Define the expected payload structure
			var playerGuessPayload PlayerGuessPayload

			// Parse the payload into the struct
			payloadBytes, err := json.Marshal(parsedMessage.Payload) // Convert interface{} to []byte
			if err != nil {
				log.Printf("Failed to marshal payload: %v", err)
				break
			}

			err = json.Unmarshal(payloadBytes, &playerGuessPayload)
			if err != nil {
				log.Printf("Invalid payload for player_guess: %v", err)
				break
			}

			// Now you can access `playerGuessPayload.PlayerId` and `playerGuessPayload.Guess`

			// Handle the guess
			c.hub.gameHandler.HandlePlayerGuess(playerGuessPayload.PlayerId, playerGuessPayload.Username, playerGuessPayload.Guess)
			continue

		default:
			break
		}

		// Broadcast the message to all clients
		jsonData, err := json.Marshal(parsedMessage)
		if err != nil {
			log.Printf("Error marshaling message: %v", err)
			continue
		}
		c.hub.Broadcast <- jsonData
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:

			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return

			}

			w, err := c.conn.NextWriter(websocket.TextMessage)

			if err != nil {
				log.Printf("Error with NextWriter on websocket Text Message")
				return
			}

			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		}

	}
}

func ServeWs(hub *Hub, gameId string, userId string, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{
		hub:      hub,
		conn:     conn,
		Send:     make(chan []byte, 256),
		gameId:   gameId,
		PlayerId: userId,
	}

	// Register the client in the Hub
	hub.Register <- client

	// Kick off read/write pumps
	go client.writePump()
	go client.readPump()

	// When client connects, send them the current game state

	gameState := hub.gameHandler.GetGameState()

	message := BroadcastMessage{
		Type:    "game_state",
		Payload: gameState,
	}

	jsonData, err := json.Marshal(message)

	if err != nil {
		log.Printf("Something went wrong marshalling game state: %v", err)
		return
	}

	client.hub.Broadcast <- jsonData
}

func parsePayloadAsMap(payload any) (map[string]interface{}, error) {
	payloadMap, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload: expected an object")
	}
	return payloadMap, nil
}
