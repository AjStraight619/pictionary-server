package game

import "time"

type Player struct {
	Id             string
	Username       string
	IsLeader       bool
	IsDrawing      bool
	IsGuessCorrect bool
	Score          int16
	Timer          *time.Timer
}

func CreatePlayer(id string, username string, isLeader bool) *Player {

	return &Player{
		Id:             id,
		Username:       username,
		IsLeader:       isLeader,
		IsDrawing:      false,
		IsGuessCorrect: false,
		Score:          0,
	}
}

func (p *Player) StartDisconnectionTimer(duration time.Duration, onExpire func()) {
	if p.Timer != nil {
		p.Timer.Stop()
	}
	p.Timer = time.AfterFunc(duration, onExpire)
}

func (p *Player) StopDisconnectionTimer() {
	if p.Timer != nil {
		p.Timer.Stop()
		p.Timer = nil
	}
}
