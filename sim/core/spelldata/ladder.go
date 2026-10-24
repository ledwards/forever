package spelldata

import (
	"fmt"
	"slices"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
)

// A spell's ranks in rank order, whether the game files state them as one spell per rank or as one
// spell with a curve. Rank 0 is untaken, so the readers below answer 0 there instead of panicking.
type Ladder struct {
	ranks []*Spell

	// Whether the ranks are spells taught at levels (Ranked), as against the points of one talent
	// (Talent). Only taught ranks answer for core.CharacterLevel.
	ranked bool
}

// A ladder of one spell per rank, lowest first. Every id has to be in the store: a rank named by typo
// has to fail loudly rather than register nothing. The ids are told to core as ranks of one spell,
// so an APL that names a rank the spellbook lacks can fall back to one it has.
func Ranked(ids ...int32) Ladder {
	ranks := make([]*Spell, len(ids))
	siblings := make([]core.RankSibling, len(ids))
	for i, id := range ids {
		ranks[i] = MustFind(id)
		siblings[i] = core.RankSibling{ID: id, LearnLevel: learnLevel(ranks[i])}
	}
	core.RegisterRankSiblings(siblings)
	return Ladder{ranks: ranks, ranked: true}
}

// The level the row is taught at, or 0 for a row with no SpellLevels data.
func learnLevel(s *Spell) int32 {
	if s.SpellLevel == 60 && s.BaseLevel == 60 && s.MaxLevel == 0 {
		return 0
	}
	return int32(s.SpellLevel)
}

// Whether the row is taught at or below core.CharacterLevel. A row with no SpellLevels data carries
// SpellLevel 60, BaseLevel 60 and MaxLevel 0, and is taught at every level; a row taught at 60 for
// real carries a MaxLevel.
func learnable(s *Spell) bool {
	return learnLevel(s) <= core.CharacterLevel
}

// The row as the ladder answers it at core.CharacterLevel: the store's own pointer when the rank is
// taught by then, otherwise a copy marked Unlearned, so no reader writes through to the store.
func (l Ladder) atLevel(s *Spell) *Spell {
	if !l.ranked || s == Nil || learnable(s) {
		return s
	}
	unlearned := *s
	unlearned.Unlearned = true
	return &unlearned
}

// The highest rank taught at or below core.CharacterLevel, or the lowest rank, marked Unlearned, when
// none is: the class code that registers its pick unconditionally still gets a row with a school
// and a cost, and the spellbook carries a spell that cannot be cast yet.
func (l Ladder) AtMost(n int32) *Spell {
	if !l.ranked {
		return l.Rank(n)
	}
	if n > l.Len() {
		n = l.Len()
	}
	for i := n; i >= 1; i-- {
		if learnable(l.ranks[i-1]) {
			return l.ranks[i-1]
		}
	}
	if len(l.ranks) == 0 {
		return Nil
	}
	return l.atLevel(l.ranks[0])
}

// A ladder of a trait-tree talent, which is one spell whose per-rank numbers live in a curve. Each
// rank is a copy of the spell with the curve's value on the effects the curve covers; an effect the
// curve has no row for keeps the spell's own base value.
func Talent(spellID int32, maxRanks int32) Ladder {
	base := MustFind(spellID)
	curve := curves[spellID]

	// A curve row states one value per rank. A row of another length means the curve and the talent
	// disagree about how many ranks there are, and the ranks past the shorter of the two would
	// silently keep the base value.
	for i, row := range curve {
		if int32(len(row)) != maxRanks {
			panic(fmt.Sprintf("spelldata: spell %d curve row %d has %d ranks, want %d",
				spellID, i, len(row), maxRanks))
		}
	}

	ranks := make([]*Spell, 0, maxRanks)
	for n := int32(1); n <= maxRanks; n++ {
		// Only the effects are copied, since only their base points differ by rank: the powers, the
		// labels and the tooltip references stay the store's own, and no rank writes through them.
		rank := *base
		rank.Effects = slices.Clone(base.Effects)
		for i := range rank.Effects {
			if i < len(curve) {
				rank.Effects[i].BasePoints = curve[i][n-1]
			}
		}
		ranks = append(ranks, &rank)
	}
	return Ladder{ranks: ranks}
}

// The spell at a rank, marked Unlearned where it is taught above core.CharacterLevel. Rank 0 is
// untaken and answers Nil, as does a rank the ladder does not have. The rank is exact so a triggered
// ladder read beside its parent pairs rank with rank; AtMost is the reader for a pinned pick.
func (l Ladder) Rank(n int32) *Spell {
	if n <= 0 || n > l.Len() {
		return Nil
	}
	return l.atLevel(l.ranks[n-1])
}

// The highest rank taught at or below core.CharacterLevel; see AtMost for what a ladder taught
// entirely above it answers.
func (l Ladder) Highest() *Spell {
	return l.AtMost(l.Len())
}

// The rank with this spell id. Panics on an id the ladder does not carry, the way MustFind does on a
// spell the store does not carry.
func (l Ladder) ByID(id int32) *Spell {
	for _, s := range l.ranks {
		if s.ID == id {
			return s
		}
	}
	panic(fmt.Sprintf("spelldata: spell %d is not one of the %d ranks of this ladder", id, len(l.ranks)))
}

func (l Ladder) Len() int32 {
	return int32(len(l.ranks))
}

// Whether the spell was registered from one of the ladder's ranks.
func (l Ladder) Contains(spell *core.Spell) bool {
	return slices.ContainsFunc(l.ranks, func(s *Spell) bool { return s.ID == spell.SpellID })
}

// Every rank in order, those taught above core.CharacterLevel marked Unlearned, so a class that
// registers each rank and indexes by rank number sees no holes.
func (l Ladder) Each(fn func(rank int32, s *Spell)) {
	for i, s := range l.ranks {
		fn(int32(i+1), l.atLevel(s))
	}
}

// The rank's only effect, in the game files' units. Rank 0 is untaken and answers 0.
func (l Ladder) ValueAt(rank int32) float64 {
	return l.only().ValueAt(rank)
}

// The game files state a percentage as an integer: 16, not 0.16.
func (l Ladder) FractionAt(rank int32) float64 {
	return l.only().FractionAt(rank)
}

// The sign comes from the data: Improved Righteous Fury states -2/-4/-6, so rank 3 gives 0.94.
func (l Ladder) MultiplierAt(rank int32) float64 {
	return l.only().MultiplierAt(rank)
}

// The game files state rage on a 0-1000 bar.
func (l Ladder) TenthsAt(rank int32) float64 {
	return l.only().TenthsAt(rank)
}

// The effect at a position, counted from 1, for the ranks that carry more than one.
func (l Ladder) EffectAt(index int32) LadderEffect {
	return LadderEffect{ladder: l, pick: func(s *Spell, _ int32) *Effect {
		e := s.EffectN(int(index))
		if e == NilEffect {
			panic(fmt.Sprintf("spell %d has no effect at position %d, in %d effects",
				s.ID, index, len(s.Effects)))
		}
		return e
	}}
}

// The effect with this aura and misc value, which panics when the rank has two of them.
func (l Ladder) Effect(aura dbcenums.EffectAuraType, misc int32) LadderEffect {
	return LadderEffect{ladder: l, pick: func(s *Spell, _ int32) *Effect { return s.Effect(aura, misc) }}
}

// Nothing named means nothing to choose between - reading the first of several silently is the bug
// this shape exists to prevent.
func (l Ladder) only() LadderEffect {
	return LadderEffect{ladder: l, pick: func(s *Spell, rank int32) *Effect {
		if len(s.Effects) != 1 {
			panic(fmt.Sprintf("spell %d rank %d has %d effects - name the one you mean with Effect(aura, misc)",
				s.ID, rank, len(s.Effects)))
		}
		return &s.Effects[0]
	}}
}

// One named effect across the ladder's ranks.
type LadderEffect struct {
	ladder Ladder
	pick   func(s *Spell, rank int32) *Effect
}

func (e LadderEffect) ValueAt(rank int32) float64 {
	return e.at(rank).BasePoints
}

func (e LadderEffect) FractionAt(rank int32) float64 {
	return e.at(rank).Percent()
}

func (e LadderEffect) MultiplierAt(rank int32) float64 {
	return 1 + e.FractionAt(rank)
}

func (e LadderEffect) TenthsAt(rank int32) float64 {
	return e.at(rank).Tenths()
}

// The effect at a rank. Rank 0 is untaken and answers NilEffect, which reads as zero.
func (e LadderEffect) at(rank int32) *Effect {
	if rank <= 0 {
		return NilEffect
	}
	if rank > e.ladder.Len() {
		panic(fmt.Sprintf("rank %d in a ladder of %d ranks", rank, e.ladder.Len()))
	}
	return e.pick(e.ladder.Rank(rank), rank)
}
