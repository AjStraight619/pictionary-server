package game

import (
	"encoding/json"
	"log"
	"time"
)

type Round struct {
	Count                 int
	CurrentDrawerIdx      int
	WordToGuess           string
	RevealedLetters       []rune
	IsActive              bool
	PlayersDrawnThisRound map[string]struct{}
	Game                  *Game
	selectWordTimer       *Timer
	guessWordTimer        *Timer
}

func (r *Round) NextRound() {
	r.Count++
	r.PlayersDrawnThisRound = make(map[string]struct{})
	log.Printf("Starting round %d", r.Count)
}

func (r *Round) NextDrawer(numPlayers int) {
	if numPlayers == 0 {
		return
	}

	for {
		r.CurrentDrawerIdx = (r.CurrentDrawerIdx + 1) % numPlayers

		nextPlayer := r.Game.Players[r.CurrentDrawerIdx]
		if !nextPlayer.HasDrawn {
			nextPlayer.IsDrawing = true
			log.Printf("Next drawer is %s (ID: %s)", nextPlayer.Username, nextPlayer.Id)
			return
		}

		if allPlayersHaveDrawn(r.Game.Players) {
			r.NextRound()
			return
		}
	}
}

func (r *Round) getCurrentDrawer() *Player {
	return r.Game.Players[r.CurrentDrawerIdx]
}

func (r *Round) StartSelectWordTimer(duration time.Duration, onExpire func()) {
	r.selectWordTimer = NewTimer(duration, onExpire)
	go func() {
		for secondsLeft := range r.selectWordTimer.GetCountdownChannel() {

			message := TimerMessage{
				Type: "select_word_timer",
				Payload: TimerPayload{
					TimeRemaining: secondsLeft,
				},
			}

			jsonData, err := json.Marshal(message)
			if err != nil {
				log.Printf("Failed to marshal JSON: %v", err)
				continue
			}

			r.Game.Hub.Broadcast <- jsonData

			// Log the remaining time for debugging
			log.Printf("Time left to select a word: %d seconds", secondsLeft)
		}
	}()
	r.selectWordTimer.Start()
}

func (r *Round) StartGuessWordTimer(duration time.Duration, onExpire func()) {
	r.guessWordTimer = NewTimer(duration, onExpire)
	go func() {
		for secondsLeft := range r.guessWordTimer.GetCountdownChannel() {

			message := TimerMessage{
				Type: "guess_word_timer",
				Payload: TimerPayload{
					TimeRemaining: secondsLeft,
				},
			}

			jsonData, err := json.Marshal(message)

			if err != nil {
				log.Printf("Failed to marshal JSON: %v", err)
				continue
			}

			r.Game.Hub.Broadcast <- jsonData

			log.Printf("Time left to guess the word: %d seconds", secondsLeft)
		}
	}()
	r.guessWordTimer.Start()
}

func (r *Round) StopSelectWordTimer() {
	if r.selectWordTimer != nil {
		r.selectWordTimer.Stop()
		r.selectWordTimer = nil
	}
}

func (r *Round) StopGuessWordTimer() {
	if r.guessWordTimer != nil {
		r.guessWordTimer.Stop()
		r.guessWordTimer = nil
	}
}

func allPlayersHaveDrawn(players []*Player) bool {
	for _, player := range players {
		if !player.HasDrawn {
			return false
		}
	}
	return true
}
