package middleware

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	grpcRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "twofa_operations_total",
			Help: "Total number of gRPC requests to twofa service",
		},
		[]string{"method", "status"},
	)
	grpcRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "twofa_mpc_latency_seconds",
			Help:    "Duration of gRPC requests to twofa service in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)
)
