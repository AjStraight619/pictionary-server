package game

import (
	"encoding/json"
	"log"
	"sync"

	ws "github.com/Ajstraight619/pictionary-server/internal/websocket"
)

type CreateGameParams struct {
	GameId   string
	PlayerId string
	Username string
	Options  GameOptions
}

type GameOptions struct {
	MaxRounds       int
	TurnTimer       int
	SelectWordTimer int
}

type GameManager struct {
	Games map[string]*Game
	mu    sync.Mutex
}

func NewGameManager() *GameManager {
	return &GameManager{
		Games: make(map[string]*Game),
	}
}

func (gm *GameManager) CreateGame(params CreateGameParams) *Game {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if _, exists := gm.Games[params.GameId]; exists {
		return nil
	}

	// Create the game without initializing the Hub yet
	game := &Game{
		Id: params.GameId,
		Players: []*Player{
			{Id: params.PlayerId, Username: params.Username, IsLeader: true},
		},
		playerIds: map[string]struct{}{
			params.PlayerId: {}, // Add the player's ID to the set
		},
		Options: params.Options,
	}

	// Initialize the Hub and associate it with the game
	game.Hub = ws.NewHub(game) // Pass the game as the GameHandler
	go game.Hub.Run()

	// Store the game
	gm.Games[params.GameId] = game

	leader := game.Players[0]

	message := BroadcastMessage{
		Type: "player-joined",
		Payload: struct {
			PlayerId  string `json:"playerId"`
			Username  string `json:"username"`
			IsLeader  bool   `json:"isLeader"`
			IsDrawing bool   `json:"isDrawing"`
			Score     int16  `json:"score"`
		}{
			PlayerId:  leader.Id,
			Username:  leader.Username,
			IsLeader:  leader.IsLeader,
			IsDrawing: leader.IsDrawing,
			Score:     leader.Score,
		},
	}

	jsonData, err := json.Marshal(&message)

	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return nil

	}

	game.Hub.Broadcast <- jsonData

	return game
}

func (gm *GameManager) GetGame(gameId string) (*Game, bool) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	game, exists := gm.Games[gameId]
	return game, exists
}

func (gm *GameManager) RemoveGame(gameId string) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	delete(gm.Games, gameId)
}
