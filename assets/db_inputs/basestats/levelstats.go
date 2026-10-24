// Package basestats carries the per-level inputs the sim reads at init. The gametables beside it
// feed the generator instead, and are not embedded.
package basestats

import _ "embed"

// Each class's base health and primary attributes at every level 1 to 60, before any race offset;
// SOURCES.md beside it says where the rows come from.
//
//go:embed class_levelstats.tsv
var ClassLevelStatsTSV string
