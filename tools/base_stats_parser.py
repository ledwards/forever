#!/usr/bin/python

import csv
import os
import re
import sqlite3
from pathlib import Path

# Generates sim/core/base_stats_auto_gen.go from the data extracted from the game files.
#
# Two sources:
#
#   tools/database/wowsims.db     PlayerExpectedStat -- base mana and the
#                                 crit-per-agility / crit-per-intellect slopes;
#                                 ExpectedStat -- the creature health, armor
#                                 and auto attack dps of each level.
#                                 `make db` produces it; when it is absent the
#                                 same two tables are read from the CSV exports
#                                 beside the gametables, and the generated
#                                 header says which was read.
#   assets/db_inputs/basestats/   the CombatRatings gametable, the two
#                                 base-crit tables described below, and the
#                                 two CSV exports
#
# Every level from 1 to the character level is emitted, plus the creature rows
# up to three levels above it; the sim indexes them by the level the process
# runs at.
#
# The per-level crit slope tables (chancetomeleecrit / chancetospellcrit) and
# octbasempbyclass used to supply the first three quantities, but the beta
# game files do not ship those gametables -- PlayerExpectedStat replaced them,
# and it disagrees: Forever retuned crit-per-agility for seven of the nine
# classes (hunter 0.0189 where the old tables said 0.0301).

BASE_DIR = ""

DIR_PATH = "assets/db_inputs/basestats/"
OUTPUT_PATH = "sim/core/"
TS_OUTPUT_PATH = "ui/sim/constants/mechanics.ts"
CONSTANTS_PATH = "sim/core/constants.go"
DB_PATH = "tools/database/wowsims.db"

# Neither the Forever nor the Era game files ship the two base-crit gametables,
# so these stay checked in by hand. They hold the vanilla per-class crit a
# character has before any agility/intellect contribution; warrior, hunter and
# rogue have no base melee crit. The Death Knight and Monk columns are unused.
MELEE_CRIT_BASE = "chancetomeleecritbase.txt"
SPELL_CRIT_BASE = "chancetospellcritbase.txt"
COMBAT_RATINGS = "combatratings.txt"

# Read out of sim/core/constants.go rather than repeated here, so there is one place
# that says what level this sim runs at by default. The level is a variable the process
# reads from LEVELSIM_LEVEL at init; the generator reads the default in its initializer.
def GoConstant(name: str):
    text = open(BASE_DIR + CONSTANTS_PATH).read()
    match = re.search(rf"^const {name} = ([\d.]+)$", text, re.M)
    if match is None:
        match = re.search(rf"^var {name} int32 = characterLevelFromEnvironment\((\d+)\)$", text, re.M)
    return float(match.group(1))


BASE_LEVEL = int(GoConstant("CharacterLevel"))

# The boss is three levels above the player; see defaultRaidBossLevel in sim/core/target.go.
BOSS_LEVEL_OFFSET = 3

PLAYER_EXPECTED_CSV = "PlayerExpectedStat.csv"
EXPECTED_CSV = "ExpectedStat.csv"

# ChrClasses.ID -> the proto.Class suffix. Hardcoded rather than read from the
# game files so the generated identifiers cannot silently follow a renumbering.
CLASS_IDS = {
    1: "Warrior",
    2: "Paladin",
    3: "Hunter",
    4: "Rogue",
    5: "Priest",
    7: "Shaman",
    8: "Mage",
    9: "Warlock",
    11: "Druid",
}

CLASSES = ["Warrior", "Paladin", "Hunter", "Rogue", "Priest", "Shaman", "Mage", "Warlock", "Druid"]


# Returns {row key: {column name: value}}. Indexing by header name rather than
# by position survives the column order changes the game files make between
# expansions -- the beta's SpellScaling grew Evoker/Adventurer/Traveler columns.
def GenNameIndexedDb(file: str):
    db = {}
    with open(file) as tsv:
        rows = list(csv.reader(tsv, delimiter="\t"))
    header = rows[0]
    for row in rows[1:]:
        db[row[0]] = dict(zip(header[1:], row[1:]))
    return db


def ReadPlayerExpectedRows(dbPath: str, csvPath: str):
    """(level, classID, baseMana, critPerAgi, critPerInt) for every row with ContentSetID 0."""
    if os.path.exists(dbPath):
        conn = sqlite3.connect(dbPath)
        rows = conn.execute(
            """SELECT Level, ClassID, BaseMana, CritPerAgility, SpellCritPerIntellect
               FROM PlayerExpectedStat WHERE ContentSetID = 0"""
        ).fetchall()
        conn.close()
        return "tools/database/wowsims.db", [(int(l), int(c), float(m), float(a), float(i)) for l, c, m, a, i in rows]
    with open(csvPath, newline="") as f:
        rows = [(int(r["Level"]), int(r["ClassID"]), float(r["BaseMana"]), float(r["CritPerAgility"]), float(r["SpellCritPerIntellect"]))
                for r in csv.DictReader(f) if int(r["ContentSetID"]) == 0]
    return csvPath, rows


def ReadCreatureExpectedRows(dbPath: str, csvPath: str):
    """(level, health, armor, autoAttackDps) for every row with ContentSetID 0."""
    if os.path.exists(dbPath):
        conn = sqlite3.connect(dbPath)
        rows = conn.execute(
            """SELECT Lvl, CreatureHealth, CreatureArmor, CreatureAutoAttackDps
               FROM ExpectedStat WHERE ContentSetID = 0"""
        ).fetchall()
        conn.close()
        return [(int(l), float(h), float(a), float(d)) for l, h, a, d in rows]
    with open(csvPath, newline="") as f:
        return [(int(r["Lvl"]), float(r["CreatureHealth"]), float(r["CreatureArmor"]), float(r["CreatureAutoAttackDps"]))
                for r in csv.DictReader(f) if int(r["ContentSetID"]) == 0]


def GenExpectedStatsByLevel(dbPath: str, csvPath: str, maxLevel: int):
    """{level: {class: {Mana, CritPerAgi, CritPerInt}}} for levels 1..maxLevel, failing on a gap."""
    source, rows = ReadPlayerExpectedRows(dbPath, csvPath)
    byLevel = {}
    for level, classID, baseMana, critPerAgi, critPerInt in rows:
        if classID not in CLASS_IDS or not 1 <= level <= maxLevel:
            continue
        # The game files store these as a fraction per point (0.0005 = one
        # percent per 20 agility); the sim's stat dependencies are in
        # percent per point.
        byLevel.setdefault(level, {})[CLASS_IDS[classID]] = {
            "Mana": baseMana,
            "CritPerAgi": critPerAgi * 100,
            "CritPerInt": critPerInt * 100,
        }
    for level in range(1, maxLevel + 1):
        missing = [c for c in CLASSES if c not in byLevel.get(level, {})]
        if missing:
            raise SystemExit(f"PlayerExpectedStat has no level {level} row for: {', '.join(missing)}")
    return source, byLevel


def GenCreatureExpectedStatsByLevel(dbPath: str, csvPath: str, maxLevel: int):
    """{level: {Health, Armor, AutoAttackDps}} for levels 1..maxLevel, failing on a gap."""
    byLevel = {}
    for level, health, armor, autoAttackDps in ReadCreatureExpectedRows(dbPath, csvPath):
        if 1 <= level <= maxLevel:
            if level in byLevel:
                raise SystemExit(f"ExpectedStat has more than one level {level} row")
            byLevel[level] = {"Health": health, "Armor": armor, "AutoAttackDps": autoAttackDps}
    for level in range(1, maxLevel + 1):
        if level not in byLevel:
            raise SystemExit(f"ExpectedStat has no level {level} row")
    return byLevel


class ClassStats:
    Source: str
    ExpectedByLevel: dict
    CreatureByLevel: dict
    MCritBase: dict
    SCritBase: dict
    CombatRatings: dict


# The sim has one hit, crit and haste rating that feeds melee, ranged and spells alike, so the
# gametable's three columns for each must convert at the same rate.
def SharedRating(ratings, kind: str) -> float:
    columns = {f"{kind} - {channel}": float(ratings[f"{kind} - {channel}"]) for channel in ("Melee", "Ranged", "Spell")}
    if len(set(columns.values())) != 1:
        raise SystemExit(f"{kind} rating converts differently per channel at level {BASE_LEVEL}: {columns}")
    return next(iter(columns.values()))


def GenExtraStatsGoFile(cs: ClassStats):
    header = '''
package core

// **************************************
// AUTO GENERATED BY BASE_STATS_PARSER.PY
// **************************************

import (
    "github.com/wowsims/forever/sim/core/proto"
    "github.com/wowsims/forever/sim/core/stats"
)

'''
    output = header
    ratings = cs.CombatRatings[str(BASE_LEVEL)]

    # Rating per whole percent, like the hit and crit constants below. ExpertiseRating feeds
    # ExpertisePercent through a stat dependency; there is no quarter-point rounding in Forever.
    output += f"const ExpertiseRatingPerExpertisePercent = {float(ratings['Expertise']):f}\n"
    output += f"const DefenseRatingPerDefenseLevel = {float(ratings['Defense Skill']):f}\n"
    output += f"const DodgeRatingPerDodgePercent = {float(ratings['Dodge']):f}\n"
    output += f"const ParryRatingPerParryPercent = {float(ratings['Parry']):f}\n"
    output += f"const BlockRatingPerBlockPercent = {float(ratings['Block']):f}\n"
    output += f"const HitRatingPerHitPercent = {SharedRating(ratings, 'Hit'):f}\n"
    output += f"const CritRatingPerCritPercent = {SharedRating(ratings, 'Crit'):f}\n"
    output += f"const HasteRatingPerHastePercent = {SharedRating(ratings, 'Haste'):f}\n"

    output += "// The game files' rows for every level, indexed by the level the process runs at.\n"
    output += f"// Read from {cs.Source}.\n\n"

    output += "type creatureRow struct {\n Health, Armor, AutoAttackDps float64\n}\n\n"
    output += "var creatureRowByLevel = map[int32]creatureRow{"
    for level in sorted(cs.CreatureByLevel):
        row = cs.CreatureByLevel[level]
        output += f"\n{level}: {{Health: {row['Health']!r}, Armor: {row['Armor']!r}, AutoAttackDps: {row['AutoAttackDps']!r}}},"
    output += "\n}\n\n"

    output += "var critPerAgiByLevel = map[int32]map[proto.Class]float64{"
    for level in sorted(cs.ExpectedByLevel):
        output += f"\n{level}: {{proto.Class_ClassUnknown: 0.0,"
        for c in CLASSES:
            output += f" proto.Class_Class{c}: {cs.ExpectedByLevel[level][c]['CritPerAgi']:.8f},"
        output += "},"
    output += "\n}\n\n"

    output += "var critPerIntByLevel = map[int32]map[proto.Class]float64{"
    for level in sorted(cs.ExpectedByLevel):
        output += f"\n{level}: {{proto.Class_ClassUnknown: 0.0,"
        for c in CLASSES:
            output += f" proto.Class_Class{c}: {cs.ExpectedByLevel[level][c]['CritPerInt']:.8f},"
        output += "},"
    output += "\n}\n\n"

    output += "var baseManaByLevel = map[int32]map[proto.Class]float64{"
    for level in sorted(cs.ExpectedByLevel):
        output += f"\n{level}: {{"
        for c in CLASSES:
            output += f" proto.Class_Class{c}: {float(cs.ExpectedByLevel[level][c]['Mana']):.4f},"
        output += "},"
    output += "\n}\n\n"

    output += "// Each class's crit before any agility or intellect, in percent.\n"
    output += "var spellCritBase = map[proto.Class]float64{"
    for c in CLASSES:
        output += f"\nproto.Class_Class{c}: {float(cs.SCritBase['1'][c]) * 100:.4f},"
    output += "\n}\n\n"
    output += "var physicalCritBase = map[proto.Class]float64{"
    for c in CLASSES:
        output += f"\nproto.Class_Class{c}: {float(cs.MCritBase['1'][c]) * 100:.4f},"
    output += "\n}\n\n"

    output += "// ExpectedStat's creature values at the character level.\n"
    output += "var CreatureHealthMaxLevel = creatureRowByLevel[CharacterLevel].Health\n"
    output += "var CreatureArmorMaxLevel = creatureRowByLevel[CharacterLevel].Armor\n"
    output += "var CreatureAutoAttackDpsMaxLevel = creatureRowByLevel[CharacterLevel].AutoAttackDps\n\n"
    output += "// The creature row of a level, for a target of that level, and whether the game files carry it.\n"
    output += "func CreatureRowAt(level int32) (creatureRow, bool) {\n row, ok := creatureRowByLevel[level]\n return row, ok\n}\n\n"
    output += "var CritPerAgiMaxLevel = critPerAgiByLevel[CharacterLevel]\n"
    output += "var CritPerIntMaxLevel = critPerIntByLevel[CharacterLevel]\n\n"
    output += "// Base mana and base crit at the character level, added to each class's base row.\n"
    output += "var ExtraClassBaseStats = extraClassBaseStatsAtLevel(CharacterLevel)\n\n"
    output += "func extraClassBaseStatsAtLevel(level int32) map[proto.Class]stats.Stats {\n"
    output += " extra := map[proto.Class]stats.Stats{proto.Class_ClassUnknown: {}}\n"
    output += " for class, mana := range baseManaByLevel[level] {\n"
    output += "  extra[class] = stats.Stats{stats.Mana: mana, stats.SpellCritPercent: spellCritBase[class], stats.PhysicalCritPercent: physicalCritBase[class]}\n"
    output += " }\n return extra\n}\n"
    return output


# ui/sim/constants/mechanics.ts used to be a hand-kept copy of these numbers, and it drifted:
# it was still converting spell hit at 8 and defense at 1.5 after the game files moved both. It is
# generated from the same values as the Go file now, so the two cannot disagree.
def GenMechanicsTsFile(cs: ClassStats):
    ratings = cs.CombatRatings[str(BASE_LEVEL)]

    def num(value):
        return f"{float(value):g}"

    lines = [
        "// **************************************",
        "// AUTO GENERATED BY BASE_STATS_PARSER.PY",
        "//",
        "// Do not edit. These mirror sim/core, and a Go test fails if they drift:",
        "// TestMechanicsConstantsMatchTheUI in sim/core/mechanics_mirror_test.go.",
        "// **************************************",
        "",
        f"export const CHARACTER_LEVEL = {BASE_LEVEL};",
        f"export const BOSS_LEVEL = CHARACTER_LEVEL + {BOSS_LEVEL_OFFSET};",
        "",
        f"export const EXPERTISE_RATING_PER_EXPERTISE_PERCENT = {num(ratings['Expertise'])};",
        f"export const HASTE_RATING_PER_HASTE_PERCENT = {num(SharedRating(ratings, 'Haste'))};",
        f"export const CRIT_RATING_PER_CRIT_PERCENT = {num(SharedRating(ratings, 'Crit'))};",
        f"export const HIT_RATING_PER_HIT_PERCENT = {num(SharedRating(ratings, 'Hit'))};",
        f"export const DODGE_RATING_PER_DODGE_PERCENT = {num(ratings['Dodge'])};",
        f"export const PARRY_RATING_PER_PARRY_PERCENT = {num(ratings['Parry'])};",
        f"export const DEFENSE_RATING_PER_DEFENSE_LEVEL = {num(ratings['Defense Skill'])};",
        f"export const BLOCK_RATING_PER_BLOCK_PERCENT = {num(ratings['Block'])};",
        "",
        f"export const MISS_DODGE_PARRY_BLOCK_CRIT_CHANCE_PER_DEFENSE = {num(GoConstant('MissDodgeParryBlockCritChancePerDefense'))};",
        "",
    ]
    return "\n".join(lines)


if __name__ == "__main__":
    args = ClassStats()
    args.Source, args.ExpectedByLevel = GenExpectedStatsByLevel(BASE_DIR + DB_PATH, BASE_DIR + DIR_PATH + PLAYER_EXPECTED_CSV, BASE_LEVEL)
    args.CreatureByLevel = GenCreatureExpectedStatsByLevel(BASE_DIR + DB_PATH, BASE_DIR + DIR_PATH + EXPECTED_CSV, BASE_LEVEL + BOSS_LEVEL_OFFSET)
    args.MCritBase = GenNameIndexedDb(BASE_DIR + DIR_PATH + MELEE_CRIT_BASE)
    args.SCritBase = GenNameIndexedDb(BASE_DIR + DIR_PATH + SPELL_CRIT_BASE)
    args.CombatRatings = GenNameIndexedDb(BASE_DIR + DIR_PATH + COMBAT_RATINGS)

    output = GenExtraStatsGoFile(args)
    fname = BASE_DIR + OUTPUT_PATH + "base_stats_auto_gen.go"
    print(f"Writing stats to: {fname}")
    f = open(fname, "w")
    f.write(output)
    f.close()
    os.system(f"gofmt -w {fname}")

    tsname = BASE_DIR + TS_OUTPUT_PATH
    print(f"Writing mechanics to: {tsname}")
    with open(tsname, "w") as tsfile:
        tsfile.write(GenMechanicsTsFile(args))
