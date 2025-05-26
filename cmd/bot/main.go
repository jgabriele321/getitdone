package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/giovannigabriele/go-todo-bot/internal/config"
	"github.com/giovannigabriele/go-todo-bot/internal/llm"
	"github.com/giovannigabriele/go-todo-bot/internal/sheets"
	"github.com/giovannigabriele/go-todo-bot/internal/telegram"
)

func main() {
	// Setup logging
	setupLogging()

	log.Info().Msg("Starting TODO Bot")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize clients
	llmClient := llm.NewClient(cfg.OpenRouterAPIKey)
	sheetsClient := sheets.NewClient(cfg.GoogleScriptURL)

	// Initialize Telegram handler
	telegramHandler, err := telegram.NewHandler(cfg.TelegramToken, llmClient, sheetsClient)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Telegram handler")
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start bot in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := telegramHandler.Start(ctx); err != nil && err != context.Canceled {
			errChan <- err
		}
	}()

	log.Info().Msg("TODO Bot is running. Press Ctrl+C to stop.")

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
	case err := <-errChan:
		log.Error().Err(err).Msg("Bot error")
	}

	// Graceful shutdown
	log.Info().Msg("Shutting down...")
	cancel()

	// Give some time for cleanup
	time.Sleep(2 * time.Second)
	log.Info().Msg("Shutdown complete")
}

// setupLogging configures the logger
func setupLogging() {
	// Configure zerolog
	zerolog.TimeFieldFormat = time.RFC3339

	// Use console writer for development
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "15:04:05",
	})

	// Set log level based on environment
	if os.Getenv("ENVIRONMENT") == "production" {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Info().Msg("Logging configured")
}
