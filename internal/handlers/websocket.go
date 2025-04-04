package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/Ajstraight619/pictionary-server/internal/server"
	"github.com/Ajstraight619/pictionary-server/internal/utils"
	"github.com/Ajstraight619/pictionary-server/internal/ws"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func ServeWs(c echo.Context, server *server.GameServer) error {
	gameID := c.Param("id")
	log.Printf("ServeWs: received gameID: %s", gameID)

	game, exists := server.GetGame(gameID)
	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Game not found"})
	}

	hub, exists := server.GetHub(gameID)
	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Hub not found"})
	}

	playerID := c.QueryParam("playerID")
	username := c.QueryParam("username")

	if playerID == "" || username == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "PlayerID and Username are required"})
	}

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Unable to upgrade connection"})
	}

	// Update the player's connection status in the game state.
	player := game.GetPlayerByID(playerID)
	if player == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Player not found"})
	}
	player.Pending = false
	player.Connected = true
	player.Client = ws.NewClient(hub, conn, playerID)

	if wsClient, ok := player.Client.(*ws.Client); ok {
		hub.Register <- wsClient
	} else {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
	}

	go player.Client.Write()
	go player.Client.Read()

	msgType := "playerJoined"
	payload := map[string]interface{}{
		"player": player,
	}

	b, err := utils.CreateMessage(msgType, payload)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
	}

	hub.Broadcast <- b

	// Temporary fix to make sure the game state the ws connection is
	time.AfterFunc(200*time.Millisecond, func() {
		log.Println("game state:", game)
		game.BroadcastGameState()
	})

	return nil
}
