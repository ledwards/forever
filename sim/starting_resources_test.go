package sim

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// The first mana change logged for a unit whose log label contains the name, a regen tick or a
// spend: the mana before it as a share of the unit's total.
func firstManaShare(t *testing.T, logs string, name string) float64 {
	t.Helper()
	spend := regexp.MustCompile(`(?:Spent|Gained) [0-9.]+ mana from .* \(([0-9.]+) --> [0-9.]+\) of ([0-9]+) total`)
	for _, line := range strings.Split(logs, "\n") {
		if !strings.Contains(line, name) {
			continue
		}
		if m := spend.FindStringSubmatch(line); m != nil {
			before, _ := strconv.ParseFloat(m[1], 64)
			total, _ := strconv.ParseFloat(m[2], 64)
			return before / total
		}
	}
	t.Fatalf("no mana spend logged for %q:\n%s", name, logs)
	return 0
}

func debugRequest(player *proto.Player) *proto.RaidSimRequest {
	rsr := minimalRequest(player)
	rsr.SimOptions.Iterations = 1
	rsr.SimOptions.Debug = true
	return rsr
}

// Starting mana is a share of the maximum at the first cast; the owner's share leaves the pet's
// alone and the pet's leaves the owner's.
func TestStartingManaAppliesToTheOwnerAndThePetSeparately(t *testing.T) {
	mage := minimalPlayers()["mage"]
	mage.Name = "Caster"
	mage.Rotation = core.GetAplRotation("../ui/specs/mage/dps/apls", "deepFire").Rotation
	forty := 40.0
	mage.StartingManaPercent = &forty
	result := core.RunRaidSim(debugRequest(mage))
	if result.Error != nil {
		t.Fatalf("mage: %s", result.Error.Message)
	}
	if got := firstManaShare(t, result.Logs, "Caster"); math.Abs(got-0.4) > 0.01 {
		t.Fatalf("mage's first cast started at %v of max mana, want 0.4", got)
	}

	warlock := minimalPlayers()["warlock"]
	warlock.Name = "Summoner"
	warlock.Spec.(*proto.Player_Warlock).Warlock.Options.ClassOptions.Summon = proto.WarlockOptions_Imp
	warlock.Rotation = core.GetAplRotation("../ui/specs/warlock/dps/apls", "default").Rotation
	thirty := 30.0
	warlock.PetStartingResources = &proto.PetStartingResources{ManaPercent: &thirty}
	result = core.RunRaidSim(debugRequest(warlock))
	if result.Error != nil {
		t.Fatalf("warlock: %s", result.Error.Message)
	}
	if got := firstManaShare(t, result.Logs, "Imp"); math.Abs(got-0.3) > 0.01 {
		t.Fatalf("imp's first cast started at %v of max mana, want 0.3", got)
	}
	if got := firstManaShare(t, result.Logs, "Summoner"); math.Abs(got-1) > 0.01 {
		t.Fatalf("warlock's first cast started at %v of max mana, want 1", got)
	}
}
