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
