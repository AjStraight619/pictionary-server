package game

import (
	"encoding/json"
	"log"
	"time"
)

type Turn struct {
	Drawer          *Player
	Game            *Game
	Word            string
	Revealed        []rune
	GuessTimer      *Timer
	SelectWordTimer *Timer
}

// Create a new turn for the given drawer.
func NewTurn(drawer *Player, game *Game) *Turn {
	return &Turn{
		Drawer:   drawer,
		Game:     game,
		Revealed: make([]rune, 0),
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
				Type: "select_word_timer",
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
				Type: "guess_word_timer",
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
			t.Revealed = append(t.Revealed, rune(t.Word[i]))

			// Broadcast the revealed letters.

			log.Printf("Revealed letters: %s", string(t.Revealed))
			message := BroadcastMessage{
				Type: "revealed_letter",
				Payload: struct {
					RevealedLetters string `json:"revealed_letter"`
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

func (g *Game) EndCurrentTurnAndStartNext(player *Player) {

	player.HasDrawn = true

	// Advance to next drawer
	nextDrawer, err := g.Round.NextDrawer()
	if err != nil {
		// Possibly the game ended
		log.Printf("Game might be over or error: %v", err)
		return
	}

	// Reset guess flags
	for _, p := range g.Players {
		p.HasGuessedCorrect = false
	}

	// Create new Turn
	g.CurrentTurn = NewTurn(nextDrawer, g)

	// Possibly broadcast word choices or just
	// automatically select a word, etc.
	// Then start the turn timers, e.g.:
	g.CurrentTurn.StartSelectWordTimer(time.Duration(g.Options.SelectWordTimer), func() {
		log.Printf("Select word timer expired for drawer %s", g.CurrentTurn.Drawer.Username)

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
