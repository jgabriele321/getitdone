package health

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

var startTime = time.Now()

type HealthResponse struct {
	Status    string        `json:"status"`
	Uptime    time.Duration `json:"uptime"`
	Timestamp time.Time     `json:"timestamp"`
}

// Handler returns an HTTP handler for the health check endpoint
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := HealthResponse{
			Status:    "ok",
			Uptime:    time.Since(startTime).Round(time.Second),
			Timestamp: time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error().Err(err).Msg("Failed to encode health check response")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
