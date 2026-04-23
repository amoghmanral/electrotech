package sim

import "math/rand"

// Real predictive energy management doesn't have access to true future
// prices and solar output. So, we wrap the ground-truth curves
// with ±10% random noise so PredictiveDispatch sees a realistic forecast
// instead of the exact future.

const forecastNoise = 0.10 // ±10% on both price and solar forecasts

func noisyFactor(seed int64) float64 {
	r := rand.New(rand.NewSource(seed))
	return 1 + (r.Float64()*2-1)*forecastNoise
}

// ForecastPrice returns a noisy forecast of grid price at currentTick+horizonTicks.
func ForecastPrice(currentTick, horizonTicks int) float64 {
	truth := GridPrice(TickToHour(currentTick + horizonTicks))
	p := truth * noisyFactor(int64(currentTick)*1_000_003+int64(horizonTicks))
	if p < GridPriceMin {
		p = GridPriceMin
	}
	return p
}

// ForecastSolar returns a noisy forecast of solar output at currentTick+horizonTicks
// for a given peak. Zero at night to make sure noise does not produce positive solar output at night.
func ForecastSolar(currentTick, horizonTicks int, peakKW float64) float64 {
	truth := SolarBase(TickToHour(currentTick+horizonTicks), peakKW)
	if truth == 0 {
		return 0
	}
	out := truth * noisyFactor(int64(currentTick)*1_000_003+int64(horizonTicks)+7919)
	if out < 0 {
		return 0
	}
	return out
}
