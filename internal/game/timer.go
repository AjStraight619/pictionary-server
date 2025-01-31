package game

import (
	"context"
	"encoding/json"
	"log"
	"time"
)

type Timer struct {
	ctx       context.Context
	cancel    context.CancelFunc
	duration  time.Duration
	countdown chan int
	onExpire  func()
	isRunning bool
	startTime time.Time
	endTime   time.Time
}

type TimerPayload struct {
	TimerType     string `json:"timerType"`
	TimeRemaining int    `json:"timeRemaining"`
}

type TimerMessage struct {
	Type    string       `json:"type"`
	Payload TimerPayload `json:"payload"`
}

func NewTimer(duration time.Duration, onExpire func()) *Timer {
	return &Timer{
		duration:  duration,
		onExpire:  onExpire,
		countdown: make(chan int),
	}
}

func (t *Timer) Start() {
	if t.isRunning {
		log.Println("Timer is already running!")
		return
	}

	t.ctx, t.cancel = context.WithCancel(context.Background())
	t.isRunning = true
	t.startTime = time.Now()
	t.endTime = t.startTime.Add(t.duration)

	go func() {
		defer close(t.countdown)
		defer func() { t.isRunning = false }()

		secondsRemaining := int(t.duration.Seconds())
		for secondsRemaining > 0 {
			select {
			case <-t.ctx.Done():
				// Timer canceled
				return
			default:
				// Send the remaining time and decrement
				t.countdown <- secondsRemaining
				time.Sleep(1 * time.Second)
				secondsRemaining--
			}
		}

		// Timer expired
		if t.onExpire != nil {
			t.onExpire()
		}
	}()
}

func (t *Timer) Stop() {
	if t.cancel != nil {
		log.Println("Stopping the timer early!")
		t.cancel()
		t.cancel = nil
		t.isRunning = false
	}
}

func (t *Timer) GetRemainingTime() int {
	if !t.isRunning {
		return 0 // Timer is not running; no time remaining
	}

	remaining := int(t.endTime.Sub(time.Now()).Seconds())
	if remaining < 0 {
		return 0 // Ensure we don't return negative values
	}

	return remaining
}

func (t *Timer) GetCountdownChannel() <-chan int {
	return t.countdown
}

func (g *Game) HandleSelectWordCountdown() {
	g.CurrentTurn.StartSelectWordTimer(time.Second*time.Duration(g.Options.SelectWordTimer), func() {
		log.Printf("Select word timer completed")
	})
}

func (g *Game) HandleGuessWordCountdown() {
	g.CurrentTurn.StartGuessTimer(time.Second*time.Duration(g.Options.TurnTimer), func() {
		log.Printf("Guess word timer completed")
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
			Type:    "gameState",
			Payload: updatedGameState,
		})

		time.Sleep(3 * time.Second)

		g.StartGame()
	})

	go func() {
		for secondsLeft := range g.StartGameTimer.GetCountdownChannel() {
			message := TimerMessage{
				Type: "startGameTimer",
				Payload: TimerPayload{
					TimeRemaining: secondsLeft,
					TimerType:     "startGameTimer",
				},
			}

			jsonData, err := json.Marshal(message)
			if err != nil {
				log.Printf("Failed to marshal JSON for start_game_timer: %v", err)
				continue
			}

			// Broadcast to all connected clients
			g.Hub.Broadcast <- jsonData

			log.Printf("Time left to start the game: %d seconds", secondsLeft)
		}
	}()

	g.StartGameTimer.Start()
}
