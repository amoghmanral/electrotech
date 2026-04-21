package sim

import "math/rand"

// LoadDemandWithRand samples a load draw for one tick using a caller-
// supplied RNG. Always at least the home's baseline; occasionally spikes
// up by a random fraction of its spike ceiling.
func LoadDemandWithRand(r *rand.Rand, baselineKW, spikeKW, spikeChance float64) float64 {
	demand := baselineKW
	if r.Float64() < spikeChance {
		demand += r.Float64() * spikeKW
	}
	return demand
}
