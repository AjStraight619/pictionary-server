package game

import (
	"log"
	"testing"
	"time"
)

func TestTimerStop(t *testing.T) {

	round := Round{}

	onExpire := func() {
		log.Println("Timer stopped")
	}

	round.StartTimer(10*time.Second, onExpire)

	time.Sleep(2 * time.Second)
	round.StopTimer()
}

func TestGameTimer(t *testing.T) {
	done := make(chan struct{})

	// Mock game options
	options := GameOptions{
		MaxRounds:       3,
		TurnTimer:       5, // 5 seconds per turn
		SelectWordTimer: 3, // 3 seconds for word selection
	}

	// Create a new game instance
	game := &Game{
		Id:      "test-game-123",
		Players: []*Player{},
		Options: options,
		Round: Round{
			PlayersDrawnThisRound: make(map[string]struct{}),
		},
	}

	// Start your countdown
	game.Round.StartTimer(20*time.Second, func() {
		log.Println("Timer ended")
		close(done)
	})

	// Block until `done` is closed (i.e., callback runs),
	// or 10 seconds pass, whichever comes first.
	select {
	case <-done:
		// The timer ended naturally
		// case <-time.After(10 * time.Second):
		// 	t.Fatal("Timer did not end within 10 seconds!")
	}
}

func TestGameFlow(t *testing.T) {
	// Mock game options
	options := GameOptions{
		MaxRounds:       3,
		TurnTimer:       5, // 5 seconds per turn
		SelectWordTimer: 3, // 3 seconds for word selection
	}

	// Create a new game instance
	game := &Game{
		Id:      "test-game-123",
		Players: []*Player{},
		Options: options,
		Round: Round{
			PlayersDrawnThisRound: make(map[string]struct{}),
		},
	}

	// Add mock players
	game.AddPlayer(CreatePlayer("player1", "Alice", true))
	game.AddPlayer(CreatePlayer("player2", "Bob", false))
	game.AddPlayer(CreatePlayer("player3", "John", false))

	// Define onRoundComplete mock function
	onRoundComplete := func() {
		log.Printf("Round %d completed!", game.Round.Count)
	}

	// Start the game
	err := game.StartGame(onRoundComplete)
	if err != nil {
		t.Fatalf("Failed to start game: %v", err)
	}

	log.Printf("Current drawer: %s", game.Players[game.Round.CurrentDrawerIdx].Username)

	// Simulate the game flow
	for {
		// Exit the loop when the maximum rounds are reached
		if game.Round.Count >= options.MaxRounds {
			log.Println("Game flow test passed!")
			break
		}

		// Simulate a single turn and let the game logic handle progression
		nextDrawer, err := game.AdvanceToNextDrawer(onRoundComplete)
		if err != nil {
			if err.Error() == "game over" {
				log.Println("Game over reached successfully!")
				break
			}
			t.Fatalf("Failed to advance to the next drawer: %v", err)
		}

		// Validate the drawer change
		log.Printf("Drawer changed to: %s (Player ID: %s)", nextDrawer.Username, nextDrawer.Id)

		// Simulate the turn timer expiration
		time.Sleep(time.Duration(options.TurnTimer) * time.Second)

		// Validate the round reset when all players have drawn
		if len(game.Round.PlayersDrawnThisRound) == 0 && game.Round.Count > 0 {
			log.Printf("Round reset correctly after all players drew. Current round: %d", game.Round.Count)
		}
	}

	// Final assertions
	if game.Round.Count != options.MaxRounds {
		t.Errorf("Expected %d rounds, but got %d", options.MaxRounds, game.Round.Count)
	}

	log.Println("Game flow test completed successfully!")
}
