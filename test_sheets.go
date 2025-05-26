package main

import (
	"context"
	"fmt"
	"os"

	"github.com/giovannigabriele/go-todo-bot/internal/config"
	"github.com/giovannigabriele/go-todo-bot/internal/sheets"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Set up logging
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	fmt.Printf("Google Script URL: %s\n", cfg.GoogleScriptURL)

	// Create sheets client
	client := sheets.NewClient(cfg)

	// Create test task
	testTasks := []sheets.Task{
		{
			People:      []string{"johnny"},
			Summary:     "Test task from direct client test",
			FullMessage: "This is a test message to debug the sheets client",
			BotNotes:    "Direct test - debugging",
		},
	}

	// Try to add tasks
	ctx := context.Background()
	fmt.Println("About to call AddTasks...")

	if err := client.AddTasks(ctx, testTasks); err != nil {
		fmt.Printf("ERROR: %v\n", err)
		log.Error().Err(err).Msg("AddTasks failed")
	} else {
		fmt.Println("SUCCESS: Tasks added successfully")
		log.Info().Msg("Tasks added successfully")
	}
}
