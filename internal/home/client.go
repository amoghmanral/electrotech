// Package home implements the home-client side of the ElectroTech
// simulator. Each home owns its local physics (solar production, load,
// battery) and calls the policy-server via gRPC once per tick for its
// strategic dispatch decision.
package home

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/amoghmanral/electrotech/internal/policypb"
	"github.com/amoghmanral/electrotech/internal/sim"
)

// RPCEvent is a compact record of one RPC, used for the dashboard log.
type RPCEvent struct {
	At        time.Time `json:"at"`
	HomeID    string    `json:"home_id"`
	Tick      int       `json:"tick"`
	LatencyMs float64   `json:"latency_ms"`
	OK        bool      `json:"ok"`
	Err       string    `json:"err,omitempty"`
	Strategy  string    `json:"strategy,omitempty"`
	BatteryKW float64   `json:"battery_kw"`
	GridKW    float64   `json:"grid_kw"`
}

// Home is one simulated household. It holds local physics state, a
// per-home Profile (archetype + jitter + sprite spec), and a gRPC client
// to the policy server.
type Home struct {
	ID      string
	Profile sim.Profile
	client  policypb.PolicyServiceClient

	mu          sync.RWMutex
	tick        int
	hour        float64
	solarKW     float64
	loadKW      float64
	batterySOC  float64
	batteryKW   float64
	gridKW      float64
	price       float64
	strategy    string
	lastLatMs   float64
	online      bool
	lastErr     string
	totalCost   float64 // $ (positive = paid to utility)
	totalImport float64 // kWh from grid
	totalExport float64 // kWh to grid
	rng         *rand.Rand

	rpcCount  atomic.Uint64
	rpcErrors atomic.Uint64

	onRPC func(RPCEvent) // dashboard hook
}

// New creates a home with its archetype/profile derived deterministically
// from the ID. The per-home RNG is seeded with the caller-supplied seed
// so solar noise and load spikes are reproducible.
func New(id string, seed int64, conn *grpc.ClientConn, onRPC func(RPCEvent)) *Home {
	p := sim.ProfileFor(id)
	return &Home{
		ID:         id,
		Profile:    p,
		client:     policypb.NewPolicyServiceClient(conn),
		rng:        rand.New(rand.NewSource(seed)),
		batterySOC: p.BatteryCapacityKWh * 0.5, // start half charged
		online:     true,
		onRPC:      onRPC,
	}
}

// Tick runs the physics + RPC for one tick. Called by the fleet driver.
func (h *Home) Tick(ctx context.Context, tick int) {
	hour := sim.TickToHour(tick)
	price := sim.GridPrice(hour)

	h.mu.Lock()
	solar := sim.SolarOutputWithRand(hour, h.Profile.SolarPeakKW, h.rng)
	load := sim.LoadDemandWithRand(h.rng, h.Profile.LoadBaselineKW, h.Profile.LoadSpikeKW, h.Profile.LoadSpikeChance)
	soc := h.batterySOC
	h.tick = tick
	h.hour = hour
	h.solarKW = solar
	h.loadKW = load
	h.price = price
	h.mu.Unlock()

	// ----- RPC to policy server -----
	req := &policypb.TickContext{
		HomeId:             h.ID,
		Tick:               int32(tick),
		Hour:               hour,
		SolarKw:            solar,
		LoadKw:             load,
		BatterySocKwh:      soc,
		GridPrice:          price,
		BatteryCapacityKwh: h.Profile.BatteryCapacityKWh,
		BatteryMaxKw:       h.Profile.BatteryMaxKW,
		SolarPeakKw:        h.Profile.SolarPeakKW,
	}
	rpcStart := time.Now()
	rpcCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	decision, err := h.client.Dispatch(rpcCtx, req)
	cancel()
	latency := time.Since(rpcStart)

	ev := RPCEvent{
		At:        time.Now(),
		HomeID:    h.ID,
		Tick:      tick,
		LatencyMs: float64(latency.Microseconds()) / 1000.0,
	}
	if err != nil {
		h.rpcErrors.Add(1)
		h.mu.Lock()
		h.lastLatMs = ev.LatencyMs
		h.online = false
		h.lastErr = shortErr(err)
		h.mu.Unlock()
		ev.OK = false
		ev.Err = shortErr(err)
		if h.onRPC != nil {
			h.onRPC(ev)
		}
		return
	}
	h.rpcCount.Add(1)

	// ----- apply decision to local physics -----
	batCmd := sim.BatteryCmd{PowerKW: decision.BatteryKw}
	actualBat, newSOC := sim.ApplyBattery(batCmd, soc, h.Profile.BatteryCapacityKWh, h.Profile.BatteryMaxKW)
	gridKW := decision.GridKw

	// simple cost accounting
	energyHour := 1.0 / float64(sim.TicksPerHour)
	if gridKW > 0 {
		h.totalImport += gridKW * energyHour
		h.totalCost += gridKW * energyHour * price
	} else if gridKW < 0 {
		h.totalExport += -gridKW * energyHour
		h.totalCost += gridKW * energyHour * price // negative = credit
	}

	h.mu.Lock()
	h.batterySOC = newSOC
	h.batteryKW = actualBat
	h.gridKW = gridKW
	h.strategy = decision.Strategy
	h.lastLatMs = ev.LatencyMs
	h.online = true
	h.lastErr = ""
	h.mu.Unlock()

	ev.OK = true
	ev.Strategy = decision.Strategy
	ev.BatteryKW = actualBat
	ev.GridKW = gridKW
	if h.onRPC != nil {
		h.onRPC(ev)
	}
}

func shortErr(err error) string {
	if s, ok := status.FromError(err); ok {
		return s.Code().String() + ": " + s.Message()
	}
	return err.Error()
}

// Snapshot is the dashboard-facing view of a home.
type Snapshot struct {
	ID                 string         `json:"id"`
	Tick               int            `json:"tick"`
	Hour               float64        `json:"hour"`
	SolarKW            float64        `json:"solar_kw"`
	LoadKW             float64        `json:"load_kw"`
	BatterySOC         float64        `json:"battery_soc_kwh"`
	BatteryKW          float64        `json:"battery_kw"`
	GridKW             float64        `json:"grid_kw"`
	Price              float64        `json:"price"`
	Strategy           string         `json:"strategy"`
	LatencyMs          float64        `json:"rpc_ms"`
	Online             bool           `json:"online"`
	LastErr            string         `json:"last_err,omitempty"`
	TotalCost          float64        `json:"total_cost"`
	TotalImport        float64        `json:"total_import_kwh"`
	TotalExport        float64        `json:"total_export_kwh"`
	RPCCount           uint64         `json:"rpc_count"`
	RPCErrors          uint64         `json:"rpc_errors"`
	Archetype          string         `json:"archetype"`
	BatteryCapacityKWh float64        `json:"battery_capacity_kwh"`
	SolarPeakKW        float64        `json:"solar_peak_kw"`
	Sprite             sim.SpriteSpec `json:"sprite"`
}

func (h *Home) Snapshot() Snapshot {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return Snapshot{
		ID: h.ID, Tick: h.tick, Hour: h.hour,
		SolarKW: h.solarKW, LoadKW: h.loadKW,
		BatterySOC: h.batterySOC, BatteryKW: h.batteryKW, GridKW: h.gridKW,
		Price: h.price, Strategy: h.strategy,
		LatencyMs: h.lastLatMs, Online: h.online, LastErr: h.lastErr,
		TotalCost: h.totalCost, TotalImport: h.totalImport, TotalExport: h.totalExport,
		RPCCount: h.rpcCount.Load(), RPCErrors: h.rpcErrors.Load(),
		Archetype:          h.Profile.ArchetypeName,
		BatteryCapacityKWh: h.Profile.BatteryCapacityKWh,
		SolarPeakKW:        h.Profile.SolarPeakKW,
		Sprite:             h.Profile.Sprite,
	}
}
