package game

import (
	"log"
	"time"

	"github.com/Ajstraight619/pictionary-server/internal/database"
	m "github.com/Ajstraight619/pictionary-server/internal/database/models"
	"gorm.io/gorm"
)

var DB *gorm.DB

func (g *Game) GetRandomWord(category string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	var word m.Word
	if err := database.DB.Where("category = ?", category).Order("RANDOM()").First(&word).Error; err != nil {
		return err
	}

	g.CurrentTurn.Word = word.Word
	g.UsedWords = append(g.UsedWords, word.Word)

	return nil
}

func (g *Game) GetRandomWords(category string, count int) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	var words []m.Word
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
	log.Println("Acquiring game lock in SetWord")
	g.mu.Lock()

	// Update the game state
	log.Printf("Setting word to guess: %s", word)
	g.CurrentTurn.Word = word
	g.UsedWords = append(g.UsedWords, word)
	g.SelectableWords = nil
	currentDrawer := g.Round.getCurrentDrawer()

	closeModalMessage := BroadcastMessage{
		Type:    "close_select_word_modal",
		Payload: map[string]interface{}{},
	}

	g.mu.Unlock()

	// Send the close modal message to player that selected the word

	if err := g.SendMessageToPlayer(currentDrawer.Id, closeModalMessage); err != nil {
		log.Printf("Failed to send close modal message: %v", err)
		return err
	}
	log.Println("Close modal message sent successfully")

	// Broadcast updated game state
	gameState := g.GetGameState()
	message := BroadcastMessage{
		Type:    "game_state",
		Payload: gameState,
	}
	if err := g.BroadcastToAll(message); err != nil {
		log.Printf("Failed to broadcast game state: %v", err)
	}

	g.Delay(3 * time.Second)

	g.CurrentTurn.StartGuessTimer(time.Second*time.Duration(g.Options.TurnTimer), func() {
		log.Printf("Guess word timer completed")

	})

	return nil
}

func ConvertWordsToJSON(words []m.Word) []m.JSONWord {
	jsonWords := make([]m.JSONWord, len(words))
	for i, word := range words {
		jsonWords[i] = m.JSONWord{
			Id:       word.Id,
			Word:     word.Word,
			Category: word.Category,
		}
	}
	return jsonWords
}
