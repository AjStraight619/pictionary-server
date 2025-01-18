package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/Ajstraight619/pictionary-server/internal/database"
	"github.com/Ajstraight619/pictionary-server/internal/game"
	"github.com/Ajstraight619/pictionary-server/internal/handlers"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var addr = flag.String("addr", ":8000", "http service address")

func main() {
	flag.Parse()

	env := os.Getenv("APP_ENV")
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	databasePath, err := filepath.Abs("../../data/game.db")
	if err != nil {
		log.Fatalf("Failed to resolve absolute path for database: %v", err)
	}

	database.InitDB(databasePath)

	gm := game.NewGameManager()

	router := gin.Default()

	router.Use(cors.Default())

	// Register game-related routes
	handlers.RegisterGameRoutes(router, gm)

	// Start the server
	log.Printf("Listening on %s...", *addr)
	log.Fatal(router.Run(*addr))
}
