package sim

import "math"

// GridPrice returns the time-of-use price for a given hour ($/kWh).
//
// Curve shape mirrors California's "duck curve" which is a curve of energy
// demand from electricity grid from houses with solar panels
func GridPrice(hour float64) float64 {
	base := interpolateWaypoints(hour, duckCurveWaypoints)

	noise := 0.012*math.Sin(hour*11.7+1.3) + 0.008*math.Sin(hour*5.3+2.1)

	p := base + noise
	if p < GridPriceMin {
		p = GridPriceMin
	}
	return p
}

// duckCurveWaypoints are (hour, $/kWh) waypoints for the duck curve
var duckCurveWaypoints = []waypoint{
	{0.0, 0.09},  // midnight — low baseload rate
	{2.0, 0.08},  // overnight trough
	{5.0, 0.08},  // pre-dawn floor
	{7.0, 0.13},  // morning ramp begins
	{9.0, 0.23},  // morning demand peak (before solar flattens load)
	{11.0, 0.17}, // solar starting to suppress prices
	{13.0, 0.12}, // midday solar belly — cheapest daytime rate
	{15.0, 0.17}, // solar output beginning to fall
	{17.0, 0.27}, // evening ramp accelerating
	{18.5, 0.37}, // evening peak — solar gone, full demand on grid
	{20.0, 0.29}, // post-peak decay
	{22.0, 0.17}, // late-evening decline
	{24.0, 0.09}, // back to overnight rate
}

type waypoint struct{ h, p float64 }

// interpolateWaypoints linearly interpolates price between the nearest pair of waypoints
func interpolateWaypoints(h float64, waypoints []waypoint) float64 {
	if h <= waypoints[0].h {
		return waypoints[0].p
	}
	for i := 1; i < len(waypoints); i++ {
		if h <= waypoints[i].h {
			t := (h - waypoints[i-1].h) / (waypoints[i].h - waypoints[i-1].h)
			return waypoints[i-1].p + t*(waypoints[i].p-waypoints[i-1].p)
		}
	}
	return waypoints[len(waypoints)-1].p
}
