package game

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Ajstraight619/pictionary-server/internal/database/models"
	ws "github.com/Ajstraight619/pictionary-server/internal/websocket"
)

type GameStatus int

const (
	StatusNotStarted GameStatus = iota
	StatusInProgress
	StatusFinished
)

type GameOptions struct {
	MaxRounds       int
	TurnTimer       int
	SelectWordTimer int
	SelectWordCount int
}

type Game struct {
	mu              sync.Mutex
	Id              string
	Hub             *ws.Hub
	Players         []*Player
	playerIds       map[string]struct{}
	Round           *Round
	ExperationTimer time.Timer
	Status          GameStatus
	Options         GameOptions
	UsedWords       []string
	SelectableWords []models.Word
	StartGameTimer  *Timer
}

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
			"WordToGuess":      g.Round.WordToGuess,
			"IsActive":         g.Round.IsActive,
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

func (g *Game) GetGameState() map[string]any {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Prepare the JSON-ready struct for players
	type JSONPlayer struct {
		Id        string `json:"playerId"`
		Username  string `json:"username"`
		IsLeader  bool   `json:"isLeader"`
		IsDrawing bool   `json:"isDrawing"`
		Score     int16  `json:"score"`
		Color     string `json:"color"`
	}

	players := []JSONPlayer{}
	for _, player := range g.Players {
		players = append(players, JSONPlayer{
			Id:        player.Id,
			Username:  player.Username,
			IsLeader:  player.IsLeader,
			IsDrawing: player.IsDrawing,
			Score:     player.Score,
			Color:     player.Color,
		})
	}

	type JSONSelectableWord struct {
		Id       uint   `json:"id"`
		Word     string `json:"word"`
		Category string `json:"category"`
	}

	selectableWords := []JSONSelectableWord{}
	for _, word := range g.SelectableWords {
		selectableWords = append(selectableWords, JSONSelectableWord{
			Id:       word.Id,
			Word:     word.Word,
			Category: word.Category,
		})
	}

	// Prepare the JSON-ready struct for the round
	type JSONRound struct {
		CurrentDrawer string `json:"currentDrawer"`
		Count         int    `json:"count"`
	}

	roundDetails := JSONRound{
		CurrentDrawer: func() string {
			if g.Round != nil && g.Round.CurrentDrawerIdx >= 0 && g.Round.CurrentDrawerIdx < len(g.Players) {
				return g.Players[g.Round.CurrentDrawerIdx].Id
			}
			return ""
		}(),
		Count: func() int {
			if g.Round != nil {
				return g.Round.Count
			}
			return 0
		}(),
	}

	// Prepare the JSON-ready struct for the game state
	type JSONGameState struct {
		Id              string               `json:"id"`
		Players         []JSONPlayer         `json:"players"`
		Status          GameStatus           `json:"status"`
		Options         GameOptions          `json:"options"`
		Round           JSONRound            `json:"round"`
		SelectableWords []JSONSelectableWord `json:"selectable_words"`
		WordToGuess     string               `json:"word_to_guess"`
	}

	// Construct the JSON-ready game state
	gameState := JSONGameState{
		Id:              g.Id,
		Players:         players,
		Status:          g.Status,
		Options:         g.Options,
		Round:           roundDetails,
		SelectableWords: selectableWords,
		WordToGuess:     g.Round.WordToGuess,
	}

	return map[string]any{
		"gameState": gameState,
	}
}

func (g *Game) AddPlayer(player *Player) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.playerIds == nil {
		g.playerIds = make(map[string]struct{})
	}

	if _, exists := g.playerIds[player.Id]; exists {
		return fmt.Errorf("player with ID %s already exists", player.Id)
	}

	player.Color = AssignUniqueColor(g.Players)

	g.Players = append(g.Players, player)
	g.playerIds[player.Id] = struct{}{}

	return nil
}

func (g *Game) GetPlayerById(id string) *Player {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, player := range g.Players {
		if id == player.Id {
			return player
		}
	}
	return nil
}

func (g *Game) StartGame() error {

	g.Round = &Round{
		Game: g, Count: 1,
		PlayersDrawnThisRound: make(map[string]struct{}),
	}

	g.Round.CurrentDrawerIdx = 0
	firstDrawer := g.Players[g.Round.CurrentDrawerIdx]
	firstDrawer.IsDrawing = true
	log.Printf("Game started with %d players, First drawer: %s (Player ID: %s)", len(g.Players), firstDrawer.Username, firstDrawer.Id)

	g.GetRandomWords("", 3)

	g.SendMessageToPlayer(firstDrawer.Id, BroadcastMessage{
		Type: "open_select_word_modal",
		Payload: struct {
			SelectableWords []models.JSONWord `json:"selectableWords"`
		}{
			SelectableWords: ConvertWordsToJSON(g.SelectableWords),
		},
	})

	go func() {
		time.Sleep(3 * time.Second)
	}()

	g.Round.IsActive = true

	g.Round.StartSelectWordTimer(time.Second*time.Duration(g.Options.SelectWordTimer), func() {
		log.Printf("Select word timer ended")

		if g.Round.WordToGuess == "" {
			word := g.SelectableWords[0].Word
			g.SetWord(word)
		}

	})

	return nil
}

func (g *Game) AdvanceToNextDrawer(onRoundComplete func()) (*Player, error) {
	totalPlayers := len(g.Players)

	if totalPlayers == 0 {
		return nil, fmt.Errorf("no players in the game")
	}

	// Mark the current drawer as having drawn
	if g.Round.IsActive {
		currentDrawer := g.Players[g.Round.CurrentDrawerIdx]
		currentDrawer.IsDrawing = false
		currentDrawer.HasDrawn = true
	}

	// Check if all players have drawn
	if allPlayersHaveDrawn(g.Players) {
		// End the current round and start a new one
		g.Round.NextRound()

		// Reset all players' HasDrawn status for the new round
		for _, player := range g.Players {
			player.HasDrawn = false
		}

		log.Printf("All players have drawn. Moving to round %d!", g.Round.Count)

		// Call the round complete handler
		onRoundComplete()

		// Stop advancing if the game has reached its maximum rounds
		if g.Round.Count >= g.Options.MaxRounds {
			g.Round.IsActive = false
			log.Println("Game over!")
			return nil, fmt.Errorf("game over")
		}
	}

	// Advance to the next drawer
	g.Round.NextDrawer(totalPlayers)

	// Set the new drawer's `IsDrawing` to true
	nextDrawer := g.Players[g.Round.CurrentDrawerIdx]
	nextDrawer.IsDrawing = true
	g.Round.IsActive = true

	// Stop any existing timers before starting a new one
	g.Round.StopGuessWordTimer()

	// Start the round timer for the new drawer
	roundDuration := time.Duration(g.Options.TurnTimer) * time.Second
	g.Round.StartGuessWordTimer(roundDuration, func() {
		log.Printf("Turn %d timer ended!", g.Round.Count)
		onRoundComplete()
	})

	return nextDrawer, nil
}

func (g *Game) HandlePlayerDisconnect(playerId string) {
	for _, player := range g.Players {
		if player.Id == playerId {
			log.Printf("Player disconnected: %s", playerId)
			player.StartDisconnectionTimer(30*time.Second, func() {
				log.Printf("Player disconnection timer expired: %s", playerId)
				g.RemovePlayer(playerId)
			})
			return
		}
	}

}

func (g *Game) HandleWordSelect(word string) {
	g.SetWord(word)
}

func (g *Game) HandlePlayerReconnect(playerId string) {
	var reconnectedPlayer *Player
	playerFound := false

	// Check if the player exists and handle reconnection
	for _, player := range g.Players {
		if player.Id == playerId {
			log.Printf("Player reconnected: %s", playerId)
			player.StopDisconnectionTimer()
			reconnectedPlayer = player
			playerFound = true
			break
		}
	}

	// If the player wasn't found in the game, log and return
	if !playerFound {
		log.Printf("Player with ID %s not found in game", playerId)
		return
	}

	message := BroadcastMessage{
		Type: "player_joined",
		Payload: struct {
			PlayerId  string `json:"playerId"`
			Username  string `json:"username"`
			IsLeader  bool   `json:"isLeader"`
			IsDrawing bool   `json:"isDrawing"`
			Score     int16  `json:"score"`
		}{
			PlayerId:  reconnectedPlayer.Id,
			Username:  reconnectedPlayer.Username,
			IsLeader:  reconnectedPlayer.IsLeader,
			IsDrawing: reconnectedPlayer.IsDrawing,
			Score:     reconnectedPlayer.Score,
		},
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message for player %s: %v", playerId, err)
		return
	}

	g.Hub.Broadcast <- jsonData
}

func (g *Game) HandleSelectWordCountdown() {
	g.Round.StartSelectWordTimer(time.Second*time.Duration(g.Options.SelectWordTimer), func() {
		log.Printf("Timer completed")
	})
}

func (g *Game) HandleGuessWordCountdown() {
	g.Round.StartGuessWordTimer(time.Second*time.Duration(g.Options.TurnTimer), func() {
		log.Printf("Timer completed")
	})
}

func (g *Game) HandleStartGameCountdown() {

	if g.StartGameTimer != nil && g.StartGameTimer.isRunning {
		log.Println("StartGameTimer is already running—aborting new timer creation.")
		return
	}

	g.StartGameTimer = NewTimer(time.Second*time.Duration(5), func() {

		g.Status = StatusInProgress

		updatedGameState := g.GetGameState()

		g.BroadcastToAll(BroadcastMessage{
			Type:    "game_state",
			Payload: updatedGameState,
		})

		time.Sleep(3 * time.Second)

		g.StartGame()

	})

	go func() {
		for secondsLeft := range g.StartGameTimer.GetCountdownChannel() {
			// Construct the message with the remaining time
			message := TimerMessage{
				Type: "start_game_timer",
				Payload: TimerPayload{
					TimeRemaining: secondsLeft,
					TimerType:     "start_game_timer",
				},
			}

			jsonData, err := json.Marshal(message)
			if err != nil {
				log.Printf("Failed to marshal JSON for start_game_timer: %v", err)
				continue
			}

			// Broadcast to all connected clients
			g.Hub.Broadcast <- jsonData

			// Optional: Log the remaining time for debugging
			log.Printf("Time left to start the game: %d seconds", secondsLeft)
		}
	}()

	g.StartGameTimer.Start()
}

func (g *Game) RemovePlayer(playerId string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	var removedPlayer *Player
	playerFound := false
	for i, player := range g.Players {
		if player.Id == playerId {
			removedPlayer = player
			playerFound = true
			g.Players = append(g.Players[:i], g.Players[i+1:]...)
			delete(g.playerIds, playerId)
			log.Printf("Player removed from game: %s", playerId)
			break
		}
	}

	if !playerFound {
		log.Printf("Player with ID %s not found in game", playerId)
		return
	}

	message := BroadcastMessage{
		Type: "player_left",
		Payload: struct {
			PlayerId string `json:"playerId"`
			Username string `json:"username"`
		}{
			PlayerId: removedPlayer.Id,
			Username: removedPlayer.Username,
		},
	}

	jsonData, err := json.Marshal(message)

	if err != nil {
		log.Printf("Error marsheling message for player %s: %v", playerId, err)
		return
	}

	g.Hub.Broadcast <- jsonData

}

func (g *Game) RunCountdownTimer(duration time.Duration, messageType string) {
	// Calculate total seconds
	totalSeconds := int(duration.Seconds())

	// Create a ticker to broadcast every second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Channel to wait until the timer finishes
	done := make(chan bool)

	go func() {
		for secondsLeft := totalSeconds; secondsLeft > 0; secondsLeft-- {
			// Broadcast the countdown to all users
			g.BroadcastToAll(BroadcastMessage{
				Type: messageType,
				Payload: struct {
					SecondsLeft int `json:"secondsLeft"`
				}{
					SecondsLeft: secondsLeft,
				},
			})

			log.Printf("Timer: %d seconds remaining", secondsLeft)

			// Wait for the next tick
			<-ticker.C
		}

		// Signal that the timer has finished
		done <- true
	}()

	// Block execution until the timer is over
	<-done
}

func (g *Game) StopGameTimer() {
	if g.StartGameTimer != nil {
		g.StartGameTimer.Stop()
		g.StartGameTimer = nil
	}
}
