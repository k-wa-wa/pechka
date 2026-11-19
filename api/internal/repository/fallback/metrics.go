package fallback

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsOnce   sync.Once
	fallbackTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pechka_api_fallback_total",
			Help: "Total number of fallback queries to PostgreSQL due to primary datastore failure or open circuit breaker.",
		},
		[]string{"backend", "operation"},
	)
)

// RegisterMetrics registers fallback metrics with the provided Prometheus registerer.
func RegisterMetrics(reg prometheus.Registerer) {
	if reg == nil {
		return
	}
	metricsOnce.Do(func() {
		// Ignore AlreadyRegisteredError to allow multiple registrations in test suites.
		_ = reg.Register(fallbackTotal)
	})
}

// RecordFallback increments the fallback counter for the specified backend and operation.
func RecordFallback(backend, operation string) {
	fallbackTotal.WithLabelValues(backend, operation).Inc()
}

// FallbackCounterVec returns the Prometheus CounterVec for inspection/testing.
func FallbackCounterVec() *prometheus.CounterVec {
	return fallbackTotal
}
