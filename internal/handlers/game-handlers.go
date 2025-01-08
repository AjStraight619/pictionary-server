package handlers

import (
	"log"
	"net/http"

	g "github.com/Ajstraight619/pictionary-server/internal/game"
	ws "github.com/Ajstraight619/pictionary-server/internal/websocket"
	"github.com/gin-gonic/gin"
)

type JoinGame struct {
	GameId     string `json:"gameId" binding:"required"`
	PlayerId   string `json:"playerId" binding:"required"`
	Playername string `json:"playerName" binding:"required"`
}

type CreateGame struct {
	GameId      string `json:"gameId" binding:"required"`
	PlayerId    string `json:"playerId" binding:"required"`
	Playername  string `json:"playerName" binding:"required"`
	GameOptions struct {
		MaxRounds       int `json:"maxRounds"`
		TurnTimer       int `json:"turnTimer"`
		SelectWordTimer int `json:"selectWordTimer"`
	} `json:"gameOptions"`
}

func RegisterGameRoutes(router *gin.Engine, gm *g.GameManager) {
	router.POST("/create-game", func(c *gin.Context) {
		var payload CreateGame
		// Bind JSON payload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload", "details": err.Error()})
			return
		}

		params := g.CreateGameParams{
			GameId:   payload.GameId,
			PlayerId: payload.PlayerId,
			Username: payload.Playername, // Match field names between structs
			Options: g.GameOptions{
				MaxRounds:       payload.GameOptions.MaxRounds,
				TurnTimer:       payload.GameOptions.TurnTimer,
				SelectWordTimer: payload.GameOptions.SelectWordTimer,
			},
		}

		createdGame := gm.CreateGame(params)

		if createdGame == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Game already exists"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	router.POST("/join-game", func(c *gin.Context) {
		var payload JoinGame

		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload", "details": err.Error()})
			return
		}

		game, exists := gm.GetGame(payload.GameId)

		if !exists {
			log.Printf("Game with ID %s does not exist", payload.GameId)
			c.JSON(http.StatusNotFound, gin.H{
				"error":  "Game not found",
				"gameId": payload.GameId,
			})
			return
		}

		if game.Status == g.StatusInProgress {
			// Return a 403 if the game is in progress
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Game is already in progress.",
			})
			return
		}

		if game.Status == g.StatusFinished {
			// Maybe return a 409 or 403—whatever makes sense if it’s finished
			c.JSON(http.StatusConflict, gin.H{
				"error": "Game has already finished.",
			})
			return
		}

		totalPlayers := len(game.Players)

		if totalPlayers < 8 {

			player := g.CreatePlayer(payload.PlayerId, payload.GameId, false)

			err := game.AddPlayer(player)

			if err != nil {
				log.Printf("Could not add player to game: %s", err)
				c.JSON(http.StatusConflict, gin.H{
					"error":   "Could not add player to game",
					"details": err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{"success": true})
		}

	})

	router.GET("/game/:id", func(c *gin.Context) {
		gameId := c.Param("id")

		g, exists := gm.GetGame(gameId)

		if !exists {
			c.String(http.StatusNotFound, "Game does not exist")
			return
		}

		userId := c.Query("userId")
		username := c.Query("username")
		log.Printf("User connecting -> gameId: %s, userId: %s, username: %s\n",
			gameId, userId, username)

		// Pass the game’s Hub to ServeWs for the WebSocket handshake
		ws.ServeWs(g.Hub, gameId, userId, c.Writer, c.Request)
	})
}
