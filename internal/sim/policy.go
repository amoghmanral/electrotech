package sim

import "fmt"

// Dispatch runs the named strategy for a single tick. This is the one
// entry point the policy server uses — strategies are pure functions of
// TickData with no state carried between calls, so one server instance
// can serve heterogeneous homes concurrently.
func Dispatch(strategy string, td TickData) (BatteryCmd, GridCmd, error) {
	switch strategy {
	case "greedy":
		b, g := GreedyDispatch(td)
		return b, g, nil
	case "reactive":
		b, g := PriceReactiveDispatch(td)
		return b, g, nil
	case "predictive":
		b, g := PredictiveDispatch(td)
		return b, g, nil
	default:
		return BatteryCmd{}, GridCmd{}, fmt.Errorf("unknown strategy %q", strategy)
	}
}

// GreedyDispatch: use solar first, battery next, grid last. No price or
// forecast awareness.
func GreedyDispatch(td TickData) (BatteryCmd, GridCmd) {
	surplus := td.SolarKW - td.LoadKW
	var bat, grid float64
	if surplus >= 0 {
		bat = surplus
		if bat > td.BatteryMaxKW {
			bat = td.BatteryMaxKW
		}
		grid = -(surplus - bat) // export the leftover
	} else {
		deficit := -surplus
		bat = -deficit
		if bat < -td.BatteryMaxKW {
			bat = -td.BatteryMaxKW
		}
		maxDischarge := td.BatteryLevel * float64(TicksPerHour)
		if -bat > maxDischarge {
			bat = -maxDischarge
		}
		grid = deficit + bat
		if grid < 0 {
			grid = 0
		}
	}
	return BatteryCmd{PowerKW: bat}, GridCmd{PowerKW: grid}
}

const priceMid = (GridPriceMin + GridPriceMax) / 2.0

// PriceReactiveDispatch: charge aggressively from the grid when it's
// cheap, greedy otherwise. Homes with no battery fall back to greedy.
func PriceReactiveDispatch(td TickData) (BatteryCmd, GridCmd) {
	if td.GridPrice >= priceMid || td.BatteryCapacityKWh <= 0 {
		return GreedyDispatch(td)
	}
	space := td.BatteryCapacityKWh - td.BatteryLevel
	bat := space * float64(TicksPerHour)
	if bat > td.BatteryMaxKW {
		bat = td.BatteryMaxKW
	}
	grid := td.LoadKW - td.SolarKW + bat
	return BatteryCmd{PowerKW: bat}, GridCmd{PowerKW: grid}
}

const (
	predictWindow    = 12
	chargeTrigger    = 0.85
	dischargeTrigger = 1.15
)

// PredictiveDispatch looks ahead 2 hours at both the price curve and
// solar forecast. If the current price is significantly below the window
// average and solar won't fill the battery on its own, charge from the
// grid; if it's well above average, discharge. Otherwise greedy.
func PredictiveDispatch(td TickData) (BatteryCmd, GridCmd) {
	if td.BatteryCapacityKWh <= 0 {
		return GreedyDispatch(td)
	}
	window := predictWindow
	if td.Tick+window >= TotalTicks {
		window = TotalTicks - td.Tick - 1
	}
	var futurePrice, futureSolarKWh float64
	for i := 1; i <= window; i++ {
		h := TickToHour(td.Tick + i)
		futurePrice += GridPrice(h)
		futureSolarKWh += SolarBase(h, td.SolarPeakKW) / float64(TicksPerHour)
	}
	if window > 0 {
		futurePrice /= float64(window)
	} else {
		futurePrice = td.GridPrice
	}
	headroom := td.BatteryCapacityKWh - td.BatteryLevel
	solarFills := futureSolarKWh >= headroom

	switch {
	case td.GridPrice < futurePrice*chargeTrigger && !solarFills:
		space := td.BatteryCapacityKWh - td.BatteryLevel
		bat := space * float64(TicksPerHour)
		if bat > td.BatteryMaxKW {
			bat = td.BatteryMaxKW
		}
		grid := td.LoadKW - td.SolarKW + bat
		return BatteryCmd{PowerKW: bat}, GridCmd{PowerKW: grid}

	case td.GridPrice > futurePrice*dischargeTrigger:
		bat := td.SolarKW - td.LoadKW
		if bat >= 0 {
			if bat > td.BatteryMaxKW {
				bat = td.BatteryMaxKW
			}
			return BatteryCmd{PowerKW: bat}, GridCmd{PowerKW: -((td.SolarKW - td.LoadKW) - bat)}
		}
		if bat < -td.BatteryMaxKW {
			bat = -td.BatteryMaxKW
		}
		maxDis := td.BatteryLevel * float64(TicksPerHour)
		if bat < -maxDis {
			bat = -maxDis
		}
		grid := td.LoadKW - td.SolarKW + bat
		if grid < 0 {
			grid = 0
		}
		return BatteryCmd{PowerKW: bat}, GridCmd{PowerKW: grid}

	default:
		return GreedyDispatch(td)
	}
}
