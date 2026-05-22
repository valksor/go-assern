package aggregator

import (
	"cmp"
	"slices"
	"strings"
	"unicode"
)

// Relevance weights for the catalog search ranker. They are intentionally
// coarse: the goal is a useful ordering with no external dependencies, leaving
// room to swap in embeddings later.
const (
	scoreNameWord = 10 // query term matches a whole word in the tool name
	scoreNameSub  = 6  // query term is a substring of the tool name
	scoreDescSub  = 3  // query term appears in the description
	scoreServer   = 2  // query term appears in the server name
	scoreCoverage = 5  // per distinct query term matched (rewards breadth)
)

// scoredEntry pairs a tool entry with its computed relevance score.
type scoredEntry struct {
	entry *ToolEntry
	score int
}

// Search ranks catalog tools against a free-text query and returns up to limit
// matches, most relevant first. An empty query returns tools in name order so a
// client can browse the catalog. A limit <= 0 returns all matches.
func (r *ToolRegistry) Search(query string, limit int) []*ToolEntry {
	all := r.r.all()

	terms := tokenize(query)
	if len(terms) == 0 {
		// all() returns a shared immutable snapshot; clone before sorting.
		browse := slices.Clone(all)
		sortEntriesByName(browse)

		return capEntries(browse, limit)
	}

	if limit > 0 {
		return topKMatches(all, terms, limit)
	}

	return allMatches(all, terms)
}

// rankCmp orders scored entries best-first: higher score wins, ties break by
// ascending prefixed name. PrefixedName is unique, so this is a total order
// (no need for a stable sort).
func rankCmp(a, b scoredEntry) int {
	if c := cmp.Compare(b.score, a.score); c != 0 {
		return c
	}

	return cmp.Compare(a.entry.PrefixedName, b.entry.PrefixedName)
}

// allMatches scores every entry and returns all matches, best-first.
func allMatches(all []*ToolEntry, terms []string) []*ToolEntry {
	scored := make([]scoredEntry, 0, len(all))
	for _, e := range all {
		if s := scoreEntry(e, terms); s > 0 {
			scored = append(scored, scoredEntry{entry: e, score: s})
		}
	}

	slices.SortFunc(scored, rankCmp)

	return entriesOf(scored)
}

// topKMatches returns the best `limit` matches, best-first. It keeps a bounded
// min-heap of candidates (worst-ranked at the root) in a single preallocated
// slice, so the full match set is never sorted and no per-candidate allocation
// occurs — O(N log limit) instead of O(N log N) when limit is small.
func topKMatches(all []*ToolEntry, terms []string, limit int) []*ToolEntry {
	kept := make([]scoredEntry, 0, limit)
	for _, e := range all {
		s := scoreEntry(e, terms)
		if s == 0 {
			continue
		}

		cand := scoredEntry{entry: e, score: s}
		switch {
		case len(kept) < limit:
			kept = append(kept, cand)
			siftUp(kept, len(kept)-1)
		case worseThan(kept[0], cand):
			// The heap is full and this candidate outranks the worst kept one.
			kept[0] = cand
			siftDown(kept, 0)
		}
	}

	slices.SortFunc(kept, rankCmp)

	return entriesOf(kept)
}

// entriesOf extracts the tool entries from a ranked slice, in order.
func entriesOf(scored []scoredEntry) []*ToolEntry {
	result := make([]*ToolEntry, len(scored))
	for i, s := range scored {
		result[i] = s.entry
	}

	return result
}

// worseThan reports whether a ranks lower than b — i.e. a is the more evictable
// of the two. It defines the min-heap order used by siftUp/siftDown, so the
// heap root is always the lowest-ranked (most evictable) entry.
func worseThan(a, b scoredEntry) bool {
	return rankCmp(a, b) > 0
}

// siftUp restores the heap invariant after appending at index i.
func siftUp(h []scoredEntry, i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if !worseThan(h[i], h[parent]) {
			break
		}

		h[i], h[parent] = h[parent], h[i]
		i = parent
	}
}

// siftDown restores the heap invariant after replacing the root at index i.
func siftDown(h []scoredEntry, i int) {
	n := len(h)
	for {
		// worst tracks the most-evictable of node i and its children; that
		// element belongs at the root of this subtree (a min-heap by rank).
		worst := i
		if l := 2*i + 1; l < n && worseThan(h[l], h[worst]) {
			worst = l
		}

		if r := 2*i + 2; r < n && worseThan(h[r], h[worst]) {
			worst = r
		}

		if worst == i {
			break
		}

		h[i], h[worst] = h[worst], h[i]
		i = worst
	}
}

// scoreEntry computes the relevance of a single tool for the given query terms.
// A score of zero means the tool does not match. It reads the entry's
// precomputed lowercased fields, populated in Register.
func scoreEntry(e *ToolEntry, terms []string) int {
	score := 0
	matchedTerms := 0

	for _, term := range terms {
		termScore := 0

		switch {
		case slices.Contains(e.nameWords, term):
			termScore += scoreNameWord
		case strings.Contains(e.lowerName, term):
			termScore += scoreNameSub
		}

		if strings.Contains(e.lowerDesc, term) {
			termScore += scoreDescSub
		}

		if strings.Contains(e.lowerServer, term) {
			termScore += scoreServer
		}

		if termScore > 0 {
			matchedTerms++
			score += termScore
		}
	}

	if matchedTerms == 0 {
		return 0
	}

	return score + matchedTerms*scoreCoverage
}

// tokenize lowercases and splits text into alphanumeric terms.
func tokenize(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// sortEntriesByName sorts entries alphabetically by their prefixed name.
func sortEntriesByName(entries []*ToolEntry) {
	slices.SortFunc(entries, func(a, b *ToolEntry) int {
		return cmp.Compare(a.PrefixedName, b.PrefixedName)
	})
}

// capEntries truncates entries to limit when limit is positive.
func capEntries(entries []*ToolEntry, limit int) []*ToolEntry {
	if limit > 0 && len(entries) > limit {
		return entries[:limit]
	}

	return entries
}
