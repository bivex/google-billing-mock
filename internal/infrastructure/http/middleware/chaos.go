package middleware

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// ChaosConfig holds chaos injection parameters.
type ChaosConfig struct {
	DefaultLatencyMs int
	ErrorRate        float64
}

// Chaos injects latency and random errors for resilience testing.
// Per-request overrides via X-Mock-Latency-Ms and X-Mock-Error-Rate headers.
func Chaos(cfg ChaosConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			latency := cfg.DefaultLatencyMs
			if v := r.Header.Get("X-Mock-Latency-Ms"); v != "" {
				if ms, err := strconv.Atoi(v); err == nil {
					latency = ms
				}
			}
			if latency > 0 {
				time.Sleep(time.Duration(latency) * time.Millisecond)
			}

			errorRate := cfg.ErrorRate
			if v := r.Header.Get("X-Mock-Error-Rate"); v != "" {
				if er, err := strconv.ParseFloat(v, 64); err == nil {
					errorRate = er
				}
			}
			if errorRate > 0 && rand.Float64() < errorRate {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]any{
						"code":    500,
						"message": "chaos: injected internal server error",
						"status":  "INTERNAL",
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
