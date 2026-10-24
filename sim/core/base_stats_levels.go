package core

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/wowsims/forever/assets/db_inputs/basestats"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/stats"
)

// The base attributes of each class, at the level the process runs at. At 60 these are the
// hand-measured rows; below it, each class's column of the per-level table is scaled so that its
// level 60 row lands on the measured one, and attack power follows the class's level formula.
var ClassBaseStats = classBaseStatsAtLevel(CharacterLevel)

var classLevelStatColumns = map[string]stats.Stat{
	"str": stats.Strength, "agi": stats.Agility, "sta": stats.Stamina, "int": stats.Intellect, "spi": stats.Spirit, "hp": stats.Health,
}

var classLevelStatNames = map[string]proto.Class{
	"warrior": proto.Class_ClassWarrior, "paladin": proto.Class_ClassPaladin, "hunter": proto.Class_ClassHunter,
	"rogue": proto.Class_ClassRogue, "priest": proto.Class_ClassPriest, "shaman": proto.Class_ClassShaman,
	"mage": proto.Class_ClassMage, "warlock": proto.Class_ClassWarlock, "druid": proto.Class_ClassDruid,
}

// The table's rows by class and level, as the stats the columns map to.
func classLevelRows() map[proto.Class]map[int32]stats.Stats {
	lines := strings.Split(strings.TrimSpace(basestats.ClassLevelStatsTSV), "\n")
	header := strings.Split(lines[0], "\t")
	rows := map[proto.Class]map[int32]stats.Stats{}
	for _, line := range lines[1:] {
		fields := strings.Split(line, "\t")
		class, ok := classLevelStatNames[fields[0]]
		if !ok {
			panic(fmt.Sprintf("class_levelstats.tsv names an unknown class %q", fields[0]))
		}
		level, err := strconv.Atoi(fields[1])
		if err != nil {
			panic(fmt.Sprintf("class_levelstats.tsv has a bad level %q", fields[1]))
		}
		row := stats.Stats{}
		for i, name := range header {
			stat, ok := classLevelStatColumns[name]
			if !ok {
				continue
			}
			value, err := strconv.ParseFloat(fields[i], 64)
			if err != nil {
				panic(fmt.Sprintf("class_levelstats.tsv %s %d: bad %s %q", fields[0], level, name, fields[i]))
			}
			row[stat] = value
		}
		if rows[class] == nil {
			rows[class] = map[int32]stats.Stats{}
		}
		rows[class][int32(level)] = row
	}
	return rows
}

func classBaseStatsAtLevel(level int32) map[proto.Class]stats.Stats {
	if level == MaxCharacterLevel {
		return classBaseStatsAtSixty
	}
	rows := classLevelRows()
	base := map[proto.Class]stats.Stats{proto.Class_ClassUnknown: {}}
	for class, measured := range classBaseStatsAtSixty {
		if class == proto.Class_ClassUnknown {
			continue
		}
		row, ok := rows[class][level]
		if !ok {
			panic(fmt.Sprintf("class_levelstats.tsv has no level %d row for %s", level, class))
		}
		top := rows[class][MaxCharacterLevel]
		scaled := stats.Stats{}
		for _, stat := range classLevelStatColumns {
			scaled[stat] = math.Round(row[stat] * measured[stat] / top[stat])
		}
		scaled[stats.AttackPower] = baseAttackPowerAtLevel(class, level)
		if class == proto.Class_ClassHunter {
			scaled[stats.RangedAttackPower] = float64(2*level - 10)
		}
		base[class] = scaled
	}
	return base
}

// The attack power a class has before any strength or agility, by level: the level 60 rows are
// 160, 100, -10 and -20, which these lines reach at 60.
func baseAttackPowerAtLevel(class proto.Class, level int32) float64 {
	l := float64(level)
	switch class {
	case proto.Class_ClassWarrior, proto.Class_ClassPaladin:
		return 3*l - 20
	case proto.Class_ClassHunter, proto.Class_ClassRogue, proto.Class_ClassShaman:
		return 2*l - 20
	case proto.Class_ClassDruid:
		return l - 80
	default:
		return l - 70
	}
}
