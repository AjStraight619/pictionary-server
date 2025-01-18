package game

import (
	"log"

	"github.com/Ajstraight619/pictionary-server/internal/database"
	"github.com/Ajstraight619/pictionary-server/internal/database/models"
	"gorm.io/gorm"
)

var DB *gorm.DB

func (g *Game) GetRandomWord(category string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	var word models.Word
	if err := database.DB.Where("category = ?", category).Order("RANDOM()").First(&word).Error; err != nil {
		return err
	}

	g.Round.WordToGuess = word.Word
	g.UsedWords = append(g.UsedWords, word.Word)

	return nil
}

func (g *Game) GetRandomWords(category string, count int) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	var words []models.Word
	query := database.DB.Order("RANDOM()").Limit(count) // Base query with random ordering and limit

	// Add category filter if provided
	if category != "" {
		query = query.Where("category = ?", category)
	}

	// Execute the query
	if err := query.Find(&words).Error; err != nil {
		log.Printf("Error fetching random words: %v", err)
		return err
	}

	// Log and append the fetched words
	for _, word := range words {
		log.Printf("Random word fetched: %s", word.Word)
		g.SelectableWords = append(g.SelectableWords, word)
	}

	if len(words) == 0 {
		log.Println("No words found for the given category or empty result set")
	}

	return nil
}

func (g *Game) SetWord(word string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.Round.WordToGuess = word
	g.UsedWords = append(g.UsedWords, word)

	updatedGameState := g.GetGameState()

	// Broadcast the updated game state to all players

	message := BroadcastMessage{
		Type:    "game_state",
		Payload: updatedGameState,
	}

	if err := g.BroadcastToAll(message); err != nil {
		log.Printf("Failed to broadcast game state: %v", err)
	}

	currentDrawer := g.Round.getCurrentDrawer()

	closeModalMessage := PlayerMessage{
		PlayerId: currentDrawer.Id,
		Type:     "close_select_word_modal",
		Payload:  map[string]interface{}{},
	}

	g.SendMessageToPlayer(currentDrawer.Id, closeModalMessage)

	return nil
}
