package main

// PolicyGreedy - simplest strategy: use solar first, then battery, then grid.
// no lookahead, no price awareness. just tries to minimize grid usage greedily.
func PolicyGreedy(dataCh <-chan TickData, batCmdCh chan<- BatteryCmd, gridCmdCh chan<- GridCmd) {
	for td := range dataCh {
		batCmd, gridCmd := greedyDispatch(td)
		batCmdCh <- batCmd
		gridCmdCh <- gridCmd
	}
}

func greedyDispatch(td TickData) (BatteryCmd, GridCmd) {
	surplus := td.SolarKW - td.LoadKW

	var batPower, gridPower float64

	if surplus >= 0 {
		// extra solar -> charge battery, export whatever doesn't fit
		batPower = surplus
		if batPower > BatteryMaxChargeRate {
			batPower = BatteryMaxChargeRate
		}
		gridPower = -(surplus - batPower)
	} else {
		// not enough solar -> pull from battery, then grid
		deficit := -surplus
		batPower = -deficit
		if batPower < -BatteryMaxDischargeRate {
			batPower = -BatteryMaxDischargeRate
		}
		// make sure we actually have juice
		availableKWh := td.BatteryLevel
		maxDischargeTick := availableKWh * float64(TicksPerHour)
		if -batPower > maxDischargeTick {
			batPower = -maxDischargeTick
		}
		gridPower = deficit + batPower // batPower is negative so this shrinks the import
		if gridPower < 0 {
			gridPower = 0
		}
	}

	return BatteryCmd{PowerKW: batPower}, GridCmd{PowerKW: gridPower}
}

const priceThreshold = (GridPriceMin + GridPriceMax) / 2.0

// PolicyPriceReactive - charges battery from solar and grid when price is cheap, falls back to greedy when expensive.
func PolicyPriceReactive(dataCh chan TickData, batCmdCh chan BatteryCmd, gridCmdCh chan GridCmd) {
	for td := range dataCh {
		batCmd, gridCmd := priceReactiveDispatch(td)
		batCmdCh <- batCmd
		gridCmdCh <- gridCmd
	}
}

func priceReactiveDispatch(td TickData) (BatteryCmd, GridCmd) {
	if td.GridPrice >= priceThreshold {
		// expensive grid price, so use greedy strategy
		return greedyDispatch(td)
	}

	// cheap grid price, charge battery as much as possible
	space := BatteryCapacity - td.BatteryLevel
	batteryPower := space * float64(TicksPerHour)
	if batteryPower > BatteryMaxChargeRate {
		batteryPower = BatteryMaxChargeRate
	}
	gridPower := td.LoadKW - td.SolarKW + batteryPower
	return BatteryCmd{PowerKW: batteryPower}, GridCmd{PowerKW: gridPower}
}

const (
	predictWindow    = 12   // ticks to look ahead (2 hours)
	chargeTrigger    = 0.85 // charge if current price < 85% of future average
	dischargeTrigger = 1.15 // discharge if current price > 115% of future average
)

// PolicyPredictive - looks ahead using the solar curve and price schedule.
// takes from grid when price is cheap now
// discharges when price is higher now, otherwise falls back to greedy.
func PolicyPredictive(dataCh chan TickData, batCmdCh chan BatteryCmd, gridCmdCh chan GridCmd) {
	for td := range dataCh {
		batCmd, gridCmd := predictiveDispatch(td)
		batCmdCh <- batCmd
		gridCmdCh <- gridCmd
	}
}

func predictiveDispatch(td TickData) (BatteryCmd, GridCmd) {
	windowSize := predictWindow
	if td.Tick + windowSize >= TotalTicks {
		windowSize = TotalTicks - td.Tick - 1
	}

	futureAvgPrice := 0.0
	futureSolarKWh := 0.0
	for i := 1; i <= windowSize; i++ {
		h := tickToHour(td.Tick + i)
		futureAvgPrice += gridPrice(h)
		futureSolarKWh += solarBase(h) / float64(TicksPerHour)
	}
	if windowSize > 0 {
		futureAvgPrice /= float64(windowSize)
	} else {
		futureAvgPrice = td.GridPrice
	}

	// skip grid charging if incoming solar will fill battery
	batteryHeadroom := BatteryCapacity - td.BatteryLevel
	solarWillFillBattery := futureSolarKWh >= batteryHeadroom

	switch {
		case td.GridPrice < futureAvgPrice * chargeTrigger && !solarWillFillBattery:
			// grid cheap now and solar won't fill battery: charge from grid
			space := BatteryCapacity - td.BatteryLevel
			batteryPower := space * float64(TicksPerHour)
			if batteryPower > BatteryMaxChargeRate {
				batteryPower = BatteryMaxChargeRate
			}
			gridPower := td.LoadKW - td.SolarKW + batteryPower
			return BatteryCmd{PowerKW: batteryPower}, GridCmd{PowerKW: gridPower}

		case td.GridPrice > futureAvgPrice * dischargeTrigger:
			// grid expensive now: sell to grid if possible
			batteryPower := td.SolarKW - td.LoadKW
			if batteryPower >= 0 {
				if batteryPower > BatteryMaxChargeRate {
					batteryPower = BatteryMaxChargeRate
				}
				return BatteryCmd{PowerKW: batteryPower}, GridCmd{PowerKW: -((td.SolarKW - td.LoadKW) - batteryPower)}
			}
			if batteryPower < -BatteryMaxDischargeRate {
				batteryPower = -BatteryMaxDischargeRate
			}
			maxDischarge := td.BatteryLevel * float64(TicksPerHour)
			if batteryPower < -maxDischarge {
				batteryPower = -maxDischarge
			}
			gridPower := td.LoadKW - td.SolarKW + batteryPower
			if gridPower < 0 {
				gridPower = 0
			}
			return BatteryCmd{PowerKW: batteryPower}, GridCmd{PowerKW: gridPower}

		default:
			return greedyDispatch(td)
	}
}
