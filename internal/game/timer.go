package game

import (
	"context"
	"log"
	"sync"
	"time"
)

type Timer struct {
	Type      string
	duration  int
	remaining int
	isRunning bool
	mu        sync.RWMutex
	cancel    context.CancelFunc
}

type TimerMessage struct {
	Type      string `json:"type"`
	Remaining int    `json:"remaining"`
}

func NewTimer(timerType string, duration int) *Timer {
	return &Timer{
		Type:      timerType,
		duration:  duration,
		remaining: duration,
		isRunning: false,
	}

}

func (t *Timer) StartCountdown(onFinish func(), onCancel func()) <-chan int {
	t.mu.Lock()
	t.isRunning = true
	t.remaining = t.duration

	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	t.mu.Unlock()

	tickCh := make(chan int, 1)

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				t.mu.Lock()
				if t.remaining <= 0 {
					t.isRunning = false
					t.mu.Unlock()
					close(tickCh)
					if onFinish != nil {
						log.Println("Timer finished")
						onFinish()
					}
					return
				}
				t.remaining--
				remaining := t.remaining
				t.mu.Unlock()
				// Non-blocking send to tickCh:
				select {
				case tickCh <- remaining:
				default:
				}
			case <-ctx.Done():
				// Timer was cancelled.
				t.mu.Lock()
				t.isRunning = false
				t.mu.Unlock()
				close(tickCh)
				if onCancel != nil {
					onCancel()
				}
				return
			}
		}
	}()

	return tickCh
}

func (g *Game) CancelTimer(timerType string) {
	g.Mu.Lock()
	defer g.Mu.Unlock()
	if timer, exists := g.timers[timerType]; exists {
		log.Printf("Cancelling timer %s", timerType)
		timer.Cancel()
		delete(g.timers, timerType)

	}
}

func (t *Timer) Cancel() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cancel != nil && t.isRunning {
		t.cancel()
		t.cancel = nil
		t.isRunning = false
		t.remaining = 0
	}
}

func (g *Game) GetRemainingTime(timerType string) int {
	g.Mu.RLock()
	defer g.Mu.RUnlock()
	if timer, exists := g.timers[timerType]; exists {
		return timer.remaining
	}
	return 0
}
