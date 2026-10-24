package core

import (
	"math"
	"testing"

	"github.com/wowsims/forever/sim/core/proto"
)

// A health fight the tester wins every iteration: every iteration is a kill, and the mean kill
// time is the mean iteration length.
func TestKillRateAndTimeOfAHealthFight(t *testing.T) {
	result, _ := runDeathTest(t, deathTestRequest(0, 2, 60, 300, &proto.SimOptions{Iterations: 20, SkipPresim: true, MaxFightSeconds: 60}))
	enc := result.EncounterMetrics
	if enc.KillRate != 1 {
		t.Fatalf("kill rate %v, want 1", enc.KillRate)
	}
	if math.Abs(enc.AvgKillTimeSeconds-result.AvgIterationDuration) > 0.001 || enc.AvgKillTimeSeconds <= 0 {
		t.Fatalf("kill time %v s, iterations lasted %v s", enc.AvgKillTimeSeconds, result.AvgIterationDuration)
	}
}

// A health fight the cap ends first has no kills and no kill time, and a duration fight reports
// neither.
func TestNoKillWhenTheCapOrTheDurationEndsTheFight(t *testing.T) {
	capped, _ := runDeathTest(t, deathTestRequest(0, 2, 60, 1e9, &proto.SimOptions{Iterations: 3, SkipPresim: true, MaxFightSeconds: 5}))
	if capped.EncounterMetrics.KillRate != 0 || capped.EncounterMetrics.AvgKillTimeSeconds != 0 || capped.AvgIterationDuration != 5 {
		t.Fatalf("capped fight: kill rate %v at %v s, iterations %v s", capped.EncounterMetrics.KillRate, capped.EncounterMetrics.AvgKillTimeSeconds, capped.AvgIterationDuration)
	}
	timed, _ := runDeathTest(t, deathTestRequest(0, 2, 10, 0, &proto.SimOptions{Iterations: 3}))
	if timed.EncounterMetrics.KillRate != 0 || timed.EncounterMetrics.AvgKillTimeSeconds != 0 {
		t.Fatalf("duration fight: kill rate %v at %v s", timed.EncounterMetrics.KillRate, timed.EncounterMetrics.AvgKillTimeSeconds)
	}
}

// The health left at the end of an iteration is reported as a fraction: a tester hit for a little
// over a short fight ends below full and above zero, and a mana-less tester reports no mana.
func TestEndOfIterationResources(t *testing.T) {
	_, player := runDeathTest(t, deathTestRequest(50, 2, 10, 0, &proto.SimOptions{Iterations: 5}))
	if player.AvgEndHealthPercent <= 0 || player.AvgEndHealthPercent >= 1 {
		t.Fatalf("end health %v, want a fraction strictly between 0 and 1", player.AvgEndHealthPercent)
	}
	if player.AvgEndManaPercent != 0 {
		t.Fatalf("end mana %v for a unit without mana", player.AvgEndManaPercent)
	}
	_, dead := runDeathTest(t, deathTestRequest(1000, 2, 60, 0, &proto.SimOptions{Iterations: 2, EndOnPlayerDeath: true}))
	if dead.AvgEndHealthPercent != 0 {
		t.Fatalf("end health %v for a unit that died every iteration", dead.AvgEndHealthPercent)
	}
}

// Split and combined results carry the same kill and end-resource figures as one run.
func TestKillAndEndResourceFiguresCombine(t *testing.T) {
	// Damage that kills the tester in some iterations before the target's health is spent.
	request := deathTestRequest(200, 2, 60, 600, &proto.SimOptions{Iterations: 60, SkipPresim: true, MaxFightSeconds: 60, EndOnPlayerDeath: true})
	whole, _ := runDeathTest(t, request)
	split := SplitSimRequestForConcurrency(request, 4)
	if split.ErrorResult != "" {
		t.Fatal(split.ErrorResult)
	}
	parts := make([]*proto.RaidSimResult, len(split.Requests))
	for i, req := range split.Requests {
		parts[i], _ = runDeathTest(t, req)
	}
	combined := CombineConcurrentSimResults(parts, false)
	if whole.EncounterMetrics.KillRate <= 0 || whole.EncounterMetrics.KillRate >= 1 {
		t.Fatalf("the fixture should kill in some iterations and die in others, kill rate %v", whole.EncounterMetrics.KillRate)
	}
	for _, c := range []struct {
		name       string
		want, have float64
	}{
		{"kill rate", whole.EncounterMetrics.KillRate, combined.EncounterMetrics.KillRate},
		{"kill time", whole.EncounterMetrics.AvgKillTimeSeconds, combined.EncounterMetrics.AvgKillTimeSeconds},
		{"end health", whole.RaidMetrics.Parties[0].Players[0].AvgEndHealthPercent, combined.RaidMetrics.Parties[0].Players[0].AvgEndHealthPercent},
	} {
		if math.Abs(c.want-c.have) > 1e-9 {
			t.Errorf("%s: whole %v, combined %v", c.name, c.want, c.have)
		}
	}
}
