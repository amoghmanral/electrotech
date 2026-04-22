package sim

// Global simulation constants. Per-home physical limits (battery
// capacity, solar peak, etc.) live on each home's Profile — see
// profile.go.
const (
	TicksPerHour = 6    // 10-minute decision interval
	SimHours     = 24.0 // one day per simulated loop
	TotalTicks   = TicksPerHour * int(SimHours)

	// Grid tariff bounds (the utility's price curve — see grid.go).
	GridPriceMin = 0.08
	GridPriceMax = 0.39
)

// TickData is the per-tick snapshot the policy consumes.
//
// It carries both the live measurements (solar/load/battery level/price)
// and the home's physical limits (battery capacity, max charge rate,
// solar peak). The policy is a pure function of its input with no access
// to shared globals, so a single server can serve heterogeneous homes.
type TickData struct {
	Tick         int
	Hour         float64
	SolarKW      float64
	LoadKW       float64
	BatteryLevel float64
	GridPrice    float64

	// Per-home physical parameters.
	BatteryCapacityKWh float64
	BatteryMaxKW       float64
	SolarPeakKW        float64
}

// BatteryCmd: positive charges the battery, negative discharges.
type BatteryCmd struct{ PowerKW float64 }

// GridCmd: positive imports from the grid, negative exports to the grid.
type GridCmd struct{ PowerKW float64 }
