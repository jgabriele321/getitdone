package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/giovannigabriele/go-todo-bot/internal/config"
	"github.com/giovannigabriele/go-todo-bot/internal/cron"
	"github.com/giovannigabriele/go-todo-bot/internal/health"
	"github.com/giovannigabriele/go-todo-bot/internal/llm"
	"github.com/giovannigabriele/go-todo-bot/internal/sheets"
	"github.com/giovannigabriele/go-todo-bot/internal/telegram"
)

func main() {
	// Configure logging
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	log.Info().Msg("Starting TODO Bot")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Create context that will be canceled on interrupt
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

	// Create and start cron manager
	cronManager := cron.NewManager()
	cronManager.Start()
	defer cronManager.Stop()

	// Create HTTP server for health check
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health.Handler())

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	// Start HTTP server
	go func() {
		log.Info().Str("port", cfg.Port).Msg("Starting HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("HTTP server error")
		}
	}()

	// Start bot in a goroutine
	go func() {
		if err := telegramHandler.Start(ctx); err != nil {
			log.Error().Err(err).Msg("Bot error")
			cancel()
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	log.Info().Msg("Shutting down...")

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server shutdown error")
	}

	// Cancel context to stop bot
	cancel()
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
