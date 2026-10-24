package core

import (
	"math"
	"testing"

	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/stats"
)

// The weapon tester, tanking one humanoid that swings for damage every swing seconds. The tester
// swings a ranged weapon too, so it carries the bow: an empty ranged slot is a swing of no speed.
func deathTestRequest(damage, swing, duration float64, targetHealth float64, options *proto.SimOptions) *proto.RaidSimRequest {
	raid := weaponTestRaid(proto.Race_RaceHuman, WeaponTestSlots(WeaponTestSwordID, 0, WeaponTestBowID), nil)
	raid.Tanks = []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 0}}
	target := &proto.Target{Name: "target", Level: 63, MobType: proto.MobType_MobTypeHumanoid, MinBaseDamage: damage, SwingSpeed: swing, TankIndex: 0}
	encounter := &proto.Encounter{Targets: []*proto.Target{target}, Duration: duration}
	if targetHealth > 0 {
		target.Stats = stats.Stats{stats.Health: targetHealth}.ToProtoArray()
		encounter.UseHealth = true
	}
	if options.Iterations == 0 {
		options.Iterations = 1
	}
	if options.RandomSeed == 0 {
		options.RandomSeed = 1
	}
	return &proto.RaidSimRequest{Raid: raid, Encounter: encounter, SimOptions: options}
}

func runDeathTest(t *testing.T, rsr *proto.RaidSimRequest) (*proto.RaidSimResult, *proto.UnitMetrics) {
	t.Helper()
	result := RunSim(rsr, nil, simsignals.CreateSignals())
	if result.Error != nil {
		t.Fatalf("sim failed: %s", result.Error.Message)
	}
	return result, result.RaidMetrics.Parties[0].Players[0]
}

// With end_on_player_death the iteration ends when the player's health reaches zero: the fight is
// shorter than its duration, the death is counted, and its time is the iteration's.
func TestDeathEndsTheIteration(t *testing.T) {
	result, player := runDeathTest(t, deathTestRequest(1000, 2, 60, 0, &proto.SimOptions{EndOnPlayerDeath: true}))
	if result.AvgIterationDuration >= 60 || result.AvgIterationDuration <= 0 {
		t.Fatalf("iteration lasted %v s, want a death before the 60 s duration", result.AvgIterationDuration)
	}
	if player.ChanceOfDeath != 1 {
		t.Fatalf("chance of death = %v, want 1", player.ChanceOfDeath)
	}
	if math.Abs(player.AvgDeathTimeSeconds-result.AvgIterationDuration) > 0.001 {
		t.Fatalf("death at %v s, iteration lasted %v s", player.AvgDeathTimeSeconds, result.AvgIterationDuration)
	}
}

// Without the option the corpse keeps fighting to the duration, as upstream has it.
func TestDeathWithoutTheOptionDoesNotEndTheIteration(t *testing.T) {
	result, player := runDeathTest(t, deathTestRequest(1000, 2, 60, 0, &proto.SimOptions{}))
	if result.AvgIterationDuration != 60 {
		t.Fatalf("iteration lasted %v s, want the full 60", result.AvgIterationDuration)
	}
	if player.ChanceOfDeath != 1 {
		t.Fatalf("chance of death = %v, want 1", player.ChanceOfDeath)
	}
}

// A health fight the player cannot finish ends at the death, not at the cap and not never.
func TestHealthFightEndsAtDeathUnderTheCap(t *testing.T) {
	result, player := runDeathTest(t, deathTestRequest(1000, 2, 0, 1e9, &proto.SimOptions{EndOnPlayerDeath: true, SkipPresim: true, MaxFightSeconds: 300}))
	if result.AvgIterationDuration >= 300 || result.AvgIterationDuration <= 0 {
		t.Fatalf("iteration lasted %v s, want a death well before the 300 s cap", result.AvgIterationDuration)
	}
	if player.ChanceOfDeath != 1 || math.Abs(player.AvgDeathTimeSeconds-result.AvgIterationDuration) > 0.001 {
		t.Fatalf("death %v at %v s, iteration lasted %v s", player.ChanceOfDeath, player.AvgDeathTimeSeconds, result.AvgIterationDuration)
	}
}

// A health fight whose target is never killed ends at the cap.
func TestHealthFightEndsAtTheCap(t *testing.T) {
	result, player := runDeathTest(t, deathTestRequest(0, 0, 0, 1e9, &proto.SimOptions{EndOnPlayerDeath: true, SkipPresim: true, MaxFightSeconds: 20}))
	if math.Abs(result.AvgIterationDuration-20) > 0.001 {
		t.Fatalf("iteration lasted %v s, want the 20 s cap", result.AvgIterationDuration)
	}
	if player.ChanceOfDeath != 0 {
		t.Fatalf("chance of death = %v, want 0", player.ChanceOfDeath)
	}
}

// The cap also shortens a duration fight.
func TestTheCapShortensADurationFight(t *testing.T) {
	result, _ := runDeathTest(t, deathTestRequest(0, 0, 60, 0, &proto.SimOptions{MaxFightSeconds: 10}))
	if math.Abs(result.AvgIterationDuration-10) > 0.001 {
		t.Fatalf("iteration lasted %v s, want the 10 s cap", result.AvgIterationDuration)
	}
}

// Starting at zero health with the option set is a death at time zero.
func TestStartingAtZeroHealthDiesAtOnce(t *testing.T) {
	rsr := deathTestRequest(0, 0, 60, 0, &proto.SimOptions{EndOnPlayerDeath: true})
	zero := 0.0
	rsr.Raid.Parties[0].Players[0].StartingHealthPercent = &zero
	result, player := runDeathTest(t, rsr)
	if result.AvgIterationDuration != 0 || player.ChanceOfDeath != 1 || player.AvgDeathTimeSeconds != 0 {
		t.Fatalf("iteration %v s, death %v at %v s; want 0, 1, 0", result.AvgIterationDuration, player.ChanceOfDeath, player.AvgDeathTimeSeconds)
	}
}
