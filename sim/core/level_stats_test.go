package core

import (
	"math"
	"testing"

	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/stats"
)

func wantStats(t *testing.T, label string, got stats.Stats, want map[stats.Stat]float64) {
	t.Helper()
	for stat, value := range want {
		if math.Abs(got[stat]-value) > 0.0005 {
			t.Errorf("%s: %s = %v, want %v", label, stat.StatName(), got[stat], value)
		}
	}
}

// At 60 the hand-measured rows are the base stats, unchanged.
func TestBaseStatsAtSixtyAreTheMeasuredRows(t *testing.T) {
	if inLevelChild() || CharacterLevel != 60 {
		t.Skip("runs at the default level only")
	}
	wantStats(t, "paladin base at 60", ClassBaseStats[proto.Class_ClassPaladin], map[stats.Stat]float64{
		stats.Health: 1381, stats.Strength: 105, stats.Agility: 65, stats.Stamina: 100, stats.Intellect: 70, stats.Spirit: 75, stats.AttackPower: 160,
	})
	wantStats(t, "paladin extra at 60", ExtraClassBaseStats[proto.Class_ClassPaladin], map[stats.Stat]float64{stats.Mana: 1512, stats.SpellCritPercent: 3.5})
	if CritPerAgiMaxLevel[proto.Class_ClassPaladin] != 0.0506 {
		t.Errorf("crit per agility at 60 = %v, want 0.0506", CritPerAgiMaxLevel[proto.Class_ClassPaladin])
	}
	if CreatureHealthMaxLevel != 4365 || CreatureArmorMaxLevel != 3566 {
		t.Errorf("creature health, armor at 60 = %v, %v, want 4365, 3566", CreatureHealthMaxLevel, CreatureArmorMaxLevel)
	}
}

// At 30 every class's base row is its level 30 row, scaled so that 60 lands on the measured one,
// and the game files' per-level mana, crit slopes and creature values are the level 30 rows.
func TestBaseStatsAtThirty(t *testing.T) {
	if !inLevelChild() {
		if out, err := runAtLevel(t, "30", "TestBaseStatsAtThirty"); err != nil {
			t.Fatalf("child at 30 failed: %v\n%s", err, out)
		}
		return
	}
	wantStats(t, "human paladin at 30", BaseStats[BaseStatsKey{Race: proto.Race_RaceHuman, Class: proto.Class_ClassPaladin}], map[stats.Stat]float64{
		stats.Health: 366, stats.Strength: 55, stats.Agility: 38, stats.Stamina: 53, stats.Intellect: 40, stats.Spirit: 42,
		stats.AttackPower: 3*30 - 20, stats.Mana: 702, stats.SpellCritPercent: 3.5,
	})
	if got := CritPerAgiMaxLevel[proto.Class_ClassPaladin]; math.Abs(got-0.0935) > 0.0001 {
		t.Errorf("paladin crit per agility at 30 = %v, want 0.0935", got)
	}
	if got := CritPerIntMaxLevel[proto.Class_ClassPaladin]; math.Abs(got-0.0313) > 0.0001 {
		t.Errorf("paladin crit per intellect at 30 = %v, want 0.0313", got)
	}
	// The orc offset is the same at every level.
	wantStats(t, "orc shaman at 30 vs its class row", BaseStats[BaseStatsKey{Race: proto.Race_RaceOrc, Class: proto.Class_ClassShaman}].Subtract(ClassBaseStats[proto.Class_ClassShaman]), map[stats.Stat]float64{
		stats.Strength: 3, stats.Agility: -3, stats.Stamina: 2, stats.Intellect: -3, stats.Spirit: 3,
	})
}

func TestBaseStatsAtOne(t *testing.T) {
	if !inLevelChild() {
		if out, err := runAtLevel(t, "1", "TestBaseStatsAtOne"); err != nil {
			t.Fatalf("child at 1 failed: %v\n%s", err, out)
		}
		return
	}
	wantStats(t, "human warrior at 1", BaseStats[BaseStatsKey{Race: proto.Race_RaceHuman, Class: proto.Class_ClassWarrior}], map[stats.Stat]float64{
		stats.Health: 20, stats.Strength: 23, stats.Agility: 20, stats.Stamina: 22, stats.Intellect: 20, stats.Spirit: 20,
		stats.AttackPower: 3*1 - 20, stats.Mana: 0,
	})
	wantStats(t, "night elf hunter at 1", BaseStats[BaseStatsKey{Race: proto.Race_RaceNightElf, Class: proto.Class_ClassHunter}], map[stats.Stat]float64{
		stats.RangedAttackPower: 2*1 - 10,
	})
	if CreatureHealthMaxLevel != 42 || CreatureArmorMaxLevel != 20 {
		t.Errorf("creature health, armor at 1 = %v, %v, want 42, 20", CreatureHealthMaxLevel, CreatureArmorMaxLevel)
	}
}

func TestBaseStatsAtFortyFive(t *testing.T) {
	if !inLevelChild() {
		if out, err := runAtLevel(t, "45", "TestBaseStatsAtFortyFive"); err != nil {
			t.Fatalf("child at 45 failed: %v\n%s", err, out)
		}
		return
	}
	wantStats(t, "human mage at 45", BaseStats[BaseStatsKey{Race: proto.Race_RaceHuman, Class: proto.Class_ClassMage}], map[stats.Stat]float64{
		stats.Health: 771, stats.Intellect: 91, stats.Mana: 958, stats.AttackPower: 45 - 70,
	})
}

// A target asked to take its stats from its level gets the game files' creature row: health, armor,
// and an auto attack whose damage is the row's dps over a 2.0 swing.
func TestTargetDefaultsFromCreatureRow(t *testing.T) {
	if !inLevelChild() {
		if out, err := runAtLevel(t, "30", "TestTargetDefaultsFromCreatureRow"); err != nil {
			t.Fatalf("child at 30 failed: %v\n%s", err, out)
		}
		return
	}
	options := &proto.Target{Level: 32, MobType: proto.MobType_MobTypeBeast, StatsFromLevel: true}
	target := NewTarget(options, 0)
	wantStats(t, "level 32 beast", target.GetStats(), map[stats.Stat]float64{stats.Health: 1374, stats.Armor: 1283})
	if options.SwingSpeed != 2 || math.Abs(options.MinBaseDamage-26.08427619934*2) > 0.001 {
		t.Errorf("auto attack = %v damage over %v, want 52.169 over 2", options.MinBaseDamage, options.SwingSpeed)
	}
	// Explicit stats win, and a target that does not ask keeps whatever it stated, type or no type.
	explicit := NewTarget(&proto.Target{Level: 32, MobType: proto.MobType_MobTypeBeast, StatsFromLevel: true, Stats: stats.Stats{stats.Health: 500}.ToProtoArray()}, 0)
	if explicit.GetStats()[stats.Health] != 500 || explicit.GetStats()[stats.Armor] != 0 {
		t.Errorf("explicit stats were overridden: %v", explicit.GetStats())
	}
	unasked := &proto.Target{Level: 60, MobType: proto.MobType_MobTypeDemon}
	plain := NewTarget(unasked, 0)
	if plain.GetStats()[stats.Health] != 0 || unasked.SwingSpeed != 0 {
		t.Errorf("a target that did not ask took creature defaults: %v swing %v", plain.GetStats(), unasked.SwingSpeed)
	}
}
