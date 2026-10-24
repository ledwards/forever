# Sources of the per-level inputs

- `PlayerExpectedStat.csv` and `ExpectedStat.csv`: the game files' DB2 tables for build
  1.60.1.70450, exported from wago.tools on 2026-10-23
  (`https://wago.tools/db2/PlayerExpectedStat/csv?build=1.60.1.70450` and
  `https://wago.tools/db2/ExpectedStat/csv?build=1.60.1.70450`), filtered to levels 1 to 63 and
  ContentSetID 0, which is the only content set either table carries. `make db` produces the same
  rows in `tools/database/wowsims.db`; the generator reads the database when it is present and these
  files otherwise, and says which in the generated header.
- `class_levelstats.tsv`: each class's base health and primary attributes at every level 1 to 60,
  before any race offset, from the CMaNGOS mangos-classic `player_levelstats` and
  `player_classlevelstats` tables (`sql/base/mangos.sql`, commit 7f98ba0f, fetched 2026-10-23).
  Rows are Human for the classes Humans play, Night Elf less the Night Elf offset for hunter and
  druid, and Orc less the Orc offset for shaman; the offsets match the ones in `base_stats.go` for
  strength, agility, stamina and intellect, and drift with level for spirit, so the spirit column
  of hunter, druid and shaman carries up to about five points of error by 60. The source repeats
  the warrior level 18 row at 19; that row is corrected here to the other races' value. The `mana`
  column is unused: base mana comes from PlayerExpectedStat.
  At level 60 these rows match the hand-measured values in `base_stats.go` exactly for strength,
  agility, stamina and intellect, and within ten points for health (priest, mage) and six for
  spirit; `base_stats_levels.go` scales each class's column so its level 60 row equals the measured
  one.
