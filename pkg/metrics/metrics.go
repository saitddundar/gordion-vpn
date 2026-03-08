package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gordion_grpc_requests_total",
			Help: "Total number of gRPC requests",
		},
		[]string{"service", "method", "status"},
	)

	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gordion_grpc_request_duration_seconds",
			Help:    "gRPC request duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"service", "method"},
	)

	ActiveConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gordion_active_connections",
			Help: "Number of active gRPC connections",
		},
		[]string{"service"},
	)

	DBQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gordion_db_queries_total",
			Help: "Total database queries executed",
		},
		[]string{"service", "operation", "status"},
	)

	DBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gordion_db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
		[]string{"service", "operation"},
	)

	// --- Domain-Specific Business Metrics ---

	ActivePeers = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gordion_active_peers_total",
			Help: "Current number of online peers in the VPN network",
		},
		[]string{"is_exit_node"},
	)

	AllocatedIPs = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gordion_allocated_ips_total",
			Help: "Total number of IP addresses currently allocated to peers",
		},
	)

	TokensIssuedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "gordion_jwt_tokens_issued_total",
			Help: "Total number of JWT authentication tokens successfully issued",
		},
	)

	AuthFailuresTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gordion_auth_failures_total",
			Help: "Total number of failed authentication attempts",
		},
		[]string{"reason"}, // e.g., "invalid_network_secret", "expired_token", "invalid_signature"
	)
)
