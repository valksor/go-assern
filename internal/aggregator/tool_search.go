package aggregator

import (
	"slices"
	"sort"
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
		sortEntriesByName(all)

		return capEntries(all, limit)
	}

	scored := make([]scoredEntry, 0, len(all))

	for _, e := range all {
		if s := scoreEntry(e, terms); s > 0 {
			scored = append(scored, scoredEntry{entry: e, score: s})
		}
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}

		return scored[i].entry.PrefixedName < scored[j].entry.PrefixedName
	})

	result := make([]*ToolEntry, len(scored))
	for i, s := range scored {
		result[i] = s.entry
	}

	return capEntries(result, limit)
}

// scoreEntry computes the relevance of a single tool for the given query terms.
// A score of zero means the tool does not match.
func scoreEntry(e *ToolEntry, terms []string) int {
	name := strings.ToLower(e.PrefixedName + " " + e.Tool.Name)
	desc := strings.ToLower(e.Tool.Description)
	server := strings.ToLower(e.ServerName)
	nameWords := tokenize(name)

	score := 0
	matchedTerms := 0

	for _, term := range terms {
		termScore := 0

		switch {
		case slices.Contains(nameWords, term):
			termScore += scoreNameWord
		case strings.Contains(name, term):
			termScore += scoreNameSub
		}

		if strings.Contains(desc, term) {
			termScore += scoreDescSub
		}

		if strings.Contains(server, term) {
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
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].PrefixedName < entries[j].PrefixedName
	})
}

// capEntries truncates entries to limit when limit is positive.
func capEntries(entries []*ToolEntry, limit int) []*ToolEntry {
	if limit > 0 && len(entries) > limit {
		return entries[:limit]
	}

	return entries
}
