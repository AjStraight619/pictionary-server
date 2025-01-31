package game

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// Message types for consistency
const (
	MessageTypePlayerReconnected = "playerReconnected"
	MessageTypePlayerJoined      = "playerJoined"
	MessageTypePlayerLeft        = "playerLeft"
	MessageTypeYourTurn          = "yourTurn"
	MessageTypeGameUpdate        = "gameUpdate"
	MessageTypeRoundStarted      = "roundStarted"
)

// BroadcastMessage represents a message sent to all players
type BroadcastMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// PlayerMessage represents a message targeted to a specific player
type PlayerMessage struct {
	PlayerId string `json:"playerId"`
	Type     string `json:"type"`
	Payload  any    `json:"payload"`
}

// Payloads for different message types
type PlayerReconnectedPayload struct {
	PlayerId  string `json:"playerId"`
	Username  string `json:"username"`
	IsLeader  bool   `json:"isLeader"`
	IsDrawing bool   `json:"isDrawing"`
	Score     int16  `json:"score"`
}

type PlayerLeftPayload struct {
	PlayerId string `json:"playerId"`
	Username string `json:"username"`
}

type RoundStartedPayload struct {
	RoundNumber int    `json:"roundNumber"`
	DrawerId    string `json:"drawerId"`
}

type YourTurnPayload struct {
	Info string `json:"info"`
}

func (g *Game) SendMessageToPlayer(playerId string, message any) error {
	for client := range g.Hub.Clients {
		if client.PlayerId == playerId {
			jsonData, err := json.Marshal(message)
			if err != nil {
				return fmt.Errorf("failed to marshal message: %w", err)
			}

			select {
			case client.Send <- jsonData:
				log.Printf("Message sent to player %s: %s", playerId, jsonData)
				return nil
			case <-time.After(500 * time.Millisecond): // Optional timeout
				log.Printf("Send channel for player %s is full, skipping message", playerId)
				return fmt.Errorf("send channel for player %s is full", playerId)
			}
		}
	}

	return fmt.Errorf("player with ID %s not found in game %s", playerId, g.Id)
}

func (g *Game) BroadcastToAll(message any) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	g.Hub.Broadcast <- jsonData
	log.Printf("Broadcast message sent: %s", string(jsonData))
	return nil
}

func (g *Game) HandleTimerStartMessages(payload map[string]interface{}) {
	timerTypeVal, ok := payload["timerType"].(string)
	if !ok {
		log.Println("timerType is missing or not a string")
		return
	}

	switch timerTypeVal {
	case "guessWordTimer":
		g.HandleGuessWordCountdown()
	case "startGameTimer":
		g.HandleStartGameCountdown()
	default:
		log.Printf("Unknown timerType: %s", timerTypeVal)
	}
}

func (g *Game) HandleTimerStopMessages(payload map[string]interface{}) {
	timerTypeVal, ok := payload["timerType"].(string)
	if !ok {
		log.Println("timerType is missing or not a string")
		return
	}

	switch timerTypeVal {
	case "guessWordTimer":
		g.CurrentTurn.StopGuessTimer()
	case "selectWordTimer":
		g.CurrentTurn.StopSelectWordTimer()
	case "startGameTimer":
		g.StopGameTimer()

	default:
		log.Printf("Unknown timerType: %s", timerTypeVal)
	}
}
