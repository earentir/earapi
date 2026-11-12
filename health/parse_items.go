package health

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	itemGramsRe       = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*g\b`)
	itemEggCountRe    = regexp.MustCompile(`(?i)(\d+)\s+eggs?\b`)
	itemParenGRe      = regexp.MustCompile(`(?i)\((\d+(?:\.\d+)?)\s*g`)
	itemTimesRe       = regexp.MustCompile(`×\s*(\d+(?:\.\d+)?)`)
	itemTotalCutRe    = regexp.MustCompile(`(?i)\s*(?:→|->)\s*\*\*`)
	itemSkipRawRe     = regexp.MustCompile(`(?i)^(?:-|–|\x{2014})\s*(\(no |\s*$)|^\s*$`)
	itemInlinePairRe  = regexp.MustCompile(`(?i)(?:\s*~?[\d,.]+\s*/\s*~?[\d,.]+)+\s*$`)
	itemOnlyTotalRe   = regexp.MustCompile(`(?i)^\s*(?:→|->)?\s*\*\*`)
	itemSaladPrefixRe  = regexp.MustCompile(`(?i)^salad:\s*`)
	itemSetLabelRe     = regexp.MustCompile(`^([^:()]{1,48}):\s*(.+)$`)
	itemSetBreakHintRe = regexp.MustCompile(`(?i)\b(pita|souvlaki|skewer|pasta|moussaka|soup|perch|cod|chicken|pork|beef|tortilla|wings)\b`)
)

// ParseMealStructureFromRaw builds top-level foods and optional sets (parts) from meal text.
// A set starts with "Set name: first item …"; further "+" chunks join the set until the next set label or a main-course line.
func ParseMealStructureFromRaw(raw string) []MealItem {
	raw = strings.TrimSpace(raw)
	if raw == "" || itemSkipRawRe.MatchString(raw) {
		return nil
	}
	var out []MealItem
	var group *MealItem

	flushGroup := func() {
		if group == nil || len(group.Parts) == 0 {
			group = nil
			return
		}
		group.Grams = sumItemGrams(group.Parts)
		out = append(out, *group)
		group = nil
	}

	for _, chunk := range splitMealChunks(raw) {
		if strings.HasPrefix(strings.TrimSpace(chunk), "→") && strings.Contains(chunk, "**") {
			continue
		}
		if m := itemSetLabelRe.FindStringSubmatch(chunk); len(m) > 2 && !strings.Contains(m[1], "/") {
			flushGroup()
			label := strings.TrimSpace(m[1])
			first := strings.TrimSpace(m[2])
			group = &MealItem{Name: label}
			if it, ok := parseOneMealChunk(first); ok {
				group.Parts = append(group.Parts, it)
			}
			continue
		}
		if group != nil {
			if itemSetBreakHintRe.MatchString(chunk) {
				flushGroup()
			} else {
				if it, ok := parseOneMealChunk(chunk); ok {
					group.Parts = append(group.Parts, it)
					continue
				}
			}
		}
		if dish, ok := parseDishChunk(chunk); ok && len(dish.Parts) > 0 {
			flushGroup()
			out = append(out, dish)
			continue
		}
		if it, ok := parseOneMealChunk(chunk); ok {
			flushGroup()
			out = append(out, it)
		}
	}
	flushGroup()
	return out
}

func sumItemGrams(items []MealItem) float64 {
	var g float64
	for _, it := range items {
		g += it.Grams
	}
	return g
}

// FlattenMealStructure returns leaf lines (sets expanded) for gaps / search.
func FlattenMealStructure(items []MealItem) []MealItem {
	var out []MealItem
	for _, it := range items {
		if len(it.Parts) > 0 {
			out = append(out, it.Parts...)
		} else {
			out = append(out, it)
		}
	}
	return out
}

// ParseMealItemsFromRaw turns markdown meal text into name+grams rows.
func ParseMealItemsFromRaw(raw string) []MealItem {
	return FlattenMealStructure(ParseMealStructureFromRaw(raw))
}

// MealItemsNeedReparse reports stale DB rows (one blob or empty) that should be rebuilt from raw.
func MealItemsNeedReparse(items []MealItem, raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	if len(items) == 0 {
		return true
	}
	if len(items) == 1 {
		it := items[0]
		if len(it.Parts) == 0 && (strings.Contains(it.Name, " + ") || strings.Contains(it.Name, "→")) {
			return true
		}
		if len(it.Parts) == 0 && it.Grams == 0 && strings.Contains(raw, "+") {
			return true
		}
	}
	return false
}

// CompactMealRaw rewrites meal text to name + grams items, keeping trailing total line(s).
func CompactMealRaw(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || itemSkipRawRe.MatchString(raw) {
		return raw
	}
	structured := ParseMealStructureFromRaw(raw)
	if len(structured) == 0 {
		return raw
	}
	var parts []string
	for _, it := range structured {
		if len(it.Parts) > 0 {
			var inner []string
			for _, p := range it.Parts {
				inner = append(inner, formatMealItemCompact(p))
			}
			parts = append(parts, fmt.Sprintf("%s: %s", it.Name, strings.Join(inner, " + ")))
			continue
		}
		parts = append(parts, formatMealItemCompact(it))
	}
	out := strings.Join(parts, " + ")
	if suf := mealTotalsTail(raw); suf != "" {
		out += " " + suf
	}
	return out
}

func formatMealItemCompact(it MealItem) string {
	name := normalizeItemName(it.Name)
	if it.Grams <= 0 {
		return name
	}
	if it.Grams == float64(int64(it.Grams)) {
		return fmt.Sprintf("%s %.0f g", name, it.Grams)
	}
	return fmt.Sprintf("%s %.1f g", name, it.Grams)
}

func normalizeItemName(name string) string {
	name = strings.TrimSpace(name)
	name = itemSaladPrefixRe.ReplaceAllString(name, "")
	return strings.TrimSpace(name)
}

func flattenItems(items []MealItem) []MealItem {
	var out []MealItem
	for _, it := range items {
		if len(it.Parts) > 0 {
			out = append(out, it.Parts...)
			continue
		}
		out = append(out, it)
	}
	return out
}

func mealTotalsTail(raw string) string {
	// Keep inline subtotals: everything from the first slot total marker onward.
	if loc := itemTotalCutRe.FindStringIndex(raw); loc != nil {
		return strings.TrimSpace(raw[loc[0]:])
	}
	return ""
}

func splitMealChunks(raw string) []string {
	parts := strings.Split(raw, " + ")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if itemOnlyTotalRe.MatchString(p) || (strings.HasPrefix(p, "→") && strings.Contains(p, "**")) {
			if sod, kc := parseMealTotalFromRaw(p); sod >= 400 && kc >= 400 {
				continue
			}
			if len(out) > 0 {
				out[len(out)-1] += " + " + p
			}
			continue
		}
		out = append(out, p)
	}
	return out
}

func parseDishChunk(ch string) (MealItem, bool) {
	ch = stripChunkForParse(ch)
	colon := strings.IndexRune(ch, ':')
	if colon <= 0 {
		return MealItem{}, false
	}
	head := strings.TrimSpace(ch[:colon])
	tail := strings.TrimSpace(ch[colon+1:])
	if tail == "" {
		return MealItem{}, false
	}
	loc := itemGramsRe.FindStringIndex(head)
	if loc == nil {
		return MealItem{}, false
	}
	dishName := strings.TrimSpace(head[:loc[0]])
	dishGrams := parseFloatLoose(itemGramsRe.FindStringSubmatch(head)[1])
	if dishName == "" {
		dishName = "salad"
	}
	var parts []MealItem
	for _, p := range splitMealChunks(tail) {
		if it, ok := parseOneMealChunk(p); ok {
			parts = append(parts, it)
		}
	}
	if len(parts) == 0 {
		return MealItem{}, false
	}
	return MealItem{Name: dishName, Grams: dishGrams, Parts: parts, Note: ch}, true
}

func parseOneMealChunk(ch string) (MealItem, bool) {
	ch = stripChunkForParse(ch)
	if ch == "" || itemSkipRawRe.MatchString(ch) {
		return MealItem{}, false
	}
	grams := 0.0
	if m := itemParenGRe.FindStringSubmatch(ch); len(m) > 1 {
		grams = parseFloatLoose(m[1])
	}
	if grams == 0 {
		loc := itemGramsRe.FindStringIndex(ch)
		if loc != nil {
			m := itemGramsRe.FindStringSubmatch(ch)
			grams = parseFloatLoose(m[1])
			rest := ch[loc[1]:]
			if xm := itemTimesRe.FindStringSubmatch(rest); len(xm) > 1 {
				n := parseFloatLoose(xm[1])
				if n > 1 {
					grams *= n
				}
			}
		}
	}
	if grams == 0 {
		if m := itemEggCountRe.FindStringSubmatch(ch); len(m) > 1 {
			n := parseFloatLoose(m[1])
			if n > 0 {
				grams = n * 55
			}
		}
	}
	count := 0.0
	if m := itemTimesRe.FindStringSubmatch(ch); len(m) > 1 {
		count = parseFloatLoose(m[1])
	}
	if grams == 0 && count > 0 {
		if pm := itemParenGRe.FindStringSubmatch(ch); len(pm) > 1 {
			grams = parseFloatLoose(pm[1])
		} else if g := itemCountGuessGrams(count, ch); g > 0 {
			grams = g
		}
	}
	name := itemNameFromChunk(ch)
	if name == "" {
		return MealItem{}, false
	}
	return MealItem{Name: name, Grams: grams, Count: count, Note: ch}, true
}

func stripChunkForParse(ch string) string {
	ch = strings.TrimSpace(ch)
	if loc := itemTotalCutRe.FindStringIndex(ch); loc != nil {
		ch = strings.TrimSpace(ch[:loc[0]])
	}
	ch = itemInlinePairRe.ReplaceAllString(ch, "")
	return strings.TrimSpace(ch)
}

// itemCountGuessGrams estimates total grams for "wafers ×4" style lines without explicit g.
func itemCountGuessGrams(n float64, ch string) float64 {
	if n <= 0 {
		return 0
	}
	low := strings.ToLower(ch)
	per := 0.0
	switch {
	case strings.Contains(low, "wafer"), strings.Contains(low, "rusk"), strings.Contains(low, "bar"),
		strings.Contains(low, "sachet"), strings.Contains(low, "croissant"), strings.Contains(low, "granola"):
		per = 10
	default:
		return 0
	}
	return n * per
}

func itemNameFromChunk(ch string) string {
	ch = stripChunkForParse(ch)
	name := ch
	if loc := itemGramsRe.FindStringIndex(ch); loc != nil {
		name = ch[:loc[0]]
	} else if strings.Contains(ch, "×") {
		name = itemTimesRe.Split(ch, 2)[0]
	}
	name = strings.TrimSpace(name)
	name = strings.Trim(name, ":-,;")
	name = strings.TrimSpace(itemTimesRe.ReplaceAllString(name, ""))
	name = itemSaladPrefixRe.ReplaceAllString(name, "")
	name = strings.Join(strings.Fields(name), " ")
	return name
}
