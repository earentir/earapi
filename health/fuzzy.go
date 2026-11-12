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
	pool := foods
	if qt := matchQueryTokens(q); len(qt) >= 2 {
		var strict []Food
		for _, f := range foods {
			if catalogMatchesMealLine(f.Name, q) {
				strict = append(strict, f)
			}
		}
		if len(strict) == 0 {
			return nil
		}
		pool = strict
	}
	minScore := 80
	if len(tokens(q)) < 2 {
		minScore = 1
	}
	var ranked []scored
	for _, f := range pool {
		sc := scoreName(f.Name, q)
		if sc >= minScore {
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
	full := strings.ToLower(cleanFoodDisplayName(name))
	// token overlap
	qt := matchQueryTokens(q)
	nt := tokens(low)
	overlap := 0
	for _, tq := range qt {
		for _, tn := range nt {
			if tokensEqualForFood(tq, tn) {
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
	if len(qt) > 0 {
		allIn := true
		for _, tq := range qt {
			if !tokenInFoodName(tq, full) {
				allIn = false
				break
			}
		}
		if allIn {
			boost := 80 + len(qt)*40
			if overlap < boost {
				overlap = boost
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

// foodPrepPhrases stripped before tokenizing (longest first).
var foodPrepPhrases = []string{
	"sunny side up", "sunny side", "sunny-side",
	"air fryer", "air fried", "air-fried", "airfried",
	"deep fried", "deep-fried", "pan fried", "pan-fried", "stir fried", "stir-fried",
	"no salt", "no-salt", "low salt", "low-salt", "low sodium", "low-sodium",
	"bone in", "bone-in", "skin on", "skin-on",
	"cooked no salt", "cooked, no salt",
	"ready meal", "ready-meal",
}

// Cooking / handling words in meal logs, not food identities (ignored when matching).
var foodPrepStopwords = map[string]bool{
	"boiled": true, "roasted": true, "fried": true, "grilled": true, "baked": true,
	"steamed": true, "smoked": true, "braised": true, "poached": true, "scrambled": true,
	"sauteed": true, "sautéed": true, "saute": true, "microwaved": true, "charred": true,
	"broiled": true, "stewed": true, "blanched": true, "seared": true, "toasted": true,
	"air": true, "deep": true, "pan": true, "stir": true,
	"raw": true, "cooked": true, "uncooked": true, "frozen": true, "dried": true,
	"canned": true, "fresh": true, "unsalted": true, "salted": true, "seasoned": true,
	"pickled": true, "marinated": true, "breaded": true, "caramelized": true,
	"shredded": true, "sliced": true, "diced": true, "chopped": true, "minced": true,
	"crispy": true, "whole": true, "boneless": true, "bone": true, "bones": true,
	"skinless": true, "skin": true, "edible": true, "homemade": true, "ready": true,
	"sunny": true, "side": true, "up": true, "omelet": true, "omelette": true,
	"style": true, "minimal": true, "leftover": true,
	"large": true, "small": true, "medium": true,
}

func stripPrepFromMealLine(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", " ")
	for _, p := range foodPrepPhrases {
		p = strings.ReplaceAll(p, "-", " ")
		s = strings.ReplaceAll(s, p, " ")
	}
	return strings.Join(strings.Fields(s), " ")
}

func isFoodPrepToken(t string) bool {
	return foodPrepStopwords[t]
}

var foodPluralSingular = map[string]string{
	"potatoes": "potato", "eggs": "egg", "tomatoes": "tomato", "grapes": "grape",
	"leaves": "leaf", "beans": "bean", "onions": "onion", "peppers": "pepper",
}

func matchQueryTokens(q string) []string {
	qt := tokens(stripPrepFromMealLine(q))
	var meaningful []string
	for _, t := range qt {
		if len(t) > 0 && t[0] >= '0' && t[0] <= '9' {
			continue
		}
		meaningful = append(meaningful, t)
	}
	if len(meaningful) == 0 {
		return nil
	}
	out := make([]string, 0, len(meaningful))
	for _, t := range meaningful {
		if isFoodPrepToken(t) {
			continue
		}
		out = append(out, t)
	}
	return out
}

func foldFoodToken(t string) string {
	t = strings.ToLower(t)
	if sing, ok := foodPluralSingular[t]; ok {
		return sing
	}
	if strings.HasSuffix(t, "ies") && len(t) > 4 {
		return t[:len(t)-3] + "y"
	}
	if strings.HasSuffix(t, "es") && len(t) > 3 {
		return t[:len(t)-2]
	}
	if strings.HasSuffix(t, "s") && !strings.HasSuffix(t, "ss") && len(t) > 3 {
		return t[:len(t)-1]
	}
	return t
}

func tokensEqualForFood(a, b string) bool {
	if a == b {
		return true
	}
	return foldFoodToken(a) == foldFoodToken(b)
}

func tokenInFoodName(token, full string) bool {
	token = strings.ToLower(token)
	if token == "" {
		return true
	}
	if strings.Contains(full, token) {
		return true
	}
	if sing, ok := foodPluralSingular[token]; ok && strings.Contains(full, sing) {
		return true
	}
	if folded := foldFoodToken(token); folded != token && strings.Contains(full, folded) {
		return true
	}
	return false
}

func mealNounTokens(mealLine string) []string {
	return matchQueryTokens(mealLine)
}

// catalogMatchesMealLine is true when catalog food fits the meal shorthand (not e.g. potato (boiled) for "boiled egg").
func catalogMatchesMealLine(catalogName, mealLine string) bool {
	q := strings.ToLower(strings.TrimSpace(mealLine))
	if q == "" {
		return false
	}
	full := strings.ToLower(cleanFoodDisplayName(catalogName))
	for _, t := range matchQueryTokens(q) {
		if !tokenInFoodName(t, full) {
			return false
		}
	}
	for _, n := range mealNounTokens(mealLine) {
		if !tokenInFoodName(n, full) {
			return false
		}
	}
	return true
}

func allQueryTokensInName(name, q string) bool {
	return catalogMatchesMealLine(name, q)
}

func filterFoodsForMealLine(foods []Food, mealLine string) []Food {
	var out []Food
	for _, f := range foods {
		if catalogMatchesMealLine(f.Name, mealLine) {
			out = append(out, f)
		}
	}
	return out
}

func mealPrepHints(mealLine string) []string {
	var hints []string
	for _, t := range tokens(strings.ToLower(strings.TrimSpace(mealLine))) {
		if isFoodPrepToken(t) {
			hints = append(hints, t)
		}
	}
	return hints
}

// Prep in the meal line boosts catalog rows that mention the same method (tie-break only).
func scoreNameForMealLine(catalogName, mealLine string) int {
	q := foodMatchQuery(mealLine)
	sc := scoreName(catalogName, q)
	full := strings.ToLower(cleanFoodDisplayName(catalogName))
	for _, h := range mealPrepHints(mealLine) {
		if strings.Contains(full, h) {
			sc += 55
		}
	}
	return sc
}

// BestFoodMatch picks the top fuzzy match for scaling nutrients.
func BestFoodMatch(foods []Food, name string) (*Food, bool) {
	q := foodMatchQuery(name)
	if q == "" {
		return nil, false
	}
	pool := filterFoodsForMealLine(foods, name)
	if len(pool) == 0 {
		return nil, false
	}
	type scored struct {
		f  Food
		sc int
	}
	var ranked []scored
	for _, f := range pool {
		sc := scoreNameForMealLine(f.Name, name)
		if sc >= 80 {
			ranked = append(ranked, scored{f, sc})
		}
	}
	if len(ranked) == 0 {
		for _, c := range pool {
			if tokenBoost(c.Name, q) {
				f := c
				return &f, true
			}
		}
		return nil, false
	}
	for i := 0; i < len(ranked); i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].sc > ranked[i].sc {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}
	best := ranked[0].sc
	if len(ranked) > 1 && ranked[1].sc >= best-15 {
		qExact := strings.ToLower(strings.TrimSpace(name))
		var tied []scored
		for _, r := range ranked {
			if r.sc >= best-15 {
				tied = append(tied, r)
			}
		}
		var exact []scored
		for _, r := range tied {
			if strings.ToLower(cleanFoodDisplayName(r.f.Name)) == qExact {
				exact = append(exact, r)
			}
		}
		pick := func(cands []scored) (*Food, bool) {
			if len(cands) == 0 {
				return nil, false
			}
			bestLen := len(cleanFoodDisplayName(cands[0].f.Name))
			idx := 0
			for i := 1; i < len(cands); i++ {
				n := len(cleanFoodDisplayName(cands[i].f.Name))
				if n < bestLen || (n == bestLen && cands[i].sc > cands[idx].sc) {
					bestLen = n
					idx = i
				}
			}
			f := cands[idx].f
			return &f, true
		}
		if len(exact) == 1 {
			f := exact[0].f
			return &f, true
		}
		if len(exact) > 1 {
			return pick(exact)
		}
		var atBest []scored
		for _, r := range tied {
			if r.sc == best {
				atBest = append(atBest, r)
			}
		}
		if len(atBest) > 1 {
			return nil, false
		}
		return pick(tied)
	}
	f := ranked[0].f
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
