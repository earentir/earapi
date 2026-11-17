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
	if strings.TrimSpace(it.FoodID) != "" || strings.TrimSpace(it.StockLotID) != "" {
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

func rollUpDishMacrosFromParts(it *MealItem) {
	if it == nil || len(it.Parts) == 0 {
		return
	}
	var batch, k, n float64
	for i := range it.Parts {
		p := &it.Parts[i]
		batch += p.Grams
		k += p.Kcal
		n += p.SodiumMg
	}
	factor := 1.0
	if batch <= 0 {
		it.Grams = 0
		it.Kcal = 0
		it.SodiumMg = 0
		return
	}
	eaten := it.Grams
	if eaten <= 0 || eaten >= batch-0.01 {
		it.Grams = batch
	} else {
		factor = eaten / batch
	}
	it.Kcal = k * factor
	it.SodiumMg = n * factor
}

// scaleMealItemMacrosRespectingCatalogLinks stretches only name-only / unlinked lines (markdown calibration).
func scaleMealItemMacrosRespectingCatalogLinks(it *MealItem, factor float64) {
	if it == nil || factor <= 0 || math.Abs(factor-1) < 0.005 {
		return
	}
	if len(it.Parts) > 0 {
		for i := range it.Parts {
			scaleMealItemMacrosRespectingCatalogLinks(&it.Parts[i], factor)
		}
		rollUpDishMacrosFromParts(it)
		return
	}
	if strings.TrimSpace(it.FoodID) != "" || strings.TrimSpace(it.StockLotID) != "" {
		return
	}
	if it.Kcal > 0 {
		it.Kcal *= factor
	}
	if it.SodiumMg > 0 {
		it.SodiumMg *= factor
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
		dayKcal += m.Kcal
	}
	totK, _ := ApplyDayMarginTotals(dayKcal, 0, marginPercent)
	return totK
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

// CalibrateMealSlotsFromRawTotals scales unlinked item lines to match markdown **~mg / ~kcal** (legacy import only; not used on live days).
func CalibrateMealSlotsFromRawTotals(d *Day, marginPercent float64) {
	if d == nil || d.Meals == nil {
		return
	}
	for slot, m := range d.Meals {
		raw := strings.TrimSpace(m.Raw)
		if raw == "" || len(m.Items) == 0 {
			continue
		}
		legSod, legKcal := parseMealTotalFromRaw(raw)
		if legKcal <= 0 && legSod <= 0 {
			continue
		}
		RecomputeMealSlotTotalsFromItems(&m)
		catalogSlot := m.Kcal
		if catalogSlot <= 0 {
			d.Meals[slot] = m
			continue
		}
		if MealHasCatalogLinkedItems(m) {
			d.Meals[slot] = m
			continue
		}
		if legKcal <= 0 {
			d.Meals[slot] = m
			continue
		}
		f := legKcal / catalogSlot
		if math.Abs(f-1) < 0.005 {
			d.Meals[slot] = m
			continue
		}
		for i := range m.Items {
			scaleMealItemMacrosRespectingCatalogLinks(&m.Items[i], f)
		}
		RecomputeMealSlotTotalsFromItems(&m)
		d.Meals[slot] = m
	}
}

// CalibrateDayTotalsFromLegacy scales unlinked lines to match day legacy column (legacy import only; not used on live days).
func CalibrateDayTotalsFromLegacy(d *Day, marginPercent float64) {
	if d == nil || d.LegacyTotals.Kcal <= 0 {
		return
	}
	if DayHasCatalogLinkedItems(d) {
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
		m := d.Meals[slot]
		for i := range m.Items {
			scaleMealItemMacrosRespectingCatalogLinks(&m.Items[i], f)
		}
		RecomputeMealSlotTotalsFromItems(&m)
		d.Meals[slot] = m
	}
}
