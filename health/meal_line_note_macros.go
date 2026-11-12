package health

import (
	"math"
	"regexp"
	"strings"
	"unicode"
)

var (
	noteLabelPer100Re    = regexp.MustCompile(`(?i)label\s*:?\s*~?(\d+(?:\.\d+)?)\s*kcal\s*/\s*~?(\d+(?:\.\d+)?)\s*mg(?:\s*/\s*[^)]*)?\s*per\s*100\s*g`)
	noteLabelPer100AltRe = regexp.MustCompile(`(?i)label\s+(\d+(?:\.\d+)?)\s*kcal\s*/\s*~?(\d+(?:\.\d+)?)\s*mg\s*per\s*100\s*g`)
	notePerPortionRe     = regexp.MustCompile(`(?i)(?:label:\s*)?~?(\d+(?:\.\d+)?)\s*kcal\s*/\s*~?(\d+(?:\.\d+)?)\s*mg\s*per\s*portion`)
	noteTrailingPairRe   = regexp.MustCompile(`(?i)\s*~?(\d+(?:\.\d+)?)\s*/\s*~?(\d+(?:\.\d+)?)\s*$`)
)

// applyMealItemNoteMacros sets line kcal/sodium from parenthetical label text in the meal note (markdown import).
func applyMealItemNoteMacros(it *MealItem) {
	if it == nil || it.Grams <= 0 {
		return
	}
	note := strings.TrimSpace(it.Note)
	if note == "" {
		return
	}
	if m := notePerPortionRe.FindStringSubmatch(note); len(m) >= 3 {
		it.Kcal = parseFloatLoose(m[1])
		it.SodiumMg = parseFloatLoose(m[2])
		return
	}
	if m := noteLabelPer100Re.FindStringSubmatch(note); len(m) >= 3 {
		k100, s100 := parseFloatLoose(m[1]), parseFloatLoose(m[2])
		it.Kcal = k100 * it.Grams / 100
		it.SodiumMg = s100 * it.Grams / 100
		return
	}
	if m := noteLabelPer100AltRe.FindStringSubmatch(note); len(m) >= 3 {
		k100, s100 := parseFloatLoose(m[1]), parseFloatLoose(m[2])
		it.Kcal = k100 * it.Grams / 100
		it.SodiumMg = s100 * it.Grams / 100
		return
	}
	if m := noteTrailingPairRe.FindStringSubmatch(note); len(m) >= 3 {
		it.SodiumMg = parseFloatLoose(m[1])
		it.Kcal = parseFloatLoose(m[2])
	}
}

func scaleMealItemMacros(it *MealItem, factor float64) {
	if it == nil || factor <= 0 || factor == 1 {
		return
	}
	if it.Kcal > 0 {
		it.Kcal *= factor
	}
	if it.SodiumMg > 0 {
		it.SodiumMg *= factor
	}
	for i := range it.Parts {
		scaleMealItemMacros(&it.Parts[i], factor)
	}
}

// ensureMealSlotFallbackItems creates one line from the slot total when parsing produced no rows (sparse Aug–Oct cells).
func ensureMealSlotFallbackItems(d *Day) {
	if d == nil || d.Meals == nil {
		return
	}
	for slot, m := range d.Meals {
		raw := strings.TrimSpace(m.Raw)
		if raw == "" || itemSkipRawRe.MatchString(raw) || len(m.Items) > 0 {
			continue
		}
		sod, kc := parseMealTotalFromRaw(raw)
		if kc <= 0 {
			continue
		}
		slotName := slot
		if slotName != "" {
			r := []rune(slotName)
			r[0] = unicode.ToUpper(r[0])
			slotName = string(r)
		}
		m.Items = []MealItem{{
			Name:     slotName + " total",
			Grams:    1,
			Kcal:     kc,
			SodiumMg: sod,
			Note:     raw,
		}}
		d.Meals[slot] = m
	}
}

func catalogDayKcalBeforeStore(d *Day, marginPercent float64) float64 {
	if d == nil {
		return 0
	}
	var dayKcal float64
	for _, slot := range []string{"lunch", "dinner", "snack"} {
		m := d.Meals[slot]
		if len(m.Items) == 0 {
			continue
		}
		RecomputeMealSlotTotalsFromItems(&m)
		d.Meals[slot] = m
		k := m.Kcal
		if ShouldApplyMealSlotMargin(&m, marginPercent) {
			k = math.Round(k * (1 + marginPercent/100))
		}
		dayKcal += k
	}
	return dayKcal
}

// repairZeroKcalSlotsFromRaw rebuilds a slot from its markdown total when catalog scaling zeroed lines.
func repairZeroKcalSlotsFromRaw(d *Day) {
	if d == nil || d.Meals == nil {
		return
	}
	for slot, m := range d.Meals {
		raw := strings.TrimSpace(m.Raw)
		if raw == "" || itemSkipRawRe.MatchString(raw) {
			continue
		}
		sod, kc := parseMealTotalFromRaw(raw)
		if kc <= 0 {
			continue
		}
		RecomputeMealSlotTotalsFromItems(&m)
		if m.Kcal > 0 {
			continue
		}
		m.Items = []MealItem{{Name: slot + " total", Grams: 1, Kcal: kc, SodiumMg: sod, Note: raw}}
		d.Meals[slot] = m
	}
}

// CalibrateMealSlotsFromRawTotals scales item lines so each slot’s catalog total matches the
// handwritten **~mg / ~kcal** (or margined) total in the meal cell raw text.
func CalibrateMealSlotsFromRawTotals(d *Day, marginPercent float64) {
	if d == nil || d.Meals == nil {
		return
	}
	for slot, m := range d.Meals {
		raw := strings.TrimSpace(m.Raw)
		if raw == "" || len(m.Items) == 0 {
			continue
		}
		_, legKcal := parseMealTotalFromRaw(raw)
		if legKcal <= 0 {
			continue
		}
		RecomputeMealSlotTotalsFromItems(&m)
		catalogSlot := m.Kcal
		if ShouldApplyMealSlotMargin(&m, marginPercent) {
			catalogSlot = m.Kcal * (1 + marginPercent/100)
		}
		if catalogSlot <= 0 {
			d.Meals[slot] = m
			continue
		}
		f := legKcal / catalogSlot
		if math.Abs(f-1) < 0.005 {
			d.Meals[slot] = m
			continue
		}
		for i := range m.Items {
			scaleMealItemMacros(&m.Items[i], f)
		}
		RecomputeMealSlotTotalsFromItems(&m)
		d.Meals[slot] = m
	}
}

// CalibrateDayTotalsFromLegacy scales all meal lines when slot-level calibration still misses the day column.
func CalibrateDayTotalsFromLegacy(d *Day, marginPercent float64) {
	if d == nil || d.LegacyTotals.Kcal <= 0 {
		return
	}
	dayKcal := catalogDayKcalBeforeStore(d, marginPercent)
	if dayKcal <= 0 {
		return
	}
	f := d.LegacyTotals.Kcal / dayKcal
	if math.Abs(f-1) < 0.005 {
		return
	}
	for slot := range d.Meals {
		for i := range d.Meals[slot].Items {
			scaleMealItemMacros(&d.Meals[slot].Items[i], f)
		}
	}
}
