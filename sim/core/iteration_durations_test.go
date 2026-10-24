package core

import (
	"math"
	"testing"

	"github.com/wowsims/forever/sim/core/proto"
)

// With save_all_values the result lists each iteration's length, which average to the mean the
// result already reports; without it the list is empty.
func TestIterationDurationsAreSavedOnRequest(t *testing.T) {
	result, _ := runDeathTest(t, deathTestRequest(0, 2, 60, 300, &proto.SimOptions{Iterations: 12, SkipPresim: true, MaxFightSeconds: 60, SaveAllValues: true}))
	if len(result.IterationDurations) != 12 {
		t.Fatalf("%d durations for 12 iterations", len(result.IterationDurations))
	}
	sum := 0.0
	for _, d := range result.IterationDurations {
		if d <= 0 {
			t.Fatalf("an iteration lasted %v s", d)
		}
		sum += d
	}
	if math.Abs(sum/12-result.AvgIterationDuration) > 1e-9 {
		t.Fatalf("durations average %v s, result says %v s", sum/12, result.AvgIterationDuration)
	}
	plain, _ := runDeathTest(t, deathTestRequest(0, 2, 60, 300, &proto.SimOptions{Iterations: 3, SkipPresim: true, MaxFightSeconds: 60}))
	if len(plain.IterationDurations) != 0 {
		t.Fatalf("durations saved without the option")
	}
}

// Iterations run in batches with consecutive seeds are the iterations of one run: a run of n with
// seed s equals runs of n/2 with seeds s and s + n/2, value for value.
func TestBatchesWithConsecutiveSeedsAreOneRun(t *testing.T) {
	options := func(seed int64, n int32) *proto.SimOptions {
		return &proto.SimOptions{Iterations: n, RandomSeed: seed, SkipPresim: true, MaxFightSeconds: 60, SaveAllValues: true}
	}
	whole, player := runDeathTest(t, deathTestRequest(0, 2, 60, 300, options(5, 10)))
	a, pa := runDeathTest(t, deathTestRequest(0, 2, 60, 300, options(5, 5)))
	b, pb := runDeathTest(t, deathTestRequest(0, 2, 60, 300, options(10, 5)))
	parts := append(append([]float64{}, pa.Dps.AllValues...), pb.Dps.AllValues...)
	for i, v := range player.Dps.AllValues {
		if v != parts[i] {
			t.Fatalf("iteration %d: whole %v, batches %v", i, v, parts[i])
		}
	}
	durations := append(append([]float64{}, a.IterationDurations...), b.IterationDurations...)
	for i, d := range whole.IterationDurations {
		if d != durations[i] {
			t.Fatalf("iteration %d: whole lasted %v, batches %v", i, d, durations[i])
		}
	}
}
