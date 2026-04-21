package sim

import (
	"math"
	"math/rand"
)

// SolarBase is the noise-free solar curve scaled to the given peak kW.
// Sine bump from 6 AM to 6 PM. Used by predictive strategies for forecast.
func SolarBase(hour, peakKW float64) float64 {
	const sunrise, sunset = 6.0, 18.0
	if hour < sunrise || hour > sunset {
		return 0.0
	}
	frac := (hour - sunrise) / (sunset - sunrise)
	return peakKW * math.Sin(frac*math.Pi)
}

// SolarOutputWithRand returns the instantaneous solar output with ±5%
// noise, using a caller-supplied RNG for deterministic per-home streams.
func SolarOutputWithRand(hour, peakKW float64, r *rand.Rand) float64 {
	base := SolarBase(hour, peakKW)
	if base == 0 {
		return 0
	}
	noise := 1.0 + (r.Float64()-0.5)*0.10
	out := base * noise
	if out < 0 {
		out = 0
	}
	return out
}
