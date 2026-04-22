# ElectroTech: Large-Scale Home Energy Management Simulation

---

## Introduction

A city's electricity cost can swing widely in the span of a day. According to one data report, the maximum cost of power in a day can be as much as twice the minimum cost of power (Duck Curves: US Power Price Duckiness over Time?, 2024). For households with solar panels and battery storage, this variability in power costs means they can save a lot in their power bills if they manage their energy consumption carefully.

ElectroTech is a simulator for this scenario: 1,000 homes with diverse solar and battery configurations coordinated by a central gRPC policy server, with 3 dispatch policies compared head-to-head over a full simulated day.

---

## System Design

Each home runs as an independent goroutine. Every 10 simulated minutes, it recomputes its "context": solar output, load, and battery level. Then, it calls the policy server's `Dispatch` RPC. The server returns a `DispatchDecision` (battery charge/discharge rate and grid import/export), the home applies it, and the cycle repeats. A fleet coordinator fans out all 1,000 RPCs concurrently per tick. Data is collected and streamed to a live monitoring dashboard.

The central server is stateless: `TickContext` carries both the home's instantaneous measurements (solar level, load demand, battery level, grid price) and its physical properties (battery capacity, max charge rate, solar peak). Passing these properties on every call lets a single strategy implementation serve the heterogeneous fleet with numerous home archetypes without a per-home session state on the server.

---

## Policies

**Greedy** consumes solar first, draws from the battery to cover any deficit, and falls back to grid import if need be. Surplus solar charges the battery. Any excess is exported.

**Reactive** adds a single threshold rule: if the current price is below the daily midpoint, then charge the battery as much as possible. Above the midpoint, fall back to greedy.

**Predictive** looks 2 hours ahead at both price and solar forecast. If the current price is lower than the average of the 2-hour window and expected solar won't fill the battery, charge from the grid. If the current price is higher than the forward average, discharge and export. Otherwise fall back to greedy.

---

## Evaluation and Results

We ran each strategy for all 20 archetypes over one simulated day. Total cost is grid import cost minus export revenue, summed across all archetypes.

| Strategy     | Total Cost  | Archetype Wins |
| ------------ | ----------- | -------------- |
| Greedy       | +$14.24     | 6              |
| **Reactive** | **−$23.44** | **14**         |
| Predictive   | +$35.87     | 0              |

Reactive policy wins by a huge margin, achieving net profit across the fleet. Predictive finishes last despite using more information. Greedy was not bad, and won for several archetypes with low or no battery capacity.

---

## References

Duck curves: US power price duckiness over time? (2024, November 29). Thunder Said Energy. https://thundersaidenergy.com/downloads/duck-curves-us-power-price-duckiness-over-time/
