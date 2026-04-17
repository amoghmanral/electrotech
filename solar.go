package main

import (
	"math"
	"math/rand"
)

// Solar goroutine - sends solar output each tick
func Solar(tickCh <-chan int, outCh chan<- float64) {
	for tick := range tickCh {
		hour := tickToHour(tick)
		outCh <- solarOutput(hour)
	}
}

// Noise-free solar curve for predictive policy
func solarBase(hour float64) float64 {
	const sunrise, sunset = 6.0, 18.0
	if hour < sunrise || hour > sunset {
		return 0.0
	}
	fraction := (hour - sunrise) / (sunset - sunrise)
	return SolarPeakKW * math.Sin(fraction * math.Pi)
}

// sine curve from 6am to 6pm, zero otherwise
// adds a bit of noise to make it less perfect
func solarOutput(hour float64) float64 {
	const sunrise, sunset = 6.0, 18.0
	if hour < sunrise || hour > sunset {
		return 0.0
	}
	fraction := (hour - sunrise) / (sunset - sunrise)
	base := SolarPeakKW * math.Sin(fraction*math.Pi)

	// +/- 5% random noise
	noise := 1.0 + (rand.Float64()-0.5)*0.10
	output := base * noise
	if output < 0 {
		output = 0
	}
	return output
}
