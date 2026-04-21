// Package policyserver is the gRPC policy service — the cloud "brain"
// that home clients call once per tick for their dispatch decision.
//
// This package is consumed by both the standalone policy-server binary
// and the in-process demo launcher. The service is stateless: each RPC
// is a pure function of the request.
package policyserver

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"

	"github.com/amoghmanral/electrotech/internal/policypb"
	"github.com/amoghmanral/electrotech/internal/sim"
)

// Service implements policypb.PolicyServiceServer.
type Service struct {
	policypb.UnimplementedPolicyServiceServer
	strategy string
	stats    *Stats
}

// NewService constructs a service with the given strategy. Valid values:
// "greedy", "reactive", "predictive".
func NewService(strategy string) (*Service, error) {
	switch strategy {
	case "greedy", "reactive", "predictive":
	default:
		return nil, fmt.Errorf("unknown strategy %q", strategy)
	}
	return &Service{strategy: strategy, stats: newStats(strategy)}, nil
}

func (s *Service) Strategy() string { return s.strategy }
func (s *Service) Stats() *Stats    { return s.stats }

// Dispatch is the gRPC handler. Single pure function call per request —
// no state carried across calls.
func (s *Service) Dispatch(ctx context.Context, req *policypb.TickContext) (*policypb.DispatchDecision, error) {
	td := sim.TickData{
		Tick:               int(req.Tick),
		Hour:               req.Hour,
		SolarKW:            req.SolarKw,
		LoadKW:             req.LoadKw,
		BatteryLevel:       req.BatterySocKwh,
		GridPrice:          req.GridPrice,
		BatteryCapacityKWh: req.BatteryCapacityKwh,
		BatteryMaxKW:       req.BatteryMaxKw,
		SolarPeakKW:        req.SolarPeakKw,
	}
	bat, grid, err := sim.Dispatch(s.strategy, td)
	if err != nil {
		return nil, err
	}
	return &policypb.DispatchDecision{
		BatteryKw: bat.PowerKW,
		GridKw:    grid.PowerKW,
		Strategy:  s.strategy,
	}, nil
}

// NewGRPCServer builds a gRPC server with the logging interceptor
// attached and the service registered on it.
func NewGRPCServer(s *Service) *grpc.Server {
	g := grpc.NewServer(grpc.UnaryInterceptor(s.stats.loggingInterceptor))
	policypb.RegisterPolicyServiceServer(g, s)
	return g
}

// ----- observability ---------------------------------------------------

// Stats tracks server-side RPC metrics exposed to the dashboard.
type Stats struct {
	totalRPCs   atomic.Uint64
	totalNanos  atomic.Uint64
	activeCalls atomic.Int64
	startTime   time.Time
	strategy    string

	mu     sync.Mutex
	recent []RecentRPC
}

// RecentRPC is one entry in the server's rolling call log.
type RecentRPC struct {
	At        time.Time `json:"at"`
	HomeID    string    `json:"home_id"`
	Tick      int32     `json:"tick"`
	LatencyMs float64   `json:"latency_ms"`
	Decision  string    `json:"decision"`
}

func newStats(strategy string) *Stats {
	return &Stats{strategy: strategy, startTime: time.Now(), recent: make([]RecentRPC, 0, 64)}
}

func (s *Stats) loggingInterceptor(
	ctx context.Context, req any,
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
) (any, error) {
	s.activeCalls.Add(1)
	defer s.activeCalls.Add(-1)
	start := time.Now()
	resp, err := handler(ctx, req)
	latency := time.Since(start)

	var homeID string
	var tick int32
	var decision string
	if tc, ok := req.(*policypb.TickContext); ok {
		homeID, tick = tc.HomeId, tc.Tick
	}
	if dd, ok := resp.(*policypb.DispatchDecision); ok {
		decision = fmt.Sprintf("bat=%+.2f grid=%+.2f", dd.BatteryKw, dd.GridKw)
	}
	s.totalRPCs.Add(1)
	s.totalNanos.Add(uint64(latency.Nanoseconds()))
	s.mu.Lock()
	s.recent = append(s.recent, RecentRPC{
		At: time.Now(), HomeID: homeID, Tick: tick,
		LatencyMs: float64(latency.Microseconds()) / 1000.0,
		Decision:  decision,
	})
	if len(s.recent) > 64 {
		s.recent = s.recent[len(s.recent)-64:]
	}
	s.mu.Unlock()
	return resp, err
}

// Snapshot is the JSON-safe dashboard payload.
type Snapshot struct {
	Strategy     string      `json:"strategy"`
	UptimeSec    float64     `json:"uptime_sec"`
	TotalRPCs    uint64      `json:"total_rpcs"`
	RatePerSec   float64     `json:"rate_per_sec"`
	AvgLatencyMs float64     `json:"avg_latency_ms"`
	ActiveCalls  int64       `json:"active_calls"`
	Recent       []RecentRPC `json:"recent"`
}

func (s *Stats) Snapshot() Snapshot {
	total := s.totalRPCs.Load()
	nanos := s.totalNanos.Load()
	uptime := time.Since(s.startTime).Seconds()
	var avg, rate float64
	if total > 0 {
		avg = float64(nanos) / float64(total) / 1e6
	}
	if uptime > 0 {
		rate = float64(total) / uptime
	}
	s.mu.Lock()
	recent := append([]RecentRPC(nil), s.recent...)
	s.mu.Unlock()
	return Snapshot{
		Strategy: s.strategy, UptimeSec: uptime,
		TotalRPCs: total, RatePerSec: rate, AvgLatencyMs: avg,
		ActiveCalls: s.activeCalls.Load(), Recent: recent,
	}
}
