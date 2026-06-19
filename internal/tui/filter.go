package tui

import (
	"sort"
	"strings"
)

// score rates how well query matches text. A substring hit scores far higher
// than a scattered fuzzy (subsequence) hit; an empty query matches everything.
// The boolean reports whether there was any match at all.
//
// This is implemented in-house (a ~tiny subsequence matcher) rather than pulling
// a fuzzy-search dependency: the working set is small and decrypted in memory, so
// a straightforward scan is more than fast enough.
func score(query, text string) (int, bool) {
	if query == "" {
		return 0, true
	}
	q := strings.ToLower(query)
	t := strings.ToLower(text)

	if idx := strings.Index(t, q); idx >= 0 {
		// Substring: reward earlier matches and tighter (shorter) text.
		return 100_000 - idx*10 - (len(t) - len(q)), true
	}
	return fuzzyScore([]rune(q), t)
}

// fuzzyScore matches query as an ordered subsequence of text, rewarding runs of
// consecutive matched runes so "abc" prefers "abcdef" over "a_b_c".
func fuzzyScore(query []rune, text string) (int, bool) {
	qi, streak, total := 0, 0, 0
	for _, tr := range text {
		if qi < len(query) && tr == query[qi] {
			qi++
			streak++
			total += streak * 2
		} else {
			streak = 0
		}
	}
	if qi == len(query) {
		return total, true
	}
	return 0, false
}

// Search returns up to limit Items matching query, best match first. A limit of
// zero means no cap. It reuses the same ranking as the interactive TUI, so
// launcher integrations (e.g. `yoink search`) behave identically.
func Search(items []Item, query string, limit int) []Item {
	idxs := filter(items, query)
	if limit > 0 && len(idxs) > limit {
		idxs = idxs[:limit]
	}
	out := make([]Item, len(idxs))
	for i, idx := range idxs {
		out[i] = items[idx]
	}
	return out
}

// filter returns the indices of items whose Text matches query, ordered by
// score (best first). Ties keep the items' original order, which is recency.
func filter(items []Item, query string) []int {
	type ranked struct {
		idx, pos, score int
	}
	var matches []ranked
	for i, it := range items {
		if s, ok := score(query, it.Text); ok {
			matches = append(matches, ranked{idx: i, pos: i, score: s})
		}
	}
	sort.SliceStable(matches, func(a, b int) bool {
		if matches[a].score != matches[b].score {
			return matches[a].score > matches[b].score
		}
		return matches[a].pos < matches[b].pos
	})
	out := make([]int, len(matches))
	for i, m := range matches {
		out[i] = m.idx
	}
	return out
}
