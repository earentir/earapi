package health

import (
	"regexp"
	"strings"
)

const defaultEggGrams = 55.0

var itemEggsTimesRe = regexp.MustCompile(`(?i)\beggs?\s*×\s*(\d+(?:\.\d+)?)`)

func countFromMealChunk(ch string) float64 {
	if m := itemEggsTimesRe.FindStringSubmatch(ch); len(m) > 1 {
		return parseFloatLoose(m[1])
	}
	if m := itemEggCountRe.FindStringSubmatch(ch); len(m) > 1 {
		return parseFloatLoose(m[1])
	}
	// Ignore × inside "(…)" notes (e.g. yield ÷5×3).
	if c := countTimesOutsideParens(ch); c > 0 {
		return c
	}
	return 0
}

func countTimesOutsideParens(ch string) float64 {
	depth := 0
	rs := []rune(ch)
	for i, r := range rs {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case '×':
			if depth > 0 {
				continue
			}
			rest := strings.TrimSpace(string(rs[i+1:]))
			if m := regexp.MustCompile(`^(\d+(?:\.\d+)?)`).FindStringSubmatch(rest); len(m) > 1 {
				return parseFloatLoose(m[1])
			}
		}
	}
	return 0
}

// normalizeMealItemCountGrams sets Count and total Grams from ×N, N eggs, and per-unit "N g ×M" markdown.
func normalizeMealItemCountGrams(it *MealItem) {
	if it == nil {
		return
	}
	ch := strings.TrimSpace(it.Note)
	if ch == "" {
		ch = strings.TrimSpace(it.Name)
	}
	if it.Count <= 0 {
		if c := countFromMealChunk(ch); c > 0 {
			it.Count = c
		}
	}
	if m := itemEggCountRe.FindStringSubmatch(ch); len(m) > 1 {
		eggN := parseFloatLoose(m[1])
		if eggN > 0 {
			it.Count = eggN
			if it.Grams == 0 {
				it.Grams = eggN * defaultEggGrams
			}
		}
	}
	if m := itemEggsTimesRe.FindStringSubmatch(ch); len(m) > 1 {
		n := parseFloatLoose(m[1])
		if n > 0 {
			it.Count = n
			if it.Grams == 0 {
				if g := gramsFromMealChunk(ch); g > 0 {
					it.Grams = g
				} else if g := itemCountGuessGrams(n, ch); g > 0 {
					it.Grams = g
				} else {
					it.Grams = n * defaultEggGrams
				}
			}
		}
	}
	if it.Count > 1 && !edibleGramsBeforeCount(ch) {
		if unitG, perUnit := firstGramsFollowedByTimes(ch); perUnit {
			if parensTotal := firstParenGramsNotBone(ch); parensTotal > unitG {
				it.Grams = parensTotal
			} else if it.Grams > 0 && it.Grams <= unitG+0.01 {
				it.Grams = unitG * it.Count
			}
		} else if it.Grams == 0 {
			if g := itemCountGuessGrams(it.Count, ch); g > 0 {
				it.Grams = g
			}
		}
	}
	clearMealItemCountUnlessMultiplier(it, ch)
}

// clearMealItemCountUnlessMultiplier drops count when markdown has no real unit multiplier (×N) or egg count.
func clearMealItemCountUnlessMultiplier(it *MealItem, ch string) {
	if it == nil || it.Count <= 0 {
		return
	}
	if itemEggCountRe.FindStringSubmatch(ch) != nil {
		return
	}
	if itemEggsTimesRe.FindStringSubmatch(ch) != nil {
		return
	}
	if countTimesOutsideParens(ch) > 0 {
		return
	}
	it.Count = 0
}

// firstGramsFollowedByTimes is true for "12 g ×3" style per-unit weights, not "3 eggs 165 g".
func firstGramsFollowedByTimes(ch string) (unitG float64, ok bool) {
	loc := itemGramsRe.FindStringIndex(ch)
	if loc == nil {
		return 0, false
	}
	m := itemGramsRe.FindStringSubmatch(ch)
	if len(m) < 2 {
		return 0, false
	}
	unitG = parseFloatLoose(m[1])
	rest := strings.TrimSpace(ch[loc[1]:])
	if !regexp.MustCompile(`(?i)^(?:edible\s*)?×\s*\d`).MatchString(rest) {
		return 0, false
	}
	return unitG, true
}

func firstParenGramsNotBone(ch string) float64 {
	for _, m := range itemParenGRe.FindAllStringSubmatch(ch, -1) {
		if len(m) < 2 {
			continue
		}
		if parenGramsIsBoneNote(ch, m[1]) {
			continue
		}
		return parseFloatLoose(m[1])
	}
	return 0
}

func normalizeMealItems(items []MealItem) {
	for i := range items {
		normalizeMealItemCountGrams(&items[i])
		if len(items[i].Parts) > 0 {
			normalizeMealItems(items[i].Parts)
		}
	}
}
