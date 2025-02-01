package game

import (
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type Turn struct {
	Drawer          *Player
	Game            *Game
	Word            string
	RevealedLetters []rune
	GuessTimer      *Timer
	SelectWordTimer *Timer
	correctGuesses  int32
	endTurnOnce     sync.Once
}

// Create a new turn for the given drawer.
func NewTurn(drawer *Player, game *Game) *Turn {
	return &Turn{
		Drawer:          drawer,
		Game:            game,
		RevealedLetters: make([]rune, 0),
	}
}

func (t *Turn) StartTurn() {
	t.StartSelectWordTimer(30*time.Second, func() {
	})
}

// Start the select word timer.

// TODO: Remove callback and provide default behavior.
func (t *Turn) StartSelectWordTimer(duration time.Duration, onExpire func()) {
	t.SelectWordTimer = NewTimer(duration, func() {
		log.Printf("Select word timer expired for drawer %s", t.Drawer.Username)

		onExpire()
	})

	go func() {
		for secondsLeft := range t.SelectWordTimer.GetCountdownChannel() {
			message := BroadcastMessage{
				Type: "selectWordTimer",
				Payload: struct {
					TimeRemaining int `json:"timeRemaining"`
				}{
					TimeRemaining: secondsLeft,
				},
			}
			jsonData, err := json.Marshal(message)
			if err != nil {
				log.Printf("Failed to marshal select word timer message: %v", err)
				continue
			}
			t.Game.Hub.Broadcast <- jsonData
		}
	}()
	t.SelectWordTimer.Start()
}

// Stop the select word timer.
func (t *Turn) StopSelectWordTimer() {
	if t.SelectWordTimer != nil {
		t.SelectWordTimer.Stop()
		t.SelectWordTimer = nil
		log.Printf("Select word timer stopped for drawer %s", t.Drawer.Username)
	}
}

func (t *Turn) StartGuessTimer(duration time.Duration, onExpire func()) {
	t.GuessTimer = NewTimer(duration, func() {
		log.Printf("Guess timer expired for drawer %s", t.Drawer.Username)
		onExpire()
	})

	go func() {
		for secondsLeft := range t.GuessTimer.GetCountdownChannel() {
			message := BroadcastMessage{
				Type: "guessWordTimer",
				Payload: struct {
					TimeRemaining int `json:"timeRemaining"`
				}{
					TimeRemaining: secondsLeft,
				},
			}
			jsonData, err := json.Marshal(message)
			if err != nil {
				log.Printf("Failed to marshal guess timer message: %v", err)
				continue
			}
			t.Game.Hub.Broadcast <- jsonData
		}
	}()

	// Start revealing letters concurrently
	go t.revealLetters(duration)

	t.GuessTimer.Start()
	log.Printf("Guess timer started for drawer %s", t.Drawer.Username)
}

// Reveal letters during the guessing phase.
func (t *Turn) revealLetters(duration time.Duration) {
	revealInterval := duration / time.Duration(len(t.Word))

	log.Printf("Reveal interval: %v", revealInterval)

	for i := 0; i < len(t.Word); i++ {
		select {
		case <-time.After(revealInterval):
			t.RevealedLetters = append(t.RevealedLetters, rune(t.Word[i]))

			// Broadcast the revealed letters.

			log.Printf("Revealed letters: %s", string(t.RevealedLetters))
			message := BroadcastMessage{
				Type: "revealedLetter",
				Payload: struct {
					RevealedLetters string `json:"revealedLetter"`
				}{
					RevealedLetters: string(t.Word[i]),
				},
			}
			jsonData, err := json.Marshal(message)

			if err != nil {
				log.Printf("Failed to marshal revealed letters: %v", err)
				continue
			}
			t.Game.Hub.Broadcast <- jsonData
		case <-t.GuessTimer.ctx.Done():
			// Stop revealing if the timer is canceled.
			return
		}
	}
}

func (g *Game) EndCurrentTurnAndStartNext(currentDrawer *Player) {
	// Mark that the current drawer has finished their turn.
	currentDrawer.HasDrawn = true

	// Reset the guessed flag for all players for the next turn.
	for _, p := range g.Players {
		p.HasGuessedCorrect = false
	}

	// Get the next drawer via your Round logic.
	nextDrawer, err := g.Round.NextDrawer()
	if err != nil {
		log.Printf("Turn transition error or game over: %v", err)
		return
	}

	// Create a new turn for the next drawer.
	g.CurrentTurn = NewTurn(nextDrawer, g)

	g.CurrentTurn.StartSelectWordTimer(time.Second*time.Duration(g.Options.SelectWordTimer), func() {
		log.Printf("Select word timer expired for drawer %s", nextDrawer.Username)
	})
}

func (t *Turn) allPlayersGuessedCorrect() bool {
	// Total players minus the drawer
	totalPlayers := len(t.Game.Players) - 1

	// Count players who have guessed correctly
	correctGuesses := 0

	for _, player := range t.Game.Players {
		if !player.IsDrawing && player.HasGuessedCorrect {
			correctGuesses++
		}
	}

	// Return true if all non-drawing players have guessed correctly
	return correctGuesses == totalPlayers
}

// Stop the guess timer.
func (t *Turn) StopGuessTimer() {
	if t.GuessTimer != nil {
		t.GuessTimer.Stop()
		t.GuessTimer = nil
		log.Printf("Guess timer stopped for drawer %s", t.Drawer.Username)
	}
}

func (g *Game) HandlePlayerGuess(playerId string, username string, guess string) {
	// Use a local copy of the current turn.
	turn := g.CurrentTurn
	if turn == nil {
		log.Println("No current turn, ignoring guess")
		return
	}

	// If the guess timer hasn’t been started or is nil, ignore guesses.
	if turn.GuessTimer == nil || !turn.GuessTimer.isRunning {
		log.Println("Guess timer not active, ignoring guess")
		return
	}

	// The drawer should not be allowed to guess.
	if playerId == turn.Drawer.Id {
		log.Println("Drawer cannot guess the word")
		return
	}

	correctWord := turn.Word
	distance := levenshteinDistance(guess, correctWord)
	threshold := 2

	if turn.GuessTimer.isRunning {
		if guess == correctWord {
			player := g.GetPlayerById(playerId)
			if player == nil {
				log.Printf("Player with ID %s not found", playerId)
				return
			}
			g.CalculateScore(player)
			player.HasGuessedCorrect = true

			// Atomically increment the correct guess count.
			newCount := atomic.AddInt32(&turn.correctGuesses, 1)
			required := int32(len(g.Players) - 1) // all non-drawers

			if newCount >= required {
				// Ensure the transition logic runs only once.
				turn.endTurnOnce.Do(func() {
					currentDrawer := g.Round.getCurrentDrawer()
					g.EndCurrentTurnAndStartNext(currentDrawer)
				})
			}

			// Broadcast a successful guess message.
			message := BroadcastMessage{
				Type: "player_guess",
				Payload: struct {
					PlayerId string `json:"playerId"`
					Username string `json:"username"`
					Guess    string `json:"guess"`
				}{
					PlayerId: playerId,
					Username: username,
					Guess:    username + " guessed the word!",
				},
			}
			if err := g.BroadcastToAll(message); err != nil {
				log.Printf("Failed to broadcast player guess: %v", err)
			}
			return
		} else if distance <= threshold {
			// Broadcast a "close" message.
			message := BroadcastMessage{
				Type: "player_guess",
				Payload: struct {
					PlayerId string `json:"playerId"`
					Username string `json:"username"`
					Guess    string `json:"guess"`
				}{
					PlayerId: playerId,
					Username: username,
					Guess:    username + " is close!",
				},
			}
			if err := g.BroadcastToAll(message); err != nil {
				log.Printf("Failed to broadcast player guess: %v", err)
			}
		} else {
			// Broadcast the guess as is.
			message := BroadcastMessage{
				Type: "playerGuess",
				Payload: struct {
					PlayerId string `json:"playerId"`
					Username string `json:"username"`
					Guess    string `json:"guess"`
				}{
					PlayerId: playerId,
					Username: username,
					Guess:    guess,
				},
			}
			if err := g.BroadcastToAll(message); err != nil {
				log.Printf("Failed to broadcast player guess: %v", err)
			}
		}
	}
}

func (g *Game) broadcastGuess(playerId, username, messageText string) {
	msg := BroadcastMessage{
		Type: "player_guess",
		Payload: struct {
			PlayerId string `json:"playerId"`
			Username string `json:"username"`
			Guess    string `json:"guess"`
		}{
			PlayerId: playerId,
			Username: username,
			Guess:    messageText,
		},
	}
	if err := g.BroadcastToAll(msg); err != nil {
		log.Printf("Failed to broadcast guess: %v", err)
	}
}
