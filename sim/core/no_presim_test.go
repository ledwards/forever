package core

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

// A health fight run without the presim, under a cap, has the same outcome as the presimmed run, and
// a single fight costs one iteration rather than a hundred and one.
func TestSkippingThePresimKeepsTheOutcome(t *testing.T) {
	// About 1000 health against the tester's swings is a fight of some thirty seconds.
	withPresim := deathTestRequest(0, 0, 0, 1000, &proto.SimOptions{Iterations: 500})
	without := deathTestRequest(0, 0, 0, 1000, &proto.SimOptions{Iterations: 500, SkipPresim: true, MaxFightSeconds: 600})
	a := RunSim(withPresim, nil, simsignals.CreateSignals())
	b := RunSim(without, nil, simsignals.CreateSignals())
	if a.Error != nil || b.Error != nil {
		t.Fatalf("sim failed: %v %v", a.Error, b.Error)
	}
	dpsA, dpsB := a.RaidMetrics.Dps.Avg, b.RaidMetrics.Dps.Avg
	if se := a.RaidMetrics.Dps.Stdev / math.Sqrt(500); math.Abs(dpsA-dpsB) > se {
		t.Fatalf("dps with presim %v, without %v, differ by more than one standard error %v", dpsA, dpsB, se)
	}
	if math.Abs(a.AvgIterationDuration-b.AvgIterationDuration) > 1 {
		t.Fatalf("kill time with presim %v s, without %v s", a.AvgIterationDuration, b.AvgIterationDuration)
	}

	const fights = 20
	start := time.Now()
	for i := 0; i < fights; i++ {
		RunSim(deathTestRequest(0, 0, 0, 1000, &proto.SimOptions{Iterations: 1, RandomSeed: int64(i + 1)}), nil, simsignals.CreateSignals())
	}
	presimmed := time.Since(start)
	start = time.Now()
	for i := 0; i < fights; i++ {
		RunSim(deathTestRequest(0, 0, 0, 1000, &proto.SimOptions{Iterations: 1, RandomSeed: int64(i + 1), SkipPresim: true, MaxFightSeconds: 600}), nil, simsignals.CreateSignals())
	}
	skipped := time.Since(start)
	// Alone the fights run some twenty times faster; the bound is lower because the whole suite
	// runs its packages in parallel and the timing shares the machine.
	if ratio := float64(presimmed) / float64(skipped); ratio < 5 {
		t.Fatalf("single fights: %v with presim, %v without, only %.1fx faster; want at least 5x", presimmed, skipped, ratio)
	}
}
