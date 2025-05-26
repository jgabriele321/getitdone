package cron

import (
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

// Manager handles scheduled tasks
type Manager struct {
	cron *cron.Cron
}

// NewManager creates a new cron manager
func NewManager() *Manager {
	c := cron.New(cron.WithLocation(time.UTC))
	return &Manager{
		cron: c,
	}
}

// Start starts the cron scheduler
func (m *Manager) Start() {
	log.Info().Msg("Starting cron manager")
	m.cron.Start()
}

// Stop stops the cron scheduler
func (m *Manager) Stop() {
	log.Info().Msg("Stopping cron manager")
	ctx := m.cron.Stop()
	<-ctx.Done()
}

// AddDailyDigestJob adds a job for daily email digests (Phase 2)
func (m *Manager) AddDailyDigestJob(fn func()) error {
	// Schedule for 06:00 GMT daily
	_, err := m.cron.AddFunc("0 6 * * *", fn)
	if err != nil {
		return err
	}

	log.Info().Msg("Daily digest job scheduled for 06:00 GMT")
	return nil
}
