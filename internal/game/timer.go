package game

import (
	"context"
	"log"
	"time"
)

type Timer struct {
	ctx    context.Context
	cancel context.CancelFunc

	duration  time.Duration
	countdown chan int
	onExpire  func()
	isRunning bool
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

func (t *Timer) GetCountdownChannel() <-chan int {
	return t.countdown
}
