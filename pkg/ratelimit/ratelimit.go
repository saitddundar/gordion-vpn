package ratelimit

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type Limiter struct {
	mu       sync.Mutex
	clients  map[string]*window
	maxReqs  int
	interval time.Duration
}

type window struct {
	count   int
	resetAt time.Time
}

func New(maxReqs int, interval time.Duration) *Limiter {
	l := &Limiter{
		clients:  make(map[string]*window),
		maxReqs:  maxReqs,
		interval: interval,
	}

	// Cleanup stale entries every interval
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			l.cleanup()
		}
	}()

	return l
}

func (l *Limiter) Allow(clientIP string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	w, exists := l.clients[clientIP]

	if !exists || now.After(w.resetAt) {
		l.clients[clientIP] = &window{
			count:   1,
			resetAt: now.Add(l.interval),
		}
		return true
	}

	if w.count >= l.maxReqs {
		return false
	}

	w.count++
	return true
}

func (l *Limiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for ip, w := range l.clients {
		if now.After(w.resetAt) {
			delete(l.clients, ip)
		}
	}
}

func UnaryInterceptor(limiter *Limiter) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		clientIP := "unknown"
		if p, ok := peer.FromContext(ctx); ok {
			clientIP = p.Addr.String()
		}

		if !limiter.Allow(clientIP) {
			return nil, status.Errorf(codes.ResourceExhausted,
				"rate limit exceeded: max %d requests per %s",
				limiter.maxReqs, limiter.interval)
		}

		return handler(ctx, req)
	}
}
