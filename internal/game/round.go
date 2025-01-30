package game

import (
	"fmt"
	"log"
)

type Round struct {
	Count                 int
	CurrentDrawerIdx      int
	PlayersDrawnThisRound map[string]struct{}
	Game                  *Game
}

// Advance to the next drawer, handling end-of-round logic if necessary.
func (r *Round) NextDrawer() (*Player, error) {
	numPlayers := len(r.Game.Players)
	if numPlayers == 0 {
		return nil, fmt.Errorf("no players in the game")
	}

	currentDrawer := r.Game.Players[r.CurrentDrawerIdx]
	currentDrawer.IsDrawing = false

	for {
		r.CurrentDrawerIdx = (r.CurrentDrawerIdx + 1) % numPlayers
		nextPlayer := r.Game.Players[r.CurrentDrawerIdx]

		// Skip players who have already drawn in this round.
		if _, drawn := r.PlayersDrawnThisRound[nextPlayer.Id]; !drawn {
			nextPlayer.IsDrawing = true
			r.PlayersDrawnThisRound[nextPlayer.Id] = struct{}{}
			log.Printf("Next drawer is %s (ID: %s)", nextPlayer.Username, nextPlayer.Id)
			return nextPlayer, nil
		}

		// If all players have drawn, start a new round.
		if len(r.PlayersDrawnThisRound) == numPlayers {
			if err := r.NextRound(); err != nil {
				return nil, err
			}
			return r.getCurrentDrawer(), nil
		}

	}
}

// Start a new round and reset player statuses.
func (r *Round) NextRound() error {
	// Check if the game has reached its maximum rounds.
	if r.Count >= r.Game.Options.MaxRounds {
		r.Game.Status = StatusFinished
		log.Println("Game over!")
		return fmt.Errorf("game over")
	}

	r.Count++
	r.PlayersDrawnThisRound = make(map[string]struct{})
	log.Printf("Starting round %d", r.Count)

	// Reset player statuses.
	for _, player := range r.Game.Players {
		player.HasDrawn = false
		player.IsDrawing = false
	}
	return nil
}

// Get the current drawer.
func (r *Round) getCurrentDrawer() *Player {
	if len(r.Game.Players) == 0 {
		return nil
	}
	return r.Game.Players[r.CurrentDrawerIdx]
}
