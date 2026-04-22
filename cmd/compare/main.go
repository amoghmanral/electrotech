// compare runs all three dispatch strategies over one simulated day for every
// home archetype and prints a cost comparison table
package main

import (
	"fmt"
	"math/rand"

	"github.com/amoghmanral/electrotech/internal/sim"
)

var strategies = []string{"greedy", "reactive", "predictive"}

func simulateCost(strategy string, a sim.Archetype, seed int64) float64 {
	r := rand.New(rand.NewSource(seed))
	batteryLevel := a.BatteryCapacityKWh / 2.0 // start at 50%
	energyHour := 1.0 / float64(sim.TicksPerHour)
	var totalCost float64

	for t := 0; t < sim.TotalTicks; t++ {
		hour := sim.TickToHour(t)
		solarKW := sim.SolarOutputWithRand(hour, a.SolarPeakKW, r)
		loadKW := sim.LoadDemandWithRand(r, a.LoadBaselineKW, a.LoadSpikeKW, a.LoadSpikeChance)
		price := sim.GridPrice(hour)

		td := sim.TickData{
			Tick:               t,
			Hour:               hour,
			SolarKW:            solarKW,
			LoadKW:             loadKW,
			BatteryLevel:       batteryLevel,
			GridPrice:          price,
			BatteryCapacityKWh: a.BatteryCapacityKWh,
			BatteryMaxKW:       a.BatteryMaxKW,
			SolarPeakKW:        a.SolarPeakKW,
		}
		batCmd, gridCmd, _ := sim.Dispatch(strategy, td)
		_, batteryLevel = sim.ApplyBattery(batCmd, batteryLevel, a.BatteryCapacityKWh, a.BatteryMaxKW)
		totalCost += gridCmd.PowerKW * energyHour * price
	}
	return totalCost
}

func main() {
	const seed = 390

	// header
	fmt.Printf("%-30s", "Archetype")
	for _, s := range strategies {
		fmt.Printf("  %12s", s)
	}
	fmt.Printf("  %12s  %12s\n", "best", "savings")
	fmt.Printf("%s\n", repeat("-", 30+3*14+2*14))

	var totals [3]float64
	for _, a := range sim.Archetypes {
		costs := [3]float64{}
		for i, s := range strategies {
			costs[i] = simulateCost(s, a, seed)
		}

		best, bestIdx := costs[0], 0
		for i, c := range costs {
			if c < best {
				best, bestIdx = c, i
			}
			totals[i] += c
		}
		savings := costs[0] - best

		fmt.Printf("%-30s", a.Name)
		for _, c := range costs {
			fmt.Printf("  %+12.4f", c)
		}
		fmt.Printf("  %12s  %+12.4f\n", strategies[bestIdx], savings)
	}

	// totals
	fmt.Printf("%s\n", repeat("-", 30+3*14+2*14))
	fmt.Printf("%-30s", "TOTAL")
	for _, t := range totals {
		fmt.Printf("  %+12.4f", t)
	}
	fmt.Println()
}

func repeat(s string, n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = s[0]
	}
	return string(out)
}
