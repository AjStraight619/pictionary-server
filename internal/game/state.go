package game

import (
	"encoding/json"
	"log"

	"github.com/Ajstraight619/pictionary-server/internal/shared"
)

type GameState struct {
	ID              string             `json:"id"`
	Players         []*shared.Player   `json:"players"`
	PlayerOrder     []string           `json:"playerOrder"`
	CurrentDrawerID string             `json:"currentDrawerID"`
	Options         shared.GameOptions `json:"options"`
	Status          Status             `json:"status"`
	Round           *Round             `json:"round"`
	Turn            *Turn              `json:"turn"`
	WordToGuess     *shared.Word       `json:"wordToGuess,omitempty"`
	SelectableWords []shared.Word      `json:"selectableWords,omitempty"`
	IsSelectingWord bool               `json:"isSelectingWord"`
}

func (g *Game) GetGameState() GameState {
	g.Mu.RLock()
	defer g.Mu.RUnlock()
	orderedPlayers := make([]*shared.Player, 0, len(g.PlayerOrder))
	for _, id := range g.PlayerOrder {
		if player, exists := g.Players[id]; exists {
			orderedPlayers = append(orderedPlayers, player)
		}
	}
	return GameState{
		ID:              g.ID,
		Players:         orderedPlayers,
		PlayerOrder:     g.PlayerOrder,
		CurrentDrawerID: g.Round.CurrentDrawerID,
		Options:         g.Options,
		Status:          g.Status,
		Round:           g.Round,
		Turn:            g.CurrentTurn,
		SelectableWords: g.SelectableWords,
		IsSelectingWord: g.isSelectingWord,
	}
}

func (g *Game) BroadcastGameState() {
	state := g.GetGameState()
	b, err := json.Marshal(map[string]interface{}{
		"type":    "gameState",
		"payload": state,
	})
	if err != nil {
		log.Println("error marshalling game state:", err)

	}
	g.Messenger.BroadcastMessage(b)
}

func (g *Game) String() string {
	state := g.GetGameState()
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return "Error marshalling game state: " + err.Error()
	}
	return string(b)
}
