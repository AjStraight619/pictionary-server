package game

import (
	"time"
)

type Player struct {
	Id             string
	Username       string
	IsLeader       bool
	IsDrawing      bool
	IsGuessCorrect bool
	Score          int16
	Color          string
	HasDrawn       bool
	Timer          *time.Timer
}

var colors = []string{
	"#FF5733", // Red-Orange
	"#33FF57", // Lime Green
	"#3357FF", // Blue
	"#FF33A6", // Pink
	"#FFFF33", // Yellow
	"#33FFF5", // Cyan
	"#A633FF", // Purple
	"#FF8C33", // Orange
}

func CreatePlayer(id string, username string, isLeader bool) *Player {

	return &Player{
		Id:       id,
		Username: username,
		IsLeader: isLeader,
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

func AssignUniqueColor(players []*Player) string {
	// Create a set of used colors
	usedColors := make(map[string]bool)
	for _, player := range players {
		if player.Color != "" {
			usedColors[player.Color] = true
		}
	}

	// Find the first available color from the predefined list
	for _, color := range colors {
		if !usedColors[color] {
			return color
		}
	}

	return ""
}
