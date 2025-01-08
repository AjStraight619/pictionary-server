package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

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
	maxMessageSize = 5128
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

type Message struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
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
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {

		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Unexpected close error: %v", err)
			} else {
				log.Printf("Read error: %v", err)
			}
			break // Ensure the loop exits on error
		}

		if string(message) == "ping" {
			log.Println("Received ping, responding with pong")
			if err := c.conn.WriteMessage(websocket.PongMessage, nil); err != nil {
				log.Printf("Error sending pong: %v", err)
			}
			continue
		}

		var parsedMessage Message

		err = json.Unmarshal(message, &parsedMessage)

		if err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		if parsedMessage.Type == "path" {
			log.Println("Recieved path data")
		}

		// TODO: Handle incomming messages based on message type. Timers need to be set/canceled. Guesses need to be checked.

		// Handle messages here

		// switch parsedMessage.Type {
		// 	case
		// }

		jsonData, err := json.Marshal(parsedMessage)

		if err != nil {
			log.Printf("Error marshaling message: %v", err)
			continue
		}

		c.hub.Broadcast <- jsonData
		// log.Printf("Received message: Type=%s, Payload=%s", parsedMessage.Type, parsedMessage.Payload)

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
		case message, ok := <-c.send:

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
		send:     make(chan []byte, 256),
		gameId:   gameId,
		playerId: userId,
	}

	// Register the client in the Hub
	hub.Register <- client

	// Kick off read/write pumps
	go client.writePump()
	go client.readPump()
}
