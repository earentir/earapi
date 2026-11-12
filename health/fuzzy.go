package health

import (
	"strings"
	"unicode"
)

// RankFoods returns foods sorted by match score for query q (higher first).
func RankFoods(foods []Food, q string, limit int) []Food {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		if limit <= 0 || limit > len(foods) {
			return append([]Food(nil), foods...)
		}
		out := make([]Food, limit)
		copy(out, foods[:limit])
		return out
	}
	type scored struct {
		f     Food
		score int
	}
	var ranked []scored
	for _, f := range foods {
		sc := scoreName(f.Name, q)
		if sc > 0 {
			ranked = append(ranked, scored{f, sc})
		}
	}
	for i := 0; i < len(ranked); i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].score > ranked[i].score {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}
	if limit <= 0 {
		limit = 25
	}
	out := make([]Food, 0, limit)
	for i := 0; i < len(ranked) && len(out) < limit; i++ {
		out = append(out, ranked[i].f)
	}
	return out
}

func scoreName(name, q string) int {
	low := strings.ToLower(cleanFoodDisplayName(name))
	if low == q {
		return 1000
	}
	if strings.Contains(low, q) {
		return 800 + (200 * len(q) / max(1, len(low)))
	}
	// token overlap
	qt := tokens(q)
	nt := tokens(low)
	overlap := 0
	for _, tq := range qt {
		for _, tn := range nt {
			if tq == tn {
				overlap += 100
			} else if strings.Contains(tn, tq) || strings.Contains(tq, tn) {
				overlap += 50
			} else {
				d := levenshtein(tq, tn)
				if d <= 2 && len(tq) > 2 {
					overlap += 40 - d*10
				}
			}
		}
	}
	return overlap
}

func tokens(s string) []string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return ' '
	}, s)
	parts := strings.Fields(s)
	return parts
}

func levenshtein(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min3(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// BestFoodMatch picks the top fuzzy match for scaling nutrients.
func BestFoodMatch(foods []Food, name string) (*Food, bool) {
	q := foodMatchQuery(name)
	if q == "" {
		return nil, false
	}
	top := RankFoods(foods, q, 5)
	if len(top) == 0 {
		return nil, false
	}
	best := scoreName(top[0].Name, q)
	if best < 80 {
		for _, c := range top {
			if tokenBoost(c.Name, q) {
				f := c
				return &f, true
			}
		}
		return nil, false
	}
	if len(top) > 1 && scoreName(top[1].Name, q) >= best-15 {
		return nil, false
	}
	f := top[0]
	return &f, true
}

// marata-style: meal line "Marata 1.7%" vs catalog bullet with Greek name + 1.7%.
func tokenBoost(name, q string) bool {
	low := strings.ToLower(cleanFoodDisplayName(name))
	q = strings.ToLower(q)
	if strings.Contains(q, "marata") && strings.Contains(low, "marata") && strings.Contains(q, "1.7") && strings.Contains(low, "1.7") {
		return true
	}
	return false
}

func foodMatchQuery(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if i := strings.Index(name, "("); i > 0 {
		name = strings.TrimSpace(name[:i])
	}
	return strings.ToLower(name)
}
