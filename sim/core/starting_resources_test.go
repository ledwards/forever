package core

import (
	"math"
	"testing"

	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

// Starting health is applied at every reset, after the agent's own reset, as a share of the maximum.
func TestStartingHealthPercentAppliesAtReset(t *testing.T) {
	raid := weaponTestRaid(proto.Race_RaceHuman, WeaponTestSlots(WeaponTestSwordID, 0, WeaponTestBowID), nil)
	forty := 40.0
	raid.Parties[0].Players[0].StartingHealthPercent = &forty
	sim := NewSim(&proto.RaidSimRequest{SimOptions: &proto.SimOptions{RandomSeed: 1}, Raid: raid, Encounter: weaponTestEncounter()}, simsignals.CreateSignals())
	sim.Reset()
	character := sim.Raid.Parties[0].Players[0].GetCharacter()
	if got := character.CurrentHealthPercent(); math.Abs(got-0.4) > 0.0001 {
		t.Fatalf("health after reset = %v of max, want 0.4", got)
	}
	// The next iteration starts from the share again, not from where the last one left off.
	character.RemoveHealth(sim, character.CurrentHealth()/2)
	sim.Cleanup()
	sim.Reset()
	if got := character.CurrentHealthPercent(); math.Abs(got-0.4) > 0.0001 {
		t.Fatalf("health after a second reset = %v of max, want 0.4", got)
	}
}

// Without the field the character starts full, as upstream has it.
func TestStartingHealthDefaultsToFull(t *testing.T) {
	raid := weaponTestRaid(proto.Race_RaceHuman, WeaponTestSlots(WeaponTestSwordID, 0, WeaponTestBowID), nil)
	sim := NewSim(&proto.RaidSimRequest{SimOptions: &proto.SimOptions{RandomSeed: 1}, Raid: raid, Encounter: weaponTestEncounter()}, simsignals.CreateSignals())
	sim.Reset()
	if got := sim.Raid.Parties[0].Players[0].GetCharacter().CurrentHealthPercent(); got != 1 {
		t.Fatalf("health after reset = %v of max, want 1", got)
	}
}
