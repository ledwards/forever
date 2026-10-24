package spelldata

import (
	"os"
	"os/exec"
	"slices"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
)

// Level is fixed per process, so a ladder at another level runs this test binary again with
// LEVELSIM_LEVEL set and reads the child's verdict.
func runAtLevel(t *testing.T, level string, test string) (string, error) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^"+test+"$", "-test.v")
	cmd.Env = append(os.Environ(), "LEVELSIM_LEVEL="+level, "LEVELSIM_LEVEL_TEST_CHILD=1")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func inLevelChild() bool { return os.Getenv("LEVELSIM_LEVEL_TEST_CHILD") == "1" }

// Rows shaped like Mage Fireball: one spell per rank, taught at 1, 6, 12, 18, 24, 30, 36, 42, 48,
// 54, 60 and 60, the last two with the max level a row taught at 60 for real carries.
var fireballIDs = []int32{133, 143, 145, 3140, 8400, 8401, 8402, 10148, 10149, 10150, 10151, 25306}

const sentinelID = 900001

func levelRows() []Spell {
	learn := []int16{1, 6, 12, 18, 24, 30, 36, 42, 48, 54, 60, 60}
	rows := make([]Spell, 0, len(fireballIDs)+2)
	for i, id := range fireballIDs {
		rows = append(rows, Spell{
			ID: id, Name: "Fireball", Rank: "Rank " + string(rune('0'+(i+1)/10)) + string(rune('0'+(i+1)%10)),
			School: core.SpellSchoolFire, SpellLevel: learn[i], BaseLevel: learn[i], MaxLevel: learn[i] + 4,
			Effects: []Effect{{Type: dbcenums.E_SCHOOL_DAMAGE, BasePoints: 10}},
		})
	}
	// No SpellLevels data: the generator writes 60, 60, 0.
	rows = append(rows, Spell{ID: sentinelID, Name: "Sentinel", SpellLevel: 60, BaseLevel: 60, MaxLevel: 0})
	// A talent: the ladder's ranks are its points.
	rows = append(rows, Spell{ID: 11213, Name: "Arcane Concentration", SpellLevel: 1, BaseLevel: 1})
	// The store binary-searches by id.
	slices.SortFunc(rows, func(a, b Spell) int { return int(a.ID - b.ID) })
	return rows
}

func installLevelRows(t *testing.T) {
	t.Helper()
	setSpells(levelRows())
	t.Cleanup(func() { setSpells(fixture()) })
}

func TestRankedLadderAtSixty(t *testing.T) {
	installLevelRows(t)
	if inLevelChild() || os.Getenv("LEVELSIM_LEVEL") != "" {
		t.Skip("runs at the default level only")
	}
	fireball := Ranked(fireballIDs...)
	if got := fireball.Highest(); got.ID != 25306 || got.Unlearned {
		t.Fatalf("Highest at 60 = %d unlearned=%v, want 25306 learnable", got.ID, got.Unlearned)
	}
	if got := fireball.AtMost(11); got.ID != 10151 || got.Unlearned {
		t.Fatalf("AtMost(11) at 60 = %d unlearned=%v, want 10151 learnable", got.ID, got.Unlearned)
	}
	fireball.Each(func(rank int32, s *Spell) {
		if s.Unlearned {
			t.Errorf("rank %d (%d) is marked unlearned at 60", rank, s.ID)
		}
	})
}

func TestRankedLadderAtThirty(t *testing.T) {
	installLevelRows(t)
	if !inLevelChild() {
		if out, err := runAtLevel(t, "30", "TestRankedLadderAtThirty"); err != nil {
			t.Fatalf("child at 30 failed: %v\n%s", err, out)
		}
		return
	}
	fireball := Ranked(fireballIDs...)
	if got := fireball.Highest(); got.ID != 8401 || got.Unlearned {
		t.Fatalf("Highest at 30 = %d unlearned=%v, want 8401 (rank 6, taught at 30)", got.ID, got.Unlearned)
	}
	if got := fireball.AtMost(11); got.ID != 8401 || got.Unlearned {
		t.Fatalf("AtMost(11) at 30 = %d unlearned=%v, want 8401", got.ID, got.Unlearned)
	}
	if got := fireball.Rank(11); got.ID != 10151 || !got.Unlearned {
		t.Fatalf("Rank(11) at 30 = %d unlearned=%v, want 10151 marked unlearned", got.ID, got.Unlearned)
	}
	if got := fireball.Rank(6); got.ID != 8401 || got.Unlearned {
		t.Fatalf("Rank(6) at 30 = %d unlearned=%v, want 8401 learnable", got.ID, got.Unlearned)
	}
	var seen int32
	fireball.Each(func(rank int32, s *Spell) {
		seen++
		if rank != seen || s.ID != fireballIDs[rank-1] {
			t.Errorf("Each passed rank %d id %d at position %d", rank, s.ID, seen)
		}
		if want := rank > 6; s.Unlearned != want {
			t.Errorf("rank %d unlearned=%v, want %v at level 30", rank, s.Unlearned, want)
		}
	})
	if seen != 12 {
		t.Fatalf("Each passed %d ranks, want all 12", seen)
	}
	// The store's own row is never written: marking is on the ladder's answer.
	if MustFind(10151).Unlearned {
		t.Fatalf("the store row for 10151 was marked unlearned")
	}
}

func TestLaddersAtLevelOne(t *testing.T) {
	installLevelRows(t)
	if !inLevelChild() {
		if out, err := runAtLevel(t, "1", "TestLaddersAtLevelOne"); err != nil {
			t.Fatalf("child at 1 failed: %v\n%s", err, out)
		}
		return
	}
	fireball := Ranked(fireballIDs...)
	if got := fireball.Highest(); got.ID != 133 || got.Unlearned {
		t.Fatalf("Highest at 1 = %d unlearned=%v, want 133 learnable", got.ID, got.Unlearned)
	}
	if got := fireball.Rank(2); !got.Unlearned {
		t.Fatalf("Rank(2) at 1 must be unlearned")
	}
	// A spell first taught above the level answers its lowest rank, marked unlearned, so the class
	// code that registers the pick unconditionally still has a row to register.
	fromSix := Ranked(143, 145)
	if got := fromSix.Highest(); got.ID != 143 || !got.Unlearned {
		t.Fatalf("Highest of a ladder taught from 6 at level 1 = %d unlearned=%v, want 143 unlearned", got.ID, got.Unlearned)
	}
	if got := fromSix.AtMost(2); got.ID != 143 || !got.Unlearned {
		t.Fatalf("AtMost(2) of a ladder taught from 6 at level 1 = %d unlearned=%v, want 143 unlearned", got.ID, got.Unlearned)
	}
	// A row with no SpellLevels data carries 60, 60, 0 and is learnable at any level.
	if got := Ranked(sentinelID).Highest(); got.Unlearned {
		t.Fatalf("a row with no SpellLevels data is unlearned at level 1")
	}
	// A genuine level-60 row carries a MaxLevel and is not learnable below 60.
	if got := Ranked(10151).Highest(); !got.Unlearned {
		t.Fatalf("Fireball rank 11 (taught at 60, max level 64) is learnable at level 1")
	}
	// Talent ranks are talent points, not levels.
	if got := Talent(11213, 5).Rank(5); got.Unlearned {
		t.Fatalf("talent ladder rank 5 is marked unlearned at level 1")
	}
}

func TestUnlearnedSpellConfigIsNotCastable(t *testing.T) {
	installLevelRows(t)
	if !inLevelChild() {
		if out, err := runAtLevel(t, "1", "TestUnlearnedSpellConfigIsNotCastable"); err != nil {
			t.Fatalf("child at 1 failed: %v\n%s", err, out)
		}
		return
	}
	unit := &core.Unit{Level: core.CharacterLevel}
	row := Ranked(fireballIDs...).Rank(11)
	config := SpellConfig(unit, row, Magic(core.ProcMaskSpellDamage))
	if !config.Unlearned {
		t.Fatalf("config of an unlearned row is not marked unlearned")
	}
	if config.ExtraCastCondition == nil || config.ExtraCastCondition(nil, nil) {
		t.Fatalf("an unlearned row must register with a cast condition that is always false")
	}
	if learned := SpellConfig(unit, Ranked(fireballIDs...).Rank(1), Magic(core.ProcMaskSpellDamage)); learned.Unlearned {
		t.Fatalf("config of a learnable row is marked unlearned")
	}
}

// Ranked ladders tell core which ids are ranks of one spell, so the APL can fall back from a rank
// the spellbook lacks to one it has.
func TestRankedLadderRegistersSiblings(t *testing.T) {
	if inLevelChild() || os.Getenv("LEVELSIM_LEVEL") != "" {
		t.Skip("runs at the default level only")
	}
	installLevelRows(t)
	if inLevelChild() {
		t.Skip("child process")
	}
	Ranked(fireballIDs...)
	siblings := core.RankSiblings(10151)
	if len(siblings) != len(fireballIDs) || siblings[0].ID != 133 || siblings[0].LearnLevel != 1 || siblings[10].LearnLevel != 60 {
		t.Fatalf("siblings of 10151 = %v, want the 12 Fireball ranks with their learn levels", siblings)
	}
	if core.RankSiblings(777777) != nil {
		t.Fatalf("a spell no ranked ladder carries has siblings")
	}
	if core.RankTaughtAboveLevel(10151) {
		t.Fatalf("rank 11, taught at 60, counts as above level at 60")
	}
	Ranked(sentinelID)
	if core.RankTaughtAboveLevel(sentinelID) {
		t.Fatalf("a row with no SpellLevels data counts as taught above level")
	}
}
