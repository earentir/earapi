package health

import (
	"regexp"
	"strings"
)

var (
	itemGramsRe    = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*g\b`)
	itemParenGRe   = regexp.MustCompile(`(?i)\((\d+(?:\.\d+)?)\s*g`)
	itemTimesRe    = regexp.MustCompile(`×\s*(\d+(?:\.\d+)?)`)
	itemTotalCutRe = regexp.MustCompile(`(?i)\s*(?:→|->)\s*\*\*`)
	itemSkipRawRe  = regexp.MustCompile(`(?i)^(—|-|–)\s*(\(no |\s*$)|^\s*$`)
)

// ParseMealItemsFromRaw turns markdown meal text into name+grams rows.
func ParseMealItemsFromRaw(raw string) []MealItem {
	raw = strings.TrimSpace(raw)
	if raw == "" || itemSkipRawRe.MatchString(raw) {
		return nil
	}
	if loc := itemTotalCutRe.FindStringIndex(raw); loc != nil {
		raw = strings.TrimSpace(raw[:loc[0]])
	}
	var out []MealItem
	for _, chunk := range splitMealChunks(raw) {
		it, ok := parseOneMealChunk(chunk)
		if ok {
			out = append(out, it)
		}
	}
	return out
}

func splitMealChunks(raw string) []string {
	parts := strings.Split(raw, " + ")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseOneMealChunk(ch string) (MealItem, bool) {
	ch = strings.TrimSpace(ch)
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
	name := itemNameFromChunk(ch)
	if name == "" {
		return MealItem{}, false
	}
	return MealItem{Name: name, Grams: grams, Note: ch}, true
}

func itemNameFromChunk(ch string) string {
	name := ch
	if loc := itemGramsRe.FindStringIndex(ch); loc != nil {
		name = ch[:loc[0]]
	}
	name = strings.TrimSpace(name)
	name = strings.Trim(name, ":-·,;(")
	name = strings.TrimSpace(itemTimesRe.ReplaceAllString(name, ""))
	name = strings.Join(strings.Fields(name), " ")
	return name
}
