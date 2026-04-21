package sim

import "math"

// GridPrice returns the time-of-use price for a given hour ($/kWh).
// Dual-Gaussian peaks at 8 AM (morning) and 6:30 PM (evening) over a
// fixed overnight floor.
func GridPrice(hour float64) float64 {
	morning := 0.22 * math.Exp(-0.5*math.Pow((hour-8.0)/1.5, 2))
	evening := 0.27 * math.Exp(-0.5*math.Pow((hour-18.5)/1.8, 2))
	return GridPriceMin + morning + evening
}
