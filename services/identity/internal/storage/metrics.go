package storage

import (
	"context"
	"time"

	"github.com/saitddundar/gordion-vpn/pkg/metrics"
)

// recordDBMetrics is a helper to record database operation metrics
func recordDBMetrics(operation string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start).Seconds()

	status := "success"
	if err != nil {
		status = "error"
	}

	metrics.DBQueriesTotal.WithLabelValues("identity", operation, status).Inc()
	metrics.DBQueryDuration.WithLabelValues("identity", operation).Observe(duration)

	return err
}

// Wrapper for CreateNode
func (s *Storage) CreateNodeWithMetrics(ctx context.Context, publicKey, version string) (*Node, error) {
	var node *Node
	err := recordDBMetrics("create_node", func() error {
		var dbErr error
		node, dbErr = s.CreateNode(ctx, publicKey, version)
		return dbErr
	})
	return node, err
}
