package main

import "math/rand"

// Load goroutine - sends household demand each tick
func Load(tickCh <-chan int, outCh chan<- float64) {
	for range tickCh {
		outCh <- loadDemand()
	}
}

// constant baseline + random spikes (appliances turning on etc)
func loadDemand() float64 {
	demand := LoadBaselineKW
	if rand.Float64() < LoadSpikeChance {
		demand += rand.Float64() * LoadMaxSpikeKW
	}
	return demand
}
