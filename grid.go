package main

import "math"

// Grid goroutine - sends price each tick, then takes a buy/sell command
// and returns the cost
func Grid(tickCh <-chan int, priceCh chan<- float64, cmdCh <-chan GridCmd, costCh chan<- float64) {
	for tick := range tickCh {
		hour := tickToHour(tick)
		price := gridPrice(hour)
		priceCh <- price

		cmd := <-cmdCh
		energy := cmd.PowerKW / float64(TicksPerHour)
		costCh <- price * energy // positive = we pay, negative = we get paid
	}
}

// dual peak pricing - cheap overnight, expensive morning + evening rush
// uses two gaussians centered around 8am and 6:30pm
func gridPrice(hour float64) float64 {
	morningPeak := 0.22 * math.Exp(-0.5*math.Pow((hour-8.0)/1.5, 2))
	eveningPeak := 0.27 * math.Exp(-0.5*math.Pow((hour-18.5)/1.8, 2))
	return GridPriceMin + morningPeak + eveningPeak
}
