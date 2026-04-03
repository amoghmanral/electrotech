package main

// sim config

const (
	TotalTicks    = 144  // 10-min intervals over 24h
	TicksPerHour  = 6
	SimHours      = 24.0

	// battery stuff (loosely based on a tesla powerwall)
	BatteryCapacity         = 13.5 // kWh
	BatteryMaxChargeRate    = 5.0  // kW
	BatteryMaxDischargeRate = 5.0

	SolarPeakKW = 8.0 // max solar output kW

	// load profile
	LoadBaselineKW  = 1.5
	LoadMaxSpikeKW  = 6.0  // oven, AC, etc
	LoadSpikeChance = 0.08 // per tick

	// grid price range $/kWh
	GridPriceMin = 0.08
	GridPriceMax = 0.35
)

// TickData is what gets sent to the policy each tick
type TickData struct {
	Tick         int
	Hour         float64
	SolarKW      float64
	LoadKW       float64
	BatteryLevel float64 // kWh
	GridPrice    float64
}

// BatteryCmd - positive = charge, negative = discharge
type BatteryCmd struct {
	PowerKW float64
}

// GridCmd - positive = import from grid, negative = export
type GridCmd struct {
	PowerKW float64
}

// TickResult holds the resolved energy balance for one tick (for logging)
type TickResult struct {
	Tick            int
	Hour            float64
	SolarKW         float64
	LoadKW          float64
	BatteryChargeKW float64
	BatteryLevel    float64
	GridKW          float64
	GridPrice       float64
	Cost            float64
	Waste           float64
}
