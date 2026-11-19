package fallback_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/k-wa-wa/pechka/api/internal/repository/fallback"
)

func TestMetrics_RegistrationAndRecording(t *testing.T) {
	reg := prometheus.NewRegistry()
	fallback.RegisterMetrics(reg)

	// Nil registerer should not panic
	fallback.RegisterMetrics(nil)

	// Verify fallback total metric increments
	backend := "mongo_test"
	operation := "list_test"

	getVal := func() float64 {
		var m dto.Metric
		if err := fallback.FallbackCounterVec().WithLabelValues(backend, operation).Write(&m); err != nil {
			t.Fatalf("failed to read metric: %v", err)
		}
		return m.GetCounter().GetValue()
	}

	before := getVal()
	fallback.RecordFallback(backend, operation)
	after := getVal()

	if after-before != 1 {
		t.Errorf("expected counter increment by 1, before=%f, after=%f", before, after)
	}
}
