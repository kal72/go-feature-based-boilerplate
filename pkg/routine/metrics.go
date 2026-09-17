package routine

import (
	"errors"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsOnce sync.Once

	// metricActiveGoroutines measures the number of actively running goroutines.
	metricActiveGoroutines = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "routine",
		Name:      "active_goroutines",
		Help:      "Current number of actively running goroutines managed by pkg/routine.",
	})

	// metricTasksTotal measures the cumulative number of completed tasks.
	metricTasksTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "routine",
		Name:      "tasks_total",
		Help:      "Total number of completed tasks executed via pkg/routine categorized by execution type.",
	}, []string{"type"})

	// metricPanicsTotal measures the cumulative number of recovered panics.
	metricPanicsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "routine",
		Name:      "panics_total",
		Help:      "Total number of panics intercepted and recovered by pkg/routine.",
	})

	// metricTaskDuration measures the latency distribution of executed tasks.
	metricTaskDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "routine",
		Name:      "task_duration_seconds",
		Help:      "Execution duration in seconds of tasks executed via pkg/routine.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"type"})
)

func init() {
	RegisterMetrics(prometheus.DefaultRegisterer)
}

// RegisterMetrics registers pkg/routine Prometheus collectors with the provided Registerer.
// If the collector is already registered, the existing registration is safely retained.
func RegisterMetrics(reg prometheus.Registerer) {
	if reg == nil {
		return
	}

	metricsOnce.Do(func() {
		collectors := []prometheus.Collector{
			metricActiveGoroutines,
			metricTasksTotal,
			metricPanicsTotal,
			metricTaskDuration,
		}

		for _, col := range collectors {
			if err := reg.Register(col); err != nil {
				var alreadyRegErr prometheus.AlreadyRegisteredError
				if !errors.As(err, &alreadyRegErr) {
					// Safe fallback if registered differently
				}
			}
		}
	})
}

// MetricsCollectors returns all Prometheus collectors used by pkg/routine.
// This allows custom registry setups to scrape routine telemetry independently.
func MetricsCollectors() []prometheus.Collector {
	return []prometheus.Collector{
		metricActiveGoroutines,
		metricTasksTotal,
		metricPanicsTotal,
		metricTaskDuration,
	}
}

// metricIncActive increments the active goroutines gauge.
func metricIncActive() {
	metricActiveGoroutines.Inc()
}

// metricDecActive decrements the active goroutines gauge.
func metricDecActive() {
	metricActiveGoroutines.Dec()
}

// metricIncTasks increments the completed task counter for a given task type.
func metricIncTasks(taskType string) {
	metricTasksTotal.WithLabelValues(taskType).Inc()
}

// metricIncPanics increments the recovered panics counter.
func metricIncPanics() {
	metricPanicsTotal.Inc()
}

// metricObserveDuration records the runtime duration of a task for a given task type.
func metricObserveDuration(taskType string, durationSeconds float64) {
	metricTaskDuration.WithLabelValues(taskType).Observe(durationSeconds)
}
