package game

import (
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
			{Id: params.PlayerId, Username: params.Username, IsLeader: true, Color: "#FF5733"},
		},
		Status:                  0,
		TempDisconnectedPlayers: make(map[string]*Player),
		playerIds: map[string]struct{}{
			params.PlayerId: {}, // Empty set of playerId's
		},
		Options: params.Options,
	}

	// Initialize the Hub and associate it with the game
	game.Hub = ws.NewHub(game)
	go game.Hub.Run()

	gm.Games[params.GameId] = game

	leader := game.Players[0]

	log.Printf("Created game %s with leader: %v", game.Id, leader)

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
