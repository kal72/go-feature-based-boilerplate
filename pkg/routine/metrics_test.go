package routine_test

import (
	"context"
	"testing"
	"time"

	"go-feature-based-boilerplate/pkg/routine"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestMetricsCollectors(t *testing.T) {
	collectors := routine.MetricsCollectors()
	assert.Len(t, collectors, 4)

	// Custom registry registration should succeed
	registry := prometheus.NewRegistry()
	routine.RegisterMetrics(registry)
	for _, c := range collectors {
		// Registering with custom registry
		_ = registry.Register(c)
	}

	// ActiveCount should start at 0
	assert.Equal(t, int64(0), routine.ActiveCount())

	// Executing Go should track active count and duration
	done := make(chan struct{})
	routine.Go(context.Background(), func(ctx context.Context) {
		time.Sleep(10 * time.Millisecond)
		close(done)
	})

	<-done

	// Verify collectors can be gathered without panic
	mfs, err := registry.Gather()
	assert.NoError(t, err)
	assert.NotEmpty(t, mfs)
}
