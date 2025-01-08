package game

import (
	"context"
	"log"
	"time"
)

type Round struct {
	Timer                 *time.Timer
	Count                 int
	CurrentDrawerIdx      int
	WordToGuess           string
	IsActive              bool
	PlayersDrawnThisRound map[string]struct{}
	timerCtx              context.Context
	timerCancel           context.CancelFunc
	Game                  *Game
}

func (r *Round) NextRound() {
	r.Count++
	r.PlayersDrawnThisRound = make(map[string]struct{})
}

func (r *Round) NextDrawer(totalPlayers int) {
	r.CurrentDrawerIdx = (r.CurrentDrawerIdx + 1) % totalPlayers
}

func (r *Round) StartTimer(duration time.Duration, onExpire func()) {
	// Create a fresh context & cancel function
	r.timerCtx, r.timerCancel = context.WithCancel(context.Background())

	// Start the countdown with that context
	countdownCh := Countdown(r.timerCtx, duration)

	// Read countdown values in a goroutine
	go func() {
		for secondsLeft := range countdownCh {
			// For each tick, you might want to broadcast/log the time left
			log.Printf("Round %d countdown: %d seconds remaining", r.Count, secondsLeft)
			// or Hub.Broadcast(...) etc.

		}

		onExpire()
	}()
}

func Countdown(ctx context.Context, duration time.Duration) <-chan int {
	ch := make(chan int)

	go func() {
		defer close(ch)

		secondsRemaining := int(duration.Seconds())
		for secondsRemaining > 0 {
			select {
			case <-ctx.Done():
				// The countdown was canceled externally
				return
			default:
				// Not canceled yet, send the countdown value
				ch <- secondsRemaining
				time.Sleep(1 * time.Second)
				secondsRemaining--
			}
		}
	}()

	return ch
}

func (r *Round) StopTimer() {
	// If we have a valid cancel func, call it
	if r.timerCancel != nil {
		log.Println("Stopping the round timer early!")
		r.timerCancel() // triggers ctx.Done() in the Countdown
		r.timerCancel = nil
	}
}
