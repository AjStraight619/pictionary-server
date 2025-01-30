package game

import (
	"encoding/json"
	"log"
)

func (g *Game) ToLoggable() map[string]any {
	players := []map[string]any{}
	for _, player := range g.Players {
		players = append(players, map[string]any{
			"Id":        player.Id,
			"Username":  player.Username,
			"IsLeader":  player.IsLeader,
			"IsDrawing": player.IsDrawing,
			"HasDrawn":  player.HasDrawn,
		})
	}

	var round map[string]any
	if g.Round != nil {
		round = map[string]any{
			"Count":            g.Round.Count,
			"CurrentDrawerIdx": g.Round.CurrentDrawerIdx,
			"WordToGuess":      g.CurrentTurn.Word,
			"PlayersDrawn":     len(g.Round.PlayersDrawnThisRound),
		}
	} else {
		round = nil // No active round
	}

	return map[string]any{
		"Id":               g.Id,
		"Players":          players,
		"Status":           g.Status,
		"Options":          g.Options,
		"Round":            round,
		"Selectable_Words": g.SelectableWords,
	}
}

func (g *Game) Print() {
	loggable := g.ToLoggable()

	// Marshal with indentation for logging
	jsonData, err := json.MarshalIndent(loggable, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal loggable data: %v", err)
		return
	}

	// Print the formatted JSON
	log.Println(string(jsonData))
}
