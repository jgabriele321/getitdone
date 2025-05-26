package cron

import (
	"time"

	"github.com/giovannigabriele/go-todo-bot/internal/config"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

// Manager handles scheduled tasks
type Manager struct {
	cron   *cron.Cron
	config *config.Config
}

// NewManager creates a new cron manager
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		cron:   cron.New(cron.WithLocation(time.UTC)),
		config: cfg,
	}
}

// Start begins the cron scheduler
func (m *Manager) Start() {
	log.Info().Msg("Starting cron scheduler...")

	// Schedule daily digest email at 06:00 GMT
	_, err := m.cron.AddFunc("0 6 * * *", m.sendDailyDigest)
	if err != nil {
		log.Error().Err(err).Msg("Failed to schedule daily digest")
	}

	// For testing, you can uncomment this to run every minute
	// _, _ = m.cron.AddFunc("@every 1m", m.sendDailyDigest)

	m.cron.Start()
	log.Info().Msg("Cron scheduler started")
}

// Stop halts the cron scheduler
func (m *Manager) Stop() {
	log.Info().Msg("Stopping cron scheduler...")
	ctx := m.cron.Stop()
	<-ctx.Done()
	log.Info().Msg("Cron scheduler stopped")
}

// sendDailyDigest sends the daily email digest
func (m *Manager) sendDailyDigest() {
	log.Info().Msg("Sending daily digest...")
	// TODO: Implement email sending logic in Phase 2
}
