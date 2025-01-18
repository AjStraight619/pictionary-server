package database_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/Ajstraight619/pictionary-server/internal/database"
	"github.com/Ajstraight619/pictionary-server/internal/database/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

var DB *gorm.DB

func getAbsPathToDb() (string, error) {

	databasePath, err := filepath.Abs("../../data/game.db")
	if err != nil {
		return "", fmt.Errorf("Failed to resolve absolute path for database: %v", err)
	}

	return databasePath, nil

}

func TestLoadWords(t *testing.T) {
	// Compute the absolute path for the database file

	databasePath, err := getAbsPathToDb()

	if err != nil {
		t.Fatalf("Error getting db path: %v", err)
	}

	// Log the resolved path for debugging
	t.Logf("Resolved database path: %s", databasePath)

	// Initialize the database
	database.InitDB(databasePath)

	// Query the database
	var words []models.Word
	if err := database.DB.Where("category = ?", "Animals").Find(&words).Error; err != nil {
		t.Fatalf("Error getting words from db: %v", err)
	}

	// Assert that words were loaded
	assert.NotEqual(t, len(words), 0, "Expected words, got none")
}

func TestLoadAllWords(t *testing.T) {

	databasePath, err := getAbsPathToDb()

	if err != nil {
		t.Fatalf("Error getting db path: %v", err)
	}

	database.InitDB(databasePath)

	var words []models.Word

	if err := database.DB.Find(&words).Error; err != nil {
		t.Fatalf("Error getting all words from db: %v", err)
	}

	assert.NotEqual(t, len(words), 0, "Expected words, got none")

}
