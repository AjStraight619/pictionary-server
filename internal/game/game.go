package game

import (
	"fmt"
	m "github.com/Ajstraight619/pictionary-server/internal/database/models"
	ws "github.com/Ajstraight619/pictionary-server/internal/websocket"
	"log"
	"math"
	"sync"
	"sync/atomic"
	"time"
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
	mu                      sync.Mutex          `json:"-"`
	Id                      string              `json:"id"`
	Hub                     *ws.Hub             `json:"-"`
	Players                 []*Player           `json:"players"`
	playerIds               map[string]struct{} `json:"-"`
	TempDisconnectedPlayers map[string]*Player  `json:"-"`
	Round                   *Round              `json:"round,omitempty"`
	ExperationTimer         time.Timer          `json:"-"`
	Status                  GameStatus          `json:"status"`
	Options                 GameOptions         `json:"options"`
	UsedWords               []string            `json:"-"`
	SelectableWords         []m.Word            `json:"selectableWords"`
	StartGameTimer          *Timer              `json:"-"`
	ActiveTimer             string              `json:"-"`
	CurrentTurn             *Turn               `json:"turn,omitempty"`
}

func (g *Game) GetGameState() map[string]any {
	// g.mu.Lock()
	// defer g.mu.Unlock()

	players := []Player{}
	for _, player := range g.Players {
		players = append(players, Player{
			Id:        player.Id,
			Username:  player.Username,
			IsLeader:  player.IsLeader,
			IsDrawing: player.IsDrawing,
			Score:     player.Score,
			Color:     player.Color,
		})
	}

	selectableWords := []m.Word{}
	for _, word := range g.SelectableWords {
		selectableWords = append(selectableWords, m.Word{
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
	type GameState struct {
		Id              string      `json:"id"`
		Players         []Player    `json:"players"`
		Status          GameStatus  `json:"status"`
		Options         GameOptions `json:"options"`
		Round           JSONRound   `json:"round"`
		SelectableWords []m.Word    `json:"selectableWords"`
		WordToGuess     string      `json:"wordToGuess"`
		RevealedLetters []rune      `json:"revealedLetters"`
	}

	// Construct the JSON-ready game state
	gameState := GameState{
		Id:              g.Id,
		Players:         players,
		Status:          g.Status,
		Options:         g.Options,
		Round:           roundDetails,
		SelectableWords: selectableWords,
		WordToGuess: func() string {
			if g.CurrentTurn != nil {
				return g.CurrentTurn.Word
			}
			return ""
		}(),
		RevealedLetters: func() []rune {
			if g.CurrentTurn != nil {
				return g.CurrentTurn.RevealedLetters
			}
			return []rune{}
		}(),
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

	g.SendMessageToPlayer(firstDrawer.Id, PlayerMessage{
		PlayerId: firstDrawer.Id,
		Type:     "openSelectWordModal",
		Payload: struct {
			SelectableWords []m.Word `json:"selectableWords"`
		}{
			SelectableWords: g.SelectableWords,
		},
	})

	g.Delay(3 * time.Second)

	// Initialize the current turn
	g.CurrentTurn = NewTurn(firstDrawer, g)

	// TODO: Send update turn message to all players.

	// Start the select word timer for the turn
	g.CurrentTurn.StartSelectWordTimer(time.Second*time.Duration(g.Options.SelectWordTimer), func() {
		log.Printf("Select word timer ended")

		log.Printf("Selectable words: %v", g.SelectableWords)

		if g.CurrentTurn.Word == "" && len(g.SelectableWords) > 0 {
			word := g.SelectableWords[0].Word
			g.SetWord(word)
		}
	})

	return nil
}

func (g *Game) HandlePlayerDisconnect(playerId string) {
	removedPlayer := g.RemovePlayer(playerId)
	if removedPlayer == nil {
		log.Printf("Player with ID %s not found during disconnect handling", playerId)
		return
	}

	// Add the player to the temporary disconnected players map
	g.mu.Lock()
	g.TempDisconnectedPlayers[playerId] = removedPlayer
	g.mu.Unlock()

	// Start the disconnection timer
	removedPlayer.StartDisconnectionTimer(30*time.Second, func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		delete(g.TempDisconnectedPlayers, playerId)
		log.Printf("Player %s permanently removed after disconnection timeout.", playerId)

		log.Println("----Game state after player disconnect----")
		g.Print()
	})
}

func (g *Game) HandleWordSelect(word string) {
	log.Printf("Word selected: %s", word)
	g.SetWord(word)
}

func (g *Game) HandlePlayerReconnect(playerId string) {
	g.mu.Lock()

	var reconnectedPlayer *Player

	// Check if the player is in TempDisconnectedPlayers
	if player, found := g.TempDisconnectedPlayers[playerId]; found {
		reconnectedPlayer = player
		delete(g.TempDisconnectedPlayers, playerId) // Remove from TempDisconnectedPlayers
		g.Players = append(g.Players, player)       // Add back to active players
		log.Printf("Player %s reconnected and added back to the game.", playerId)
	} else {
		// Check if the player is already in the active players list
		for _, player := range g.Players {
			if player.Id == playerId {
				reconnectedPlayer = player
				log.Printf("Player %s reconnected but was already in the game.", playerId)
				break
			}
		}
	}

	if reconnectedPlayer == nil {
		log.Printf("Player with ID %s not found in game or TempDisconnectedPlayers.", playerId)
		return
	}

	reconnectedPlayer.StopDisconnectionTimer()

	g.mu.Unlock()

	// NOTE: Need to handle removing players differently

	// // Broadcast updated game state
	// updatedGameState := g.GetGameState()
	// if err := g.BroadcastToAll(BroadcastMessage{
	// 	Type:    "game_state",
	// 	Payload: updatedGameState,
	// }); err != nil {
	// 	log.Printf("Failed to broadcast updated game state: %v", err)
	// }
	//
	// log.Println("----Game state after player reconnect----")
	g.Print()
}

func (g *Game) RemovePlayer(playerId string) *Player {
	g.mu.Lock()

	var removedPlayer *Player
	playerFound := false

	// Find and remove the player
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
		return nil
	}

	g.mu.Unlock() // Unlock before updating the game state avoid deadlock

	updatedGameState := g.GetGameState()

	// Send a toast message to the clients
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

	if err := g.BroadcastToAll(message); err != nil {
		log.Printf("Failed to broadcast player left message: %v", err)
	}

	// Broadcast the updated game state

	if err := g.BroadcastToAll(BroadcastMessage{
		Type:    "game_state",
		Payload: updatedGameState,
	}); err != nil {
		log.Printf("Failed to broadcast updated game state: %v", err)
	}

	return removedPlayer
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

func levenshteinDistance(s1, s2 string) int {
	m, n := len(s1), len(s2)
	dp := make([][]int, m+1)

	// Initialize the DP table
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	// Base cases
	for i := 0; i <= m; i++ {
		dp[i][0] = i
	}
	for j := 0; j <= n; j++ {
		dp[0][j] = j
	}

	// Fill the DP table
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if s1[i-1] == s2[j-1] {
				dp[i][j] = dp[i-1][j-1]
			} else {
				dp[i][j] = 1 + int(math.Min(math.Min(
					float64(dp[i-1][j]), // Deletion
					float64(dp[i][j-1]), // Insertion
				), float64(dp[i-1][j-1]))) // Substitution
			}
		}
	}

	return dp[m][n]
}

func (g *Game) CalculateScore(player *Player) {

	// Prevent calculating score multiple times
	if player.HasGuessedCorrect {
		return
	}

	maxTime := g.Options.TurnTimer
	remainingTime := g.CurrentTurn.GuessTimer.GetRemainingTime()

	const baseScore = 100      // Fixed base score
	const timeBonusFactor = 50 // Maximum bonus points for speed

	// Calculate time bonus
	timeBonus := int(float64(remainingTime) / float64(maxTime) * float64(timeBonusFactor))

	totalScore := baseScore + timeBonus

	player.Score += int16(totalScore)

	message := BroadcastMessage{
		Type: "scoreUpdate",
		Payload: struct {
			PlayerId string `json:"playerId"`
			Score    int16  `json:"score"`
		}{
			PlayerId: player.Id,
			Score:    player.Score,
		},
	}
	if err := g.BroadcastToAll(message); err != nil {
		log.Printf("Failed to broadcast score update: %v", err)
	}

}

func (g *Game) StopGameTimer() {
	if g.StartGameTimer != nil {
		g.StartGameTimer.Stop()
		g.StartGameTimer = nil
	}
}

func (g *Game) Delay(duration time.Duration) {
	go func() {
		time.Sleep(duration)
	}()
}
