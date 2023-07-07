package workqueue

// This file provides abstractions for setting the provider (e.g., prometheus)
// of metrics.

type queueMetrics[T comparable] interface {
	add(item T)
	get(item T)
	done(item T)
	updateUnfinishedWork()
}

// GaugeMetric represents a single numerical value that can arbitrarily go up
// and down.
type GaugeMetric interface {
	Inc()
	Dec()
}

// SettableGaugeMetric represents a single numerical value that can arbitrarily go up
// and down.
type SettableGaugeMetric interface {
	Set(float64)
}

