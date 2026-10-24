package core

// One rank of a spell as the ranked ladders describe it to core.
type RankSibling struct {
	ID         int32
	LearnLevel int32 // The level the rank is taught at; 0 where the game files state none.
}

// The ranks of one spell, keyed by each of their ids. Filled by the ranked ladders as the class
// packages initialize, so every write is done before the first request; read when an APL names a
// rank the character has not been taught.
var rankSiblings = map[int32][]RankSibling{}

// Records that these ranks are the ranks of one spell.
func RegisterRankSiblings(ranks []RankSibling) {
	for _, rank := range ranks {
		rankSiblings[rank.ID] = ranks
	}
}

// The ranks of the spell this id is a rank of, or nil where no ranked ladder carries it.
func RankSiblings(id int32) []RankSibling {
	return rankSiblings[id]
}

// Whether this id is a rank taught above CharacterLevel. An id no ranked ladder carries is not.
func RankTaughtAboveLevel(id int32) bool {
	for _, rank := range rankSiblings[id] {
		if rank.ID == id {
			return rank.LearnLevel > CharacterLevel
		}
	}
	return false
}
