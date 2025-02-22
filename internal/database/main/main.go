package main

import (
	"encoding/json"
	"fmt"
	"github.com/Ajstraight619/pictionary-server/internal/database"
	"github.com/Ajstraight619/pictionary-server/internal/database/models"
	"log"
	"os"
)

func main() {
	databasePath := "data/game.db"

	os.MkdirAll("data", os.ModePerm)
	database.InitDB(databasePath)

	database.MigrateModels(&models.Word{})

	data, err := os.ReadFile("internal/database/words.json")

	if err != nil {
		fmt.Printf("Error reading data from json file: %v", err)
		return
	}

	var wordsMap map[string][]string

	if err := json.Unmarshal(data, &wordsMap); err != nil {
		log.Fatalf("Failed to parse JSON file: %v", err)
	}

	var words []models.Word

	for category, wordslist := range wordsMap {
		for _, word := range wordslist {
			words = append(words, models.Word{
				Word:     word,
				Category: category,
			})
		}

	}

	if err := database.DB.CreateInBatches(words, 100).Error; err != nil {
		log.Fatalf("Failed to insert words into database: %v", err)
	}

}
