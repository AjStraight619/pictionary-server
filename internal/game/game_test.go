package game

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/Ajstraight619/pictionary-server/internal/database"
)

func createTestPlayers(num int) []*Player {
	players := make([]*Player, num)
	for i := 0; i < num; i++ {
		players[i] = &Player{
			Id:       fmt.Sprintf("player-%d", i),
			Username: fmt.Sprintf("Player %d", i),
			IsLeader: i == 0, // First player is the leader
		}
	}
	return players
}

func createTestGame(players []*Player, options GameOptions) *Game {
	return &Game{
		Id:        "test-game",
		Players:   players,
		playerIds: make(map[string]struct{}),
		Options:   options,
		Round: &Round{
			Count:                 1,
			CurrentDrawerIdx:      0,
			PlayersDrawnThisRound: make(map[string]struct{}),
		},
	}
}

func TestGameLoop(t *testing.T) {

	databasePath, err := filepath.Abs("../../data/game.db")
	if err != nil {
		t.Fatalf("Failed to resolve absolute path for database: %v", err)
	}

	database.InitDB(databasePath)

	gameOptions := GameOptions{
		MaxRounds:       6,
		SelectWordTimer: 10,
		SelectWordCount: 3,
		TurnTimer:       15,
	}

	players := createTestPlayers(4)

	game := createTestGame(players, gameOptions)

	game.Print()

	game.GetRandomWords("", 3)

	game.Print()

}
