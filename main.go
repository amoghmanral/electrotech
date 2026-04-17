package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	policy := flag.String("policy", "greedy", "dispatch policy: greedy | reactive | predictive")
	flag.Parse()

	// channels for each component
	solarTickCh := make(chan int)
	solarOutCh := make(chan float64)

	loadTickCh := make(chan int)
	loadOutCh := make(chan float64)

	gridTickCh := make(chan int)
	gridPriceCh := make(chan float64)
	gridCmdCh := make(chan GridCmd)
	gridCostCh := make(chan float64)

	batCmdCh := make(chan BatteryCmd)
	batLevelCh := make(chan float64)
	batActualCh := make(chan float64)

	policyDataCh := make(chan TickData)
	policyBatCh := make(chan BatteryCmd)
	policyGridCh := make(chan GridCmd)

	// spin up all the goroutines
	go Solar(solarTickCh, solarOutCh)
	go Load(loadTickCh, loadOutCh)
	go Grid(gridTickCh, gridPriceCh, gridCmdCh, gridCostCh)
	go Battery(batCmdCh, batLevelCh, batActualCh)

	switch *policy {
		case "greedy":
			go PolicyGreedy(policyDataCh, policyBatCh, policyGridCh)
		case "reactive":
			go PolicyPriceReactive(policyDataCh, policyBatCh, policyGridCh)
		case "predictive":
			go PolicyPredictive(policyDataCh, policyBatCh, policyGridCh)
		default:
			fmt.Printf("unknown policy (choices: greedy | reactive | predictive)\n")
			os.Exit(1)
	}
	fmt.Printf("Policy: %s\n\n", *policy)

	// orchestrator tracks battery level separately so it can feed policy
	batteryLevel := BatteryCapacity * 0.5

	var totalCost, totalSolar, totalGridImport, totalGridExport, totalWaste float64
	var peakBattery float64

	fmt.Println("Tick | Hour  | Solar kW | Load kW | Bat kW | Bat SOC | Grid kW | Price $/kWh | Cost $  | Waste kW")
	fmt.Println("-----|-------|----------|---------|--------|---------|---------|-------------|---------|----------")

	for tick := 0; tick < TotalTicks; tick++ {
		// send tick to components
		solarTickCh <- tick
		loadTickCh <- tick
		gridTickCh <- tick

		// collect readings
		solarKW := <-solarOutCh
		loadKW := <-loadOutCh
		price := <-gridPriceCh

		// feed everything to policy
		policyDataCh <- TickData{
			Tick:         tick,
			Hour:         tickToHour(tick),
			SolarKW:      solarKW,
			LoadKW:       loadKW,
			BatteryLevel: batteryLevel,
			GridPrice:    price,
		}

		// get policy decisions
		batCmd := <-policyBatCh
		gridCmd := <-policyGridCh

		// execute battery command
		batCmdCh <- batCmd
		batteryLevel = <-batLevelCh
		batActual := <-batActualCh

		// energy balance check - any excess is wasted
		supplyMinusDemand := solarKW + gridCmd.PowerKW - loadKW - batActual
		waste := 0.0
		if supplyMinusDemand > 0.01 { // float tolerance
			waste = supplyMinusDemand
		}

		// settle grid transaction
		gridCmdCh <- gridCmd
		cost := <-gridCostCh

		// accumulate stats
		totalCost += cost
		totalSolar += solarKW / float64(TicksPerHour)
		if gridCmd.PowerKW > 0 {
			totalGridImport += gridCmd.PowerKW / float64(TicksPerHour)
		} else {
			totalGridExport += -gridCmd.PowerKW / float64(TicksPerHour)
		}
		totalWaste += waste / float64(TicksPerHour)
		if batteryLevel > peakBattery {
			peakBattery = batteryLevel
		}

		fmt.Printf("%4d | %5.1f | %8.2f | %7.2f | %+6.2f | %7.2f | %+7.2f | %11.4f | %7.4f | %8.2f\n",
			tick, tickToHour(tick), solarKW, loadKW, batActual, batteryLevel,
			gridCmd.PowerKW, price, cost, waste)
	}

	// cleanup
	close(solarTickCh)
	close(loadTickCh)
	close(gridTickCh)
	close(policyDataCh)
	close(batCmdCh)

	// summary
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("           SIMULATION SUMMARY")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Printf("  Total cost:          $%.2f\n", totalCost)
	fmt.Printf("  Total solar:         %.2f kWh\n", totalSolar)
	fmt.Printf("  Total grid import:   %.2f kWh\n", totalGridImport)
	fmt.Printf("  Total grid export:   %.2f kWh\n", totalGridExport)
	fmt.Printf("  Total waste:         %.2f kWh\n", totalWaste)
	fmt.Printf("  Peak battery level:  %.2f / %.1f kWh\n", peakBattery, BatteryCapacity)
	fmt.Println("═══════════════════════════════════════════")
}
