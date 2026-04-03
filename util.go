package main

func tickToHour(tick int) float64 {
	return SimHours * float64(tick) / float64(TotalTicks)
}
