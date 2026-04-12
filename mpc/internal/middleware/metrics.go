package middleware

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	grpcRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mpc_operations_total",
			Help: "Total number of gRPC requests to mpc service",
		},
		[]string{"method", "status"},
	)
	grpcRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mpc_request_duration_seconds",
			Help:    "Duration of gRPC requests to mpc service in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)
)
