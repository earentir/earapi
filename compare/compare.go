// Package compare computes set relationships between watchlists.
package compare

import (
	"sort"
	"strconv"
	"strings"

	"earapi/imdb"
)

// Entry is a title plus who has it on their list.
type Entry struct {
	Title  imdb.Title `json:"title"`
	Owners []string   `json:"owners"`
}

// Result is the full comparison.
//
// Partial is deliberately kept separate from Common and Unique: with three or
// more lists, "on 2 of 3" is the interesting middle ground that a plain
// common/unique split would silently discard.
type Result struct {
	Owners  []string           `json:"owners"`
	Common  []imdb.Title       `json:"common"`  // on every list
	Unique  map[string][]Entry `json:"unique"`  // on exactly one list, by owner
	Partial []Entry            `json:"partial"` // on some, not all
	All     []Entry            `json:"all"`     // union, with owner attribution
	Stats   Stats              `json:"stats"`
}

// Stats summarises the comparison.
type Stats struct {
	ListCount   int            `json:"list_count"`
	PerOwner    map[string]int `json:"per_owner"`
	UniqueCount map[string]int `json:"unique_count"`
	CommonCount int            `json:"common_count"`
	PartialCnt  int            `json:"partial_count"`
	UnionCount  int            `json:"union_count"`
	Pairwise    []Overlap      `json:"pairwise"`
}

// Overlap is the shared-title count between two owners.
type Overlap struct {
	A       string  `json:"a"`
	B       string  `json:"b"`
	Shared  int     `json:"shared"`
	Jaccard float64 `json:"jaccard"`
}

// Input is one named list going into a comparison.
type Input struct {
	Owner  string
	Titles []imdb.Title
}

// Compare computes common / unique / partial across two or more lists.
//
// Owner names are de-duplicated (two lists labelled "Sam" would otherwise make
// "on every list" impossible to reason about) and each list is de-duplicated
// internally so a repeated title can't inflate the overlap counts.
func Compare(inputs []Input) *Result {
	// Every slice starts non-nil: these are serialised straight to JSON, and a
	// nil slice becomes null, which the UI would have to guard on every access.
	res := &Result{
		Owners:  []string{},
		Common:  []imdb.Title{},
		Unique:  map[string][]Entry{},
		Partial: []Entry{},
		All:     []Entry{},
		Stats: Stats{
			PerOwner:    map[string]int{},
			UniqueCount: map[string]int{},
			Pairwise:    []Overlap{},
		},
	}
	if len(inputs) == 0 {
		return res
	}

	owners := make([]string, 0, len(inputs))
	sets := make(map[string]map[string]bool, len(inputs))
	// Keep the richest metadata we saw for each title id: lists fetched from
	// different sources (CSV vs API) carry different amounts of detail.
	meta := map[string]imdb.Title{}

	usedNames := map[string]int{}
	for _, in := range inputs {
		name := strings.TrimSpace(in.Owner)
		if name == "" {
			name = "List"
		}
		if n, clash := usedNames[name]; clash {
			usedNames[name] = n + 1
			name = name + " (" + strconv.Itoa(n+1) + ")"
		} else {
			usedNames[name] = 1
		}

		owners = append(owners, name)
		set := make(map[string]bool, len(in.Titles))
		for _, t := range in.Titles {
			if t.IMDbID == "" {
				continue
			}
			set[t.IMDbID] = true
			if richer(t, meta[t.IMDbID]) {
				meta[t.IMDbID] = t
			}
		}
		sets[name] = set
		res.Stats.PerOwner[name] = len(set)
	}
	res.Owners = owners
	res.Stats.ListCount = len(owners)

	// Attribute every title to its owners, preserving owner input order.
	ownersOf := map[string][]string{}
	for _, o := range owners {
		for id := range sets[o] {
			ownersOf[id] = append(ownersOf[id], o)
		}
	}
	order := map[string]int{}
	for i, o := range owners {
		order[o] = i
	}

	ids := make([]string, 0, len(ownersOf))
	for id := range ownersOf {
		ids = append(ids, id)
	}
	// Deterministic output: by title, then year, then id.
	sort.Slice(ids, func(i, j int) bool { return lessTitle(meta[ids[i]], meta[ids[j]]) })

	for _, id := range ids {
		os := ownersOf[id]
		sort.Slice(os, func(i, j int) bool { return order[os[i]] < order[os[j]] })
		e := Entry{Title: meta[id], Owners: os}
		res.All = append(res.All, e)

		switch {
		case len(os) == len(owners) && len(owners) > 0:
			res.Common = append(res.Common, e.Title)
		case len(os) == 1:
			res.Unique[os[0]] = append(res.Unique[os[0]], e)
		default:
			res.Partial = append(res.Partial, e)
		}
	}

	for _, o := range owners {
		if _, ok := res.Unique[o]; !ok {
			res.Unique[o] = []Entry{} // present-but-empty reads better in JSON
		}
		res.Stats.UniqueCount[o] = len(res.Unique[o])
	}
	res.Stats.CommonCount = len(res.Common)
	res.Stats.PartialCnt = len(res.Partial)
	res.Stats.UnionCount = len(res.All)

	for i := range owners {
		for j := i + 1; j < len(owners); j++ {
			a, b := owners[i], owners[j]
			shared := 0
			for id := range sets[a] {
				if sets[b][id] {
					shared++
				}
			}
			union := len(sets[a]) + len(sets[b]) - shared
			var jac float64
			if union > 0 {
				jac = float64(shared) / float64(union)
			}
			res.Stats.Pairwise = append(res.Stats.Pairwise, Overlap{
				A: a, B: b, Shared: shared, Jaccard: jac,
			})
		}
	}
	return res
}

// richer reports whether a carries more usable metadata than b, so a CSV-only
// record never overwrites a fully hydrated one.
func richer(a, b imdb.Title) bool {
	if b.IMDbID == "" {
		return true
	}
	return score(a) > score(b)
}

func score(t imdb.Title) int {
	n := 0
	for _, ok := range []bool{
		t.Title != "", t.Year > 0, t.Type != "", t.PosterURL != "",
		t.Plot != "", len(t.Genres) > 0, t.Rating > 0, t.RuntimeSec > 0,
	} {
		if ok {
			n++
		}
	}
	return n
}

func lessTitle(a, b imdb.Title) bool {
	at, bt := strings.ToLower(a.Title), strings.ToLower(b.Title)
	if at != bt {
		return at < bt
	}
	if a.Year != b.Year {
		return a.Year < b.Year
	}
	return a.IMDbID < b.IMDbID
}

// TitlesOf flattens entries back to plain titles, for export and Jellyfin sync.
func TitlesOf(entries []Entry) []imdb.Title {
	out := make([]imdb.Title, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Title)
	}
	return out
}

// View returns one named slice of the result, ready for export or sync.
// name is "common", "partial", "all", or "unique:<owner>".
func (r *Result) View(name string) ([]imdb.Title, bool) {
	switch {
	case name == "common":
		return r.Common, true
	case name == "partial":
		return TitlesOf(r.Partial), true
	case name == "all" || name == "":
		return TitlesOf(r.All), true
	case strings.HasPrefix(name, "unique:"):
		owner := strings.TrimPrefix(name, "unique:")
		entries, ok := r.Unique[owner]
		if !ok {
			return nil, false
		}
		return TitlesOf(entries), true
	}
	return nil, false
}
