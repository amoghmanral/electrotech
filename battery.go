package main

// Battery goroutine - tracks state of charge, applies commands each tick.
// sends back the updated level and actual power that was applied
// (might differ from requested if we hit limits)
func Battery(cmdCh <-chan BatteryCmd, levelCh chan<- float64, actualCh chan<- float64) {
	soc := BatteryCapacity * 0.5 // start half charged

	for cmd := range cmdCh {
		power := cmd.PowerKW
		actual := clampBattery(power, soc)
		soc += actual / float64(TicksPerHour) // kW -> kWh for this tick
		if soc < 0 {
			soc = 0
		}
		if soc > BatteryCapacity {
			soc = BatteryCapacity
		}
		levelCh <- soc
		actualCh <- actual
	}
}

// clamp to rate limits and capacity bounds
func clampBattery(powerKW, currentSOC float64) float64 {
	if powerKW > BatteryMaxChargeRate {
		powerKW = BatteryMaxChargeRate
	}
	if powerKW < -BatteryMaxDischargeRate {
		powerKW = -BatteryMaxDischargeRate
	}
	// don't overcharge
	energyPerTick := powerKW / float64(TicksPerHour)
	if currentSOC+energyPerTick > BatteryCapacity {
		powerKW = (BatteryCapacity - currentSOC) * float64(TicksPerHour)
	}
	// don't go below zero
	if currentSOC+energyPerTick < 0 {
		powerKW = -currentSOC * float64(TicksPerHour)
	}
	return powerKW
}
