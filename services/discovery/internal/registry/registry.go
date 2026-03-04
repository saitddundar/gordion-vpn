package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	peerKeyPrefix = "peer:"
	peersSetKey   = "peers:online"
)

type Peer struct {
	NodeID     string   `json:"node_id"`
	PublicKey  string   `json:"public_key"`
	Endpoint   string   `json:"endpoint"`
	Version    string   `json:"version"`
	PeerID     string   `json:"peer_id,omitempty"`
	P2PAddrs   []string `json:"p2p_addrs,omitempty"`
	IsExitNode bool     `json:"is_exit_node,omitempty"`
	LastSeen   int64    `json:"last_seen"`
}

type Registry struct {
	client *redis.Client
	ttl    time.Duration
}

func New(redisURL string, heartbeatTTL int) (*Registry, error) {
	client := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &Registry{
		client: client,
		ttl:    time.Duration(heartbeatTTL) * time.Second,
	}, nil
}

func (r *Registry) Close() error {
	return r.client.Close()
}

func (r *Registry) Ping() error {
	return r.client.Ping(context.Background()).Err()
}

func (r *Registry) Register(ctx context.Context, peer *Peer) error {
	peer.LastSeen = time.Now().Unix()

	data, err := json.Marshal(peer)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	key := peerKeyPrefix + peer.NodeID
	if err := r.client.Set(ctx, key, data, r.ttl).Err(); err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	if err := r.client.SAdd(ctx, peersSetKey, peer.NodeID).Err(); err != nil {
		return fmt.Errorf("redis sadd error: %w", err)
	}

	return nil
}

func (r *Registry) Heartbeat(ctx context.Context, nodeID string) error {
	key := peerKeyPrefix + nodeID

	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("redis exists error: %w", err)
	}
	if exists == 0 {
		return fmt.Errorf("peer not found: %s", nodeID)
	}

	if err := r.client.Expire(ctx, key, r.ttl).Err(); err != nil {
		return fmt.Errorf("redis expire error: %w", err)
	}

	return nil
}

func (r *Registry) Unregister(ctx context.Context, nodeID string) error {
	key := peerKeyPrefix + nodeID

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis del error: %w", err)
	}

	if err := r.client.SRem(ctx, peersSetKey, nodeID).Err(); err != nil {
		return fmt.Errorf("redis srem error: %w", err)
	}

	return nil
}

func (r *Registry) GetPeer(ctx context.Context, nodeID string) (*Peer, error) {
	key := peerKeyPrefix + nodeID

	data, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("peer not found: %s", nodeID)
	}
	if err != nil {
		return nil, fmt.Errorf("redis get error: %w", err)
	}

	var peer Peer
	if err := json.Unmarshal([]byte(data), &peer); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	return &peer, nil
}

func (r *Registry) ListOnlinePeers(ctx context.Context) ([]*Peer, error) {
	nodeIDs, err := r.client.SMembers(ctx, peersSetKey).Result()
	if err != nil {
		return nil, fmt.Errorf("redis smembers error: %w", err)
	}

	var peers []*Peer
	for _, nodeID := range nodeIDs {
		peer, err := r.GetPeer(ctx, nodeID)
		if err != nil {
			r.client.SRem(ctx, peersSetKey, nodeID)
			continue
		}
		peers = append(peers, peer)
	}

	return peers, nil
}

// CleanupStale removes Set members whose Redis key has already expired.
// Runs even when no one is listing peers, keeping the Set coherent.
// Should be called periodically with interval ~= HeartbeatTTL.
func (r *Registry) CleanupStale(ctx context.Context) (int, error) {
	nodeIDs, err := r.client.SMembers(ctx, peersSetKey).Result()
	if err != nil {
		return 0, fmt.Errorf("redis smembers error: %w", err)
	}
	if len(nodeIDs) == 0 {
		return 0, nil
	}

	pipe := r.client.Pipeline()
	cmds := make([]*redis.IntCmd, len(nodeIDs))
	for i, id := range nodeIDs {
		cmds[i] = pipe.Exists(ctx, peerKeyPrefix+id)
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return 0, fmt.Errorf("pipeline exec error: %w", err)
	}

	var removed int
	for i, cmd := range cmds {
		if cmd.Val() == 0 {
			r.client.SRem(ctx, peersSetKey, nodeIDs[i])
			removed++
		}
	}

	return removed, nil
}
