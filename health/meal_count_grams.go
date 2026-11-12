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
	if m := itemTimesRe.FindStringSubmatch(ch); len(m) > 1 {
		return parseFloatLoose(m[1])
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
	if !itemTimesRe.MatchString(rest) {
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
