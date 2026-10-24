package sim

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/wowsims/forever/assets/database"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

func init() {
	RegisterAll()
}

// Level is fixed per process, so a fight at another level runs this test binary again with
// LEVELSIM_LEVEL set and reads the child's verdict.
func runAtLevel(t *testing.T, level string, test string) (string, error) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^"+test+"$", "-test.v")
	cmd.Env = append(os.Environ(), "LEVELSIM_LEVEL="+level, "LEVELSIM_LEVEL_TEST_CHILD=1")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func inLevelChild() bool { return os.Getenv("LEVELSIM_LEVEL_TEST_CHILD") == "1" }

// Whether this process already runs at a chosen level, as a child of a test or under the makefile's
// test-levels loop, so a test runs its body here rather than starting a child.
func atChosenLevel() bool { return inLevelChild() || os.Getenv("LEVELSIM_LEVEL") != "" }

// One player per class with a spec, a race that can play it, no talents and no gear.
func minimalPlayers() map[string]*proto.Player {
	return map[string]*proto.Player{
		"warrior": {Class: proto.Class_ClassWarrior, Race: proto.Race_RaceHuman,
			Spec: &proto.Player_DpsWarrior{DpsWarrior: &proto.DpsWarrior{Options: &proto.DpsWarrior_Options{ClassOptions: &proto.WarriorOptions{}}}}},
		"paladin": {Class: proto.Class_ClassPaladin, Race: proto.Race_RaceHuman,
			Spec: &proto.Player_RetributionPaladin{RetributionPaladin: &proto.RetributionPaladin{Options: &proto.RetributionPaladin_Options{ClassOptions: &proto.PaladinOptions{}}}}},
		"hunter": {Class: proto.Class_ClassHunter, Race: proto.Race_RaceNightElf,
			Spec: &proto.Player_Hunter{Hunter: &proto.Hunter{Options: &proto.Hunter_Options{ClassOptions: &proto.HunterOptions{}}}}},
		"rogue": {Class: proto.Class_ClassRogue, Race: proto.Race_RaceHuman,
			Spec: &proto.Player_Rogue{Rogue: &proto.Rogue{Options: &proto.Rogue_Options{ClassOptions: &proto.RogueOptions{}}}}},
		"priest": {Class: proto.Class_ClassPriest, Race: proto.Race_RaceHuman,
			Spec: &proto.Player_DpsPriest{DpsPriest: &proto.DpsPriest{Options: &proto.DpsPriest_Options{ClassOptions: &proto.PriestOptions{}}}}},
		"shaman": {Class: proto.Class_ClassShaman, Race: proto.Race_RaceOrc,
			Spec: &proto.Player_EnhancementShaman{EnhancementShaman: &proto.EnhancementShaman{Options: &proto.EnhancementShaman_Options{ClassOptions: &proto.ShamanOptions{}}}}},
		"mage": {Class: proto.Class_ClassMage, Race: proto.Race_RaceHuman,
			Spec: &proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{ClassOptions: &proto.MageOptions{}}}}},
		"warlock": {Class: proto.Class_ClassWarlock, Race: proto.Race_RaceHuman,
			Spec: &proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.Warlock_Options{ClassOptions: &proto.WarlockOptions{}}}}},
		"druid": {Class: proto.Class_ClassDruid, Race: proto.Race_RaceNightElf,
			Spec: &proto.Player_BalanceDruid{BalanceDruid: &proto.BalanceDruid{Options: &proto.BalanceDruid_Options{ClassOptions: &proto.DruidOptions{}}}}},
	}
}

// The lowest-id one-hand sword and bow in the game files' item database. A bow is a ranged-slot
// item, not a weapon-type one.
func fixtureSword() int32 {
	return fixtureItem(func(i *proto.UIItem) bool {
		return i.Type == proto.ItemType_ItemTypeWeapon && i.WeaponType == proto.WeaponType_WeaponTypeSword && i.HandType == proto.HandType_HandTypeOneHand
	})
}
func fixtureBow() int32 {
	return fixtureItem(func(i *proto.UIItem) bool { return i.RangedWeaponType == proto.RangedWeaponType_RangedWeaponTypeBow })
}

func fixtureItem(matches func(*proto.UIItem) bool) int32 {
	var best int32
	for _, item := range database.Load().Items {
		if matches(item) && (best == 0 || item.Id < best) {
			best = item.Id
		}
	}
	if best == 0 {
		panic(fmt.Sprintf("no fixture item among %d items", len(database.Load().Items)))
	}
	return best
}

// A short duration fight against one mob of the character's level, two iterations.
func minimalRequest(player *proto.Player) *proto.RaidSimRequest {
	// A sword and a bow: a melee or ranged class with no weapon has no swing to run, which would be
	// a fixture problem rather than a level one.
	player.Equipment = &proto.EquipmentSpec{Items: core.WeaponTestSlots(fixtureSword(), 0, fixtureBow())}
	// An auto attack hands the next decision to the rotation, so a character needs one even to
	// do nothing but swing.
	if player.Rotation == nil {
		player.Rotation = &proto.APLRotation{Type: proto.APLRotation_TypeAPL}
	}
	return &proto.RaidSimRequest{
		Raid:       &proto.Raid{Parties: []*proto.Party{{Players: []*proto.Player{player}}}},
		Encounter:  &proto.Encounter{Duration: 30, Targets: []*proto.Target{{Level: core.CharacterLevel, MobType: proto.MobType_MobTypeBeast}}},
		SimOptions: &proto.SimOptions{Iterations: 2, RandomSeed: 1},
	}
}

// Every class builds and fights at the chosen level, 1 by default: the ranks its code registers
// unconditionally are taught above 1 for most spells, and each of those registers as a spell the
// character cannot cast yet.
func TestEveryClassConstructsAtLevelOne(t *testing.T) {
	if !atChosenLevel() {
		if out, err := runAtLevel(t, "1", "TestEveryClassConstructsAtLevelOne"); err != nil {
			t.Fatalf("child at level 1 failed: %v\n%s", err, out)
		}
		return
	}
	for name, player := range minimalPlayers() {
		t.Run(name, func(t *testing.T) {
			player.Name = name
			result := core.RunRaidSim(minimalRequest(player))
			if result.Error != nil {
				t.Fatalf("%s at level %d: %s", name, core.CharacterLevel, result.Error.Message)
			}
		})
	}
}

// A rotation written for 60 names ranks a lower level has not been taught; the APL falls back to
// the highest rank the spellbook can cast, so the preset still casts rather than doing nothing. The
// chosen level is 30 by default.
func TestPresetRotationCastsAtLevelThirty(t *testing.T) {
	if !atChosenLevel() {
		if out, err := runAtLevel(t, "30", "TestPresetRotationCastsAtLevelThirty"); err != nil {
			t.Fatalf("child at level 30 failed: %v\n%s", err, out)
		}
		return
	}
	player := minimalPlayers()["mage"]
	player.Name = "mage"
	player.Rotation = core.GetAplRotation("../ui/specs/mage/dps/apls", "deepFire").Rotation
	if player.Rotation == nil {
		t.Fatalf("the deepFire preset did not load")
	}
	result := core.RunRaidSim(minimalRequest(player))
	if result.Error != nil {
		t.Fatalf("mage at level %d: %s", core.CharacterLevel, result.Error.Message)
	}
	if dps := result.RaidMetrics.Dps.Avg; dps <= 0 {
		t.Fatalf("mage at level %d with the 60 preset dealt no damage; the rank fallback did not engage", core.CharacterLevel)
	}
}
