package circuitbreaker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

type Config struct {
	MaxFailures   int
	OpenTimeout   time.Duration
	HalfOpenMax   int
	OnStateChange func(name string, from, to State)
}

func DefaultConfig() Config {
	return Config{
		MaxFailures: 5,
		OpenTimeout: 30 * time.Second,
		HalfOpenMax: 1,
	}
}

type CircuitBreaker struct {
	mu   sync.Mutex
	name string
	cfg  Config

	state       State
	failures    int
	successes   int
	lastFailure time.Time
	halfOpenReq int
}

var ErrOpen = fmt.Errorf("circuit breaker open")

func New(name string, cfg Config) *CircuitBreaker {
	if cfg.MaxFailures <= 0 {
		cfg.MaxFailures = DefaultConfig().MaxFailures
	}
	if cfg.OpenTimeout <= 0 {
		cfg.OpenTimeout = DefaultConfig().OpenTimeout
	}
	if cfg.HalfOpenMax <= 0 {
		cfg.HalfOpenMax = DefaultConfig().HalfOpenMax
	}
	return &CircuitBreaker{
		name:  name,
		cfg:   cfg,
		state: StateClosed,
	}
}

func (cb *CircuitBreaker) Name() string { return cb.name }

func (cb *CircuitBreaker) CurrentState() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		if time.Since(cb.lastFailure) >= cb.cfg.OpenTimeout {
			cb.transition(StateHalfOpen)
			cb.halfOpenReq = 1
			return true
		}
		return false

	case StateHalfOpen:
		if cb.halfOpenReq < cb.cfg.HalfOpenMax {
			cb.halfOpenReq++
			return true
		}
		return false
	}

	return false
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0

	if cb.state == StateHalfOpen {
		cb.successes++
		if cb.successes >= cb.cfg.HalfOpenMax {
			cb.successes = 0
			cb.halfOpenReq = 0
			cb.transition(StateClosed)
		}
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailure = time.Now()

	switch cb.state {
	case StateClosed:
		cb.failures++
		if cb.failures >= cb.cfg.MaxFailures {
			cb.successes = 0
			cb.halfOpenReq = 0
			cb.transition(StateOpen)
		}

	case StateHalfOpen:
		cb.failures++
		cb.successes = 0
		cb.halfOpenReq = 0
		cb.transition(StateOpen)
	}
}

func (cb *CircuitBreaker) transition(to State) {
	from := cb.state
	if from == to {
		return
	}
	cb.state = to
	if cb.cfg.OnStateChange != nil {
		cb.mu.Unlock()
		cb.cfg.OnStateChange(cb.name, from, to)
		cb.mu.Lock()
	}
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.Allow() {
		return status.Errorf(codes.Unavailable,
			"circuit breaker [%s] is OPEN (retry after %s)",
			cb.name, cb.cfg.OpenTimeout)
	}

	err := fn()
	if err != nil {
		if s, ok := status.FromError(err); ok {
			switch s.Code() {
			case codes.Canceled, codes.DeadlineExceeded:
			default:
				cb.RecordFailure()
			}
		} else {
			cb.RecordFailure()
		}
		return err
	}

	cb.RecordSuccess()
	return nil
}

func (cb *CircuitBreaker) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		return cb.Execute(func() error {
			return invoker(ctx, method, req, reply, cc, opts...)
		})
	}
}
