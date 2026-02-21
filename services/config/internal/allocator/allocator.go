package allocator

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/redis/go-redis/v9"
)

const (
	allocatedSetKey = "ip:allocated"
	nodeIPKeyPrefix = "ip:node:"
)

type Allocator struct {
	client  *redis.Client
	network *net.IPNet
	gateway net.IP
	mu      sync.Mutex
}

func New(redisURL string, networkCIDR string) (*Allocator, error) {
	client := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	_, network, err := net.ParseCIDR(networkCIDR)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %w", err)
	}

	// Gateway = first usable IP (10.8.0.1)
	gateway := make(net.IP, len(network.IP))
	copy(gateway, network.IP)
	gateway[len(gateway)-1] = 1

	return &Allocator{
		client:  client,
		network: network,
		gateway: gateway,
	}, nil
}

func (a *Allocator) Close() error {
	return a.client.Close()
}

func (a *Allocator) Gateway() string {
	return a.gateway.String()
}

func (a *Allocator) SubnetMask() string {
	mask := a.network.Mask
	return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
}

func (a *Allocator) AllocateIP(ctx context.Context, nodeID string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	existing, err := a.client.Get(ctx, nodeIPKeyPrefix+nodeID).Result()
	if err == nil {
		return existing, nil
	}

	ip, err := a.findAvailableIP(ctx)
	if err != nil {
		return "", err
	}

	ipStr := ip.String()

	if err := a.client.SAdd(ctx, allocatedSetKey, ipStr).Err(); err != nil {
		return "", fmt.Errorf("redis sadd error: %w", err)
	}

	if err := a.client.Set(ctx, nodeIPKeyPrefix+nodeID, ipStr, 0).Err(); err != nil {
		return "", fmt.Errorf("redis set error: %w", err)
	}

	return ipStr, nil
}

func (a *Allocator) ReleaseIP(ctx context.Context, nodeID string, ipAddress string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.client.SRem(ctx, allocatedSetKey, ipAddress).Err(); err != nil {
		return fmt.Errorf("redis srem error: %w", err)
	}

	if err := a.client.Del(ctx, nodeIPKeyPrefix+nodeID).Err(); err != nil {
		return fmt.Errorf("redis del error: %w", err)
	}

	return nil
}

func (a *Allocator) findAvailableIP(ctx context.Context) (net.IP, error) {
	allocated, err := a.client.SMembers(ctx, allocatedSetKey).Result()
	if err != nil {
		return nil, fmt.Errorf("redis smembers error: %w", err)
	}

	usedSet := make(map[string]bool)
	for _, ip := range allocated {
		usedSet[ip] = true
	}

	ip := make(net.IP, len(a.network.IP))
	copy(ip, a.network.IP)

	for ip[len(ip)-1] = 2; a.network.Contains(ip); incrementIP(ip) {
		if !usedSet[ip.String()] {
			result := make(net.IP, len(ip))
			copy(result, ip)
			return result, nil
		}
	}

	return nil, fmt.Errorf("no available IPs in network")
}

func incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}
