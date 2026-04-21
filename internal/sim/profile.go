package sim

import (
	"hash/fnv"
	"math/rand"
)

// Archetype is one of a fixed set of representative home configurations —
// a studio apartment, suburban family, Tesla fanatic, farmhouse, etc.
// Each real home is assigned one archetype (deterministically from its
// seed), then small per-home jitter is applied on top.
type Archetype struct {
	Name               string
	Shape              string  // sprite shape tag: "apartment" "standard" "big" "cabin" "ranch"
	SolarPeakKW        float64 // 0 = no solar
	BatteryCapacityKWh float64 // 0 = no battery
	BatteryMaxKW       float64 // symmetric charge/discharge rate limit
	LoadBaselineKW     float64
	LoadSpikeKW        float64 // upper bound on an individual spike
	LoadSpikeChance    float64 // probability per tick
	HasEV              bool    // visual flag; also influences spike scale
}

// Archetypes is the fixed catalog. Keep this list ~20 long and concise
// so the prof can read it and recognize each one.
var Archetypes = []Archetype{
	{"studio apartment", "apartment", 0.0, 0.0, 0.0, 0.5, 2.0, 0.04, false},
	{"small apartment", "apartment", 0.0, 0.0, 0.0, 0.9, 3.0, 0.06, false},
	{"condo (solar only)", "apartment", 3.0, 0.0, 0.0, 1.1, 3.0, 0.07, false},
	{"condo + mini battery", "apartment", 3.0, 5.0, 2.5, 1.1, 3.0, 0.07, false},
	{"urban townhouse", "apartment", 3.5, 13.5, 3.0, 1.3, 3.5, 0.08, false},
	{"suburban starter", "standard", 5.0, 13.5, 5.0, 1.3, 4.5, 0.07, false},
	{"suburban family", "standard", 7.0, 13.5, 5.0, 1.8, 5.5, 0.09, false},
	{"work-from-home techie", "standard", 6.0, 13.5, 5.0, 1.7, 4.0, 0.06, false},
	{"retired couple", "standard", 5.0, 13.5, 5.0, 0.9, 3.0, 0.04, false},
	{"retrofit inefficient", "standard", 5.0, 13.5, 5.0, 2.1, 5.0, 0.10, false},
	{"new-build eco home", "standard", 9.0, 27.0, 10.0, 0.9, 3.5, 0.05, false},
	{"beach cottage w/ AC", "cabin", 4.0, 13.5, 5.0, 1.0, 5.0, 0.11, false},
	{"family + EV", "standard", 7.0, 13.5, 5.0, 1.6, 8.0, 0.12, true},
	{"family + 2 EVs", "big", 10.0, 27.0, 10.0, 1.8, 9.0, 0.14, true},
	{"big family home", "big", 9.0, 27.0, 10.0, 2.2, 6.5, 0.10, false},
	{"luxury + pool", "big", 8.0, 27.0, 10.0, 2.4, 7.5, 0.12, false},
	{"McMansion + 2 EVs", "big", 12.0, 40.5, 15.0, 2.8, 10.0, 0.16, true},
	{"Tesla fanatic", "ranch", 14.0, 40.5, 15.0, 1.8, 8.5, 0.10, true},
	{"off-grid-curious", "ranch", 15.0, 40.5, 15.0, 1.0, 4.0, 0.06, false},
	{"farmhouse", "ranch", 8.0, 27.0, 10.0, 1.8, 8.0, 0.12, false},
}

// SpriteSpec is the dashboard-side render hint: shape + palette + accessories.
type SpriteSpec struct {
	Shape      string `json:"shape"`
	WallColor  string `json:"wall"`
	RoofColor  string `json:"roof"`
	DoorColor  string `json:"door"`
	SolarCells int    `json:"solar_cells"` // 0..4 strips of panels on the roof
	HasEV      bool   `json:"has_ev"`
}

// Profile is the per-home config: archetype-sourced physical limits, with
// small jitter applied so homes aren't identical, plus sprite spec.
type Profile struct {
	ArchetypeIdx       int         `json:"archetype_idx"`
	ArchetypeName      string      `json:"archetype"`
	SolarPeakKW        float64     `json:"solar_peak_kw"`
	BatteryCapacityKWh float64     `json:"battery_capacity_kwh"`
	BatteryMaxKW       float64     `json:"battery_max_kw"`
	LoadBaselineKW     float64     `json:"load_baseline_kw"`
	LoadSpikeKW        float64     `json:"load_spike_kw"`
	LoadSpikeChance    float64     `json:"load_spike_chance"`
	Sprite             SpriteSpec  `json:"sprite"`
}

var wallPalette = []string{"#d9c99e", "#c9b080", "#9fb5a3", "#c4a48a", "#b0b9c4", "#d0a090"}
var roofPalette = []string{"#8b2e2b", "#4a3a5c", "#3c5a45", "#6a4020", "#3a4050"}
var doorPalette = []string{"#6f4a2a", "#3a5a7a", "#7a3a3a", "#3a6a4a"}

// ProfileFor returns the deterministic profile for a given home ID. The
// hash ensures the same ID maps to the same archetype + jitter every time.
func ProfileFor(homeID string) Profile {
	h := fnv.New64a()
	h.Write([]byte(homeID))
	r := rand.New(rand.NewSource(int64(h.Sum64())))

	idx := r.Intn(len(Archetypes))
	a := Archetypes[idx]

	jitter := func(v, pct float64) float64 {
		if v == 0 {
			return 0
		}
		return v * (1 - pct + 2*pct*r.Float64())
	}

	solar := jitter(a.SolarPeakKW, 0.10)
	batCap := jitter(a.BatteryCapacityKWh, 0.05)
	batMax := jitter(a.BatteryMaxKW, 0.05)
	loadBase := jitter(a.LoadBaselineKW, 0.10)
	loadSpike := jitter(a.LoadSpikeKW, 0.10)

	cells := 0
	switch {
	case a.SolarPeakKW >= 12:
		cells = 4
	case a.SolarPeakKW >= 8:
		cells = 3
	case a.SolarPeakKW >= 5:
		cells = 2
	case a.SolarPeakKW > 0:
		cells = 1
	}

	return Profile{
		ArchetypeIdx:       idx,
		ArchetypeName:      a.Name,
		SolarPeakKW:        solar,
		BatteryCapacityKWh: batCap,
		BatteryMaxKW:       batMax,
		LoadBaselineKW:     loadBase,
		LoadSpikeKW:        loadSpike,
		LoadSpikeChance:    a.LoadSpikeChance,
		Sprite: SpriteSpec{
			Shape:      a.Shape,
			WallColor:  wallPalette[r.Intn(len(wallPalette))],
			RoofColor:  roofPalette[r.Intn(len(roofPalette))],
			DoorColor:  doorPalette[r.Intn(len(doorPalette))],
			SolarCells: cells,
			HasEV:      a.HasEV,
		},
	}
}
