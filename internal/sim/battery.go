package sim

// ApplyBattery clamps a requested battery command to the given per-home
// physical limits (symmetric charge/discharge rate + capacity bounds) and
// returns the actual power applied plus the new state of charge.
//
// A capacity of 0 means the home has no battery; the command is ignored.
func ApplyBattery(cmd BatteryCmd, soc, capacityKWh, maxKW float64) (actualKW, newSOC float64) {
	if capacityKWh <= 0 || maxKW <= 0 {
		return 0, 0
	}
	p := cmd.PowerKW
	if p > maxKW {
		p = maxKW
	}
	if p < -maxKW {
		p = -maxKW
	}
	deltaKWh := p / float64(TicksPerHour)
	if soc+deltaKWh > capacityKWh {
		p = (capacityKWh - soc) * float64(TicksPerHour)
		deltaKWh = p / float64(TicksPerHour)
	}
	if soc+deltaKWh < 0 {
		p = -soc * float64(TicksPerHour)
		deltaKWh = p / float64(TicksPerHour)
	}
	return p, soc + deltaKWh
}
