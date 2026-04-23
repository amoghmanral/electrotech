// compare runs all three dispatch strategies over one simulated day for every
// home archetype and prints a cost comparison table. Monte Carlo: each
// (strategy, archetype) is replayed over many seeds so we can report mean,
// std dev, and the stability of the per-archetype winner.
package main

import (
	"flag"
	"fmt"
	"math"
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

func meanStddev(xs []float64) (mean, stddev float64) {
	if len(xs) == 0 {
		return 0, 0
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	mean = sum / float64(len(xs))
	var sq float64
	for _, x := range xs {
		d := x - mean
		sq += d * d
	}
	stddev = math.Sqrt(sq / float64(len(xs)))
	return
}

func main() {
	nSeeds := flag.Int("seeds", 100, "number of Monte Carlo seeds")
	baseSeed := flag.Int64("seed", 390, "base seed (seeds run baseSeed..baseSeed+N-1)")
	flag.Parse()

	fmt.Printf("Monte Carlo policy comparison: %d seeds per (strategy, archetype)\n\n", *nSeeds)

	// header
	fmt.Printf("%-26s", "Archetype")
	for _, s := range strategies {
		fmt.Printf("  %18s", s)
	}
	fmt.Printf("  %12s  %9s\n", "best", "stability")
	fmt.Printf("%s\n", repeat("-", 26+3*20+14+11))

	// totals[strategyIdx] = per-seed total across all archetypes
	totals := make([][]float64, 3)
	for i := range totals {
		totals[i] = make([]float64, *nSeeds)
	}

	for _, a := range sim.Archetypes {
		runs := make([][]float64, 3)
		for i := range runs {
			runs[i] = make([]float64, *nSeeds)
		}
		for si := 0; si < *nSeeds; si++ {
			seed := *baseSeed + int64(si)
			for i, s := range strategies {
				cost := simulateCost(s, a, seed)
				runs[i][si] = cost
				totals[i][si] += cost
			}
		}

		means := make([]float64, 3)
		sds := make([]float64, 3)
		for i := 0; i < 3; i++ {
			means[i], sds[i] = meanStddev(runs[i])
		}

		// best by mean
		bestIdx := 0
		for i := 1; i < 3; i++ {
			if means[i] < means[bestIdx] {
				bestIdx = i
			}
		}

		// stability: fraction of seeds where bestIdx is also the per-seed winner
		stable := 0
		for si := 0; si < *nSeeds; si++ {
			localBest := 0
			for i := 1; i < 3; i++ {
				if runs[i][si] < runs[localBest][si] {
					localBest = i
				}
			}
			if localBest == bestIdx {
				stable++
			}
		}
		stability := 100.0 * float64(stable) / float64(*nSeeds)

		fmt.Printf("%-26s", a.Name)
		for i := range strategies {
			fmt.Printf("  %+9.3f ± %6.3f", means[i], sds[i])
		}
		fmt.Printf("  %12s  %8.0f%%\n", strategies[bestIdx], stability)
	}

	// totals across archetypes
	fmt.Printf("%s\n", repeat("-", 26+3*20+14+11))
	fmt.Printf("%-26s", "TOTAL")
	for i := range strategies {
		m, sd := meanStddev(totals[i])
		fmt.Printf("  %+9.2f ± %6.2f", m, sd)
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
