package game

import (
	"fmt"
	"time"
)

type Player struct {
	Id                string `json:"playerId"`
	Username          string `json:"username"`
	IsLeader          bool   `json:"isLeader"`
	IsDrawing         bool   `json:"isDrawing"`
	HasGuessedCorrect bool   `json:"hasGuessedCorrect"`
	Score             int16  `json:"score"`
	Color             string `json:"color"`
	HasDrawn          bool
	Timer             *time.Timer
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

func (p *Player) Print() {
	fmt.Printf("Player Details:\n")
	fmt.Printf("ID: %s\n", p.Id)
	fmt.Printf("Username: %s\n", p.Username)
	fmt.Printf("Is Leader: %v\n", p.IsLeader)
	fmt.Printf("Is Drawing: %v\n", p.IsDrawing)
	fmt.Printf("Has Guessed Correctly: %v\n", p.HasGuessedCorrect)
	fmt.Printf("Score: %d\n", p.Score)
	fmt.Printf("Color: %s\n", p.Color)
	fmt.Printf("Has Drawn This Round: %v\n", p.HasDrawn)
	fmt.Printf("Disconnection Timer Active: %v\n", p.Timer != nil)
}
