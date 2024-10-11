// metrics/prometheus_metrics.go
package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Define Prometheus metrics
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests",
		Buckets: prometheus.DefBuckets,
	},
		[]string{"method", "endpoint"})
)

// Initialize Prometheus metrics
func initPrometheusMetrics() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

// Prometheus middleware to track request count and duration
func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timer := prometheus.NewTimer(httpRequestDuration.WithLabelValues(r.Method, r.URL.Path))
		defer timer.ObserveDuration()

		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path).Inc()
		next.ServeHTTP(w, r)
	})
}

// Handle Prometheus metrics endpoint
func handlePrometheusMetrics() {
	http.Handle("/metrics", promhttp.Handler())
}
