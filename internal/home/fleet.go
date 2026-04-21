package home

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/amoghmanral/electrotech/internal/sim"
)

// Fleet is the simulation clock for a group of homes. Each wall-clock
// tick, all homes run one simulated tick in parallel.
type Fleet struct {
	Homes []*Home

	mu             sync.RWMutex
	tick           int
	day            int
	speed          float64 // 1.0 = 500ms per tick
	running        bool
	history        []TickSummary
	recentRPCs     []RPCEvent
	rpcCount       atomic.Uint64
	rpcErrors      atomic.Uint64
	tickRate       float64       // ticks/sec in recent window
	tickLastMeas   time.Time
	tickLastCount  uint64
}

// TickSummary is a per-tick rollup for the chart.
type TickSummary struct {
	Tick       int     `json:"tick"`
	Hour       float64 `json:"hour"`
	Price      float64 `json:"price"`
	AggSolarKW float64 `json:"agg_solar_kw"`
	AggLoadKW  float64 `json:"agg_load_kw"`
	AggGridKW  float64 `json:"agg_grid_kw"`
	AvgSOC     float64 `json:"avg_soc_pct"`
}

const maxHistory = 120

func NewFleet(homes []*Home) *Fleet {
	return &Fleet{
		Homes:        homes,
		speed:        1.0,
		running:      true,
		history:      make([]TickSummary, 0, maxHistory),
		recentRPCs:   make([]RPCEvent, 0, 120),
		tickLastMeas: time.Now(),
	}
}

// WireRPCSink installs each home's RPC hook so events flow into the
// fleet's rolling log (for the dashboard).
func (f *Fleet) WireRPCSink() {
	for _, h := range f.Homes {
		h.onRPC = f.recordRPC
	}
}

func (f *Fleet) recordRPC(ev RPCEvent) {
	if ev.OK {
		f.rpcCount.Add(1)
	} else {
		f.rpcErrors.Add(1)
	}
	f.mu.Lock()
	f.recentRPCs = append(f.recentRPCs, ev)
	if len(f.recentRPCs) > 120 {
		f.recentRPCs = f.recentRPCs[len(f.recentRPCs)-120:]
	}
	f.mu.Unlock()
}

// Run drives the simulation clock. Each iteration advances one tick.
func (f *Fleet) Run(ctx context.Context) {
	basePeriod := 500 * time.Millisecond
	for {
		f.mu.RLock()
		speed := f.speed
		running := f.running
		f.mu.RUnlock()
		if speed <= 0 {
			speed = 1
		}
		wait := time.Duration(float64(basePeriod) / speed)
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		if !running {
			continue
		}
		f.advance(ctx)
	}
}

func (f *Fleet) advance(ctx context.Context) {
	f.mu.Lock()
	f.tick++
	if f.tick >= sim.TotalTicks {
		f.tick = 0
		f.day++
	}
	tick := f.tick
	f.mu.Unlock()

	// Fan-out RPCs: every home runs its tick concurrently.
	var wg sync.WaitGroup
	wg.Add(len(f.Homes))
	for _, h := range f.Homes {
		h := h
		go func() {
			defer wg.Done()
			h.Tick(ctx, tick)
		}()
	}
	wg.Wait()

	// Roll up into history. Average SOC is weighted by battery capacity so
	// a home with no battery doesn't drag the fleet average.
	var aggSolar, aggLoad, aggGrid, sumSOC, socDenom float64
	for _, h := range f.Homes {
		s := h.Snapshot()
		aggSolar += s.SolarKW
		aggLoad += s.LoadKW
		aggGrid += s.GridKW
		if s.BatteryCapacityKWh > 0 {
			sumSOC += s.BatterySOC / s.BatteryCapacityKWh
			socDenom++
		}
	}
	var avgSOC float64
	if socDenom > 0 {
		avgSOC = sumSOC / socDenom * 100
	}
	hour := sim.TickToHour(tick)
	price := sim.GridPrice(hour)

	f.mu.Lock()
	f.history = append(f.history, TickSummary{
		Tick: tick, Hour: hour, Price: price,
		AggSolarKW: aggSolar, AggLoadKW: aggLoad, AggGridKW: aggGrid, AvgSOC: avgSOC,
	})
	if len(f.history) > maxHistory {
		f.history = f.history[len(f.history)-maxHistory:]
	}
	// tick rate smoothing
	now := time.Now()
	if dt := now.Sub(f.tickLastMeas).Seconds(); dt >= 1.0 {
		newCount := f.rpcCount.Load()
		f.tickRate = float64(newCount-f.tickLastCount) / dt
		f.tickLastMeas = now
		f.tickLastCount = newCount
	}
	f.mu.Unlock()
}

// ----- controls + snapshot for the dashboard -----

func (f *Fleet) SetSpeed(s float64) {
	f.mu.Lock()
	f.speed = s
	f.mu.Unlock()
}

func (f *Fleet) SetRunning(r bool) {
	f.mu.Lock()
	f.running = r
	f.mu.Unlock()
}

// Snapshot returns the combined dashboard view.
type FleetSnapshot struct {
	Tick       int            `json:"tick"`
	Day        int            `json:"day"`
	Hour       float64        `json:"hour"`
	Price      float64        `json:"price"`
	Speed      float64        `json:"speed"`
	Running    bool           `json:"running"`
	Homes      []Snapshot     `json:"homes"`
	History    []TickSummary  `json:"history"`
	RecentRPCs []RPCEvent     `json:"recent_rpcs"`
	RPCsOK     uint64         `json:"rpcs_ok"`
	RPCsErr    uint64         `json:"rpcs_err"`
	RPCRate    float64        `json:"rpc_rate"`
	FleetAgg   TickSummary    `json:"fleet_now"`
}

func (f *Fleet) Snapshot() FleetSnapshot {
	f.mu.RLock()
	tick := f.tick
	day := f.day
	speed := f.speed
	running := f.running
	history := append([]TickSummary(nil), f.history...)
	rpcs := append([]RPCEvent(nil), f.recentRPCs...)
	rate := f.tickRate
	f.mu.RUnlock()

	homes := make([]Snapshot, len(f.Homes))
	var aggSolar, aggLoad, aggGrid, sumSOC, socDenom float64
	for i, h := range f.Homes {
		homes[i] = h.Snapshot()
		aggSolar += homes[i].SolarKW
		aggLoad += homes[i].LoadKW
		aggGrid += homes[i].GridKW
		if homes[i].BatteryCapacityKWh > 0 {
			sumSOC += homes[i].BatterySOC / homes[i].BatteryCapacityKWh
			socDenom++
		}
	}
	var avgSOC float64
	if socDenom > 0 {
		avgSOC = sumSOC / socDenom * 100
	}
	hour := sim.TickToHour(tick)
	price := sim.GridPrice(hour)
	return FleetSnapshot{
		Tick: tick, Day: day, Hour: hour, Price: price,
		Speed: speed, Running: running,
		Homes: homes, History: history, RecentRPCs: rpcs,
		RPCsOK: f.rpcCount.Load(), RPCsErr: f.rpcErrors.Load(),
		RPCRate: rate,
		FleetAgg: TickSummary{
			Tick: tick, Hour: hour, Price: price,
			AggSolarKW: aggSolar, AggLoadKW: aggLoad, AggGridKW: aggGrid,
			AvgSOC: avgSOC,
		},
	}
}
