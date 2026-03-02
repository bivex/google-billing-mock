package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	apimetrics "github.com/bivex/google-billing-mock/internal/infrastructure/metrics"
)

// Metrics records Prometheus request duration per method/path/status.
func Metrics(m *apimetrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)
			m.RequestDuration.WithLabelValues(
				r.Method,
				r.URL.Path,
				strconv.Itoa(rw.status),
			).Observe(time.Since(start).Seconds())
		})
	}
}

// StatusText maps status codes to a short label for metrics.
func StatusText(code int) string {
	return fmt.Sprintf("%d", code)
}
