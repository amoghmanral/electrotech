package sim

// TickToHour maps a tick number to simulated wall-clock hour [0, 24).
func TickToHour(tick int) float64 {
	return SimHours * float64(tick) / float64(TotalTicks)
}
