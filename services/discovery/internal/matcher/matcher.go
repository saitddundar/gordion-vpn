package matcher

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/saitddundar/gordion-vpn/services/discovery/internal/registry"
)

type Matcher struct {
	registry *registry.Registry
}

func New(registry *registry.Registry) *Matcher {
	return &Matcher{registry: registry}
}

func (m *Matcher) FindPeer(ctx context.Context, requesterID string) (*registry.Peer, error) {
	peers, err := m.registry.ListOnlinePeers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list peers: %w", err)
	}

	// Filter out the requester itself
	var candidates []*registry.Peer
	for _, p := range peers {
		if p.NodeID != requesterID {
			candidates = append(candidates, p)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available peers")
	}

	// Random selection  (i will upgrade it later)
	selected := candidates[rand.Intn(len(candidates))]
	return selected, nil
}
