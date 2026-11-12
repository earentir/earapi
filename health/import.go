package health

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ImportFromMDDir loads weight, foods, stock lots, and monthly day logs from mddata/.
func ImportFromMDDir(dir string) ([]WeighIn, []Food, []Day, []StockLot, error) {
	var weights []WeighIn
	var foods []Food
	var days []Day
	var stock []StockLot

	wpath := filepath.Join(dir, "Health Tracker - Weight.md")
	if b, err := os.ReadFile(wpath); err == nil {
		weights = parseWeightMarkdown(string(b), 2026)
	}

	fpath := filepath.Join(dir, "Health Tracker - Food Data.md")
	if b, err := os.ReadFile(fpath); err == nil {
		foods = filterStockFromFoods(parseFoodMarkdown(string(b)))
	}

	mpath := filepath.Join(dir, "Health Tracker - Misc.md")
	if b, err := os.ReadFile(mpath); err == nil {
		stock = filterStockLotsForImport(parseStockMarkdown(string(b)))
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "Health Tracker - 20") || !strings.HasSuffix(name, ".md") {
			continue
		}
		if strings.Contains(name, ":Zone") {
			continue
		}
		year := extractYearFromMonthFilename(name)
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		days = append(days, parseMonthMarkdown(string(b), year)...)
	}
	return weights, foods, days, stock, nil
}

func extractYearFromMonthFilename(name string) int {
	re := regexp.MustCompile(`20\d{2}`)
	m := re.FindString(name)
	if m == "" {
		return 2026
	}
	y, _ := strconv.Atoi(m)
	if y == 0 {
		return 2026
	}
	return y
}

func parseWeightMarkdown(content string, defaultYear int) []WeighIn {
	lines := strings.Split(content, "\n")
	var out []WeighIn
	dayCount := map[string]int{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") || strings.Contains(line, "---") || strings.Contains(line, "Date |") {
			continue
		}
		cells := splitTableRow(line)
		if len(cells) < 2 {
			continue
		}
		dateStr := strings.TrimSpace(cells[0])
		kgStr := strings.TrimSpace(cells[1])
		if dateStr == "Date" || kgStr == "kg" {
			continue
		}
		iso, _ := parseTrackerDate(dateStr, defaultYear)
		if iso == "" {
			continue
		}
		kg, kgRaw := parseKgCell(kgStr)
		var rhr *int
		if len(cells) > 2 {
			rhr = parseRHR(cells[2])
		}
		note := ""
		if len(cells) > 3 {
			note = strings.TrimSpace(cells[3])
		}
		protocol := strings.Contains(strings.ToUpper(note), "PROTOCOL")
		dayCount[iso]++
		id := fmt.Sprintf("w_%s_%d", iso, dayCount[iso])
		w := WeighIn{ID: id, Date: iso, Kg: kg, RHRBpm: rhr, Note: note, Protocol: protocol, KgRaw: kgRaw}
		if kg == 0 && kgRaw != "" {
			w.Note = strings.TrimSpace(kgRaw + ", " + note)
		}
		out = append(out, w)
	}
	return out
}

func parseKgCell(s string) (float64, string) {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" || s == "\u2014" {
		return 0, s
	}
	// first number in cell
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)`)
	m := re.FindString(s)
	if m == "" {
		return 0, s
	}
	f, err := strconv.ParseFloat(m, 64)
	if err != nil {
		return 0, s
	}
	if strings.Contains(s, "/") {
		return f, s
	}
	return f, ""
}

func parseRHR(s string) *int {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" || s == "\u2014" {
		return nil
	}
	re := regexp.MustCompile(`(\d+)`)
	m := re.FindString(s)
	if m == "" {
		return nil
	}
	n, _ := strconv.Atoi(m)
	return &n
}

var monthMap = map[string]int{
	"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6,
	"jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
}

func parseTrackerDate(s string, year int) (string, int) {
	s = strings.TrimSpace(strings.TrimPrefix(s, "~"))
	parts := strings.Fields(s)
	if len(parts) < 2 {
		return "", 0
	}
	mon := strings.ToLower(parts[0])
	m, ok := monthMap[mon[:3]]
	if !ok {
		return "", 0
	}
	if year == 0 {
		year = 2026
	}
	day, _ := strconv.Atoi(strings.TrimSuffix(parts[1], ","))
	if day < 1 || day > 31 {
		return "", m
	}
	return fmt.Sprintf("%04d-%02d-%02d", year, m, day), m
}

func parseFoodMarkdown(content string) []Food {
	lines := strings.Split(content, "\n")
	var out []Food
	taken := map[string]bool{}
	inRefTable := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Reference kcal") || strings.Contains(line, "Reference per 100 g") {
			inRefTable = true
			continue
		}
		if inRefTable && strings.HasPrefix(line, "## ") && !strings.Contains(line, "Reference") {
			inRefTable = false
		}
		if inRefTable && strings.HasPrefix(line, "|") && !strings.Contains(line, "---") {
			cells := splitTableRow(line)
			if len(cells) >= 3 && cells[0] != "Food" {
				name := strings.TrimSpace(cells[0])
				if name == "" {
					continue
				}
				var f Food
				if isEUNutritionTableRow(cells) {
					f = foodFromEUTableCells(cells)
				} else {
					f = foodFromLegacyTableCells(cells)
				}
				id := UniqueSlug(name, taken)
				taken[id] = true
				f.ID = id
				out = append(out, f)
			}
		}
		// bullet lines as note-only foods
		if strings.HasPrefix(line, "- ") && !strings.HasPrefix(line, "- |") {
			text := strings.TrimPrefix(line, "- ")
			if len(text) < 8 || skipFoodBulletForCatalog(text) {
				continue
			}
			name := text
			if idx := strings.Index(text, ":"); idx > 0 && idx < 60 {
				name = strings.TrimSpace(text[:idx])
			} else if idx := strings.Index(text, " ("); idx > 0 && idx < 60 {
				name = strings.TrimSpace(text[:idx])
			}
			name = cleanFoodDisplayName(name)
			if len(name) > 80 {
				name = name[:80]
			}
			id := UniqueSlug(name, taken)
			taken[id] = true
			kcal, sod := parseNutrientsFromNotes(text)
			f := Food{ID: id, Name: name, Notes: text, KcalPer100g: kcal, SodiumMgPer100g: sod}
			ApplyLabelFromNotes(&f)
			out = append(out, f)
		}
	}
	return out
}

func parseApproxNumber(s string) *float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" || s == "\u2014" {
		return nil
	}
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)`)
	m := re.FindString(s)
	if m == "" {
		return nil
	}
	f, err := strconv.ParseFloat(m, 64)
	if err != nil {
		return nil
	}
	return &f
}

func parseMonthMarkdown(content string, year int) []Day {
	lines := strings.Split(content, "\n")
	var out []Day
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") || strings.Contains(line, "---") {
			continue
		}
		if strings.Contains(line, "Date |") || strings.Contains(line, "Lunch (mg") {
			continue
		}
		cells := splitTableRow(line)
		if len(cells) < 5 {
			continue
		}
		dateRaw := strings.TrimSpace(cells[0])
		if strings.EqualFold(dateRaw, "Date") {
			continue
		}
		iso, _ := parseTrackerDate(dateRaw, year)
		if iso == "" {
			// try "Nov 1" with only month in file
			continue
		}
		d := Day{
			Date:  iso,
			Meals: map[string]MealSlot{},
		}
		if len(cells) > 1 {
			d.Meals["lunch"] = MealSlot{Raw: strings.TrimSpace(cells[1])}
		}
		if len(cells) > 2 {
			d.Meals["dinner"] = MealSlot{Raw: strings.TrimSpace(cells[2])}
		}
		if len(cells) > 3 {
			d.Meals["snack"] = MealSlot{Raw: strings.TrimSpace(cells[3])}
		}
		for i := 4; i < len(cells); i++ {
			c := strings.TrimSpace(cells[i])
			if c == "" {
				continue
			}
			if looksLikeDayTotalColumn(c) {
				if d.Totals.Kcal == 0 && d.Totals.SodiumMg == 0 {
					sod, kc := parseDayTotalCell(c)
					d.Totals = DayTotals{SodiumMg: sod, Kcal: kc}
				}
				continue
			}
			if looksLikeOnlyTotal(c) {
				if d.Totals.Kcal == 0 && d.Totals.SodiumMg == 0 {
					sod, kc := parseDayTotalCell(c)
					d.Totals = DayTotals{SodiumMg: sod, Kcal: kc}
				}
				continue
			}
			if looksLikeBP(c) {
				d.BP = c
				continue
			}
			putExtraMealRaw(&d, c)
		}
		hydrateDayFromRaw(&d)
		out = append(out, d)
	}
	return out
}

var (
	mealMarginRE = regexp.MustCompile(`(?i)\*\*~?([\d,.]+)\s*/\s*~?([\d,.]+)\s*(?:→|->)\s*\+?\s*[\d.]+\s*%\s*margin\s*=\s*~?([\d,.]+)`)
	mealTotalRE  = regexp.MustCompile(`\*\*~?([\d,.]+)\s*/\s*~?([\d,.]+)\*\*`)
	pairRE       = regexp.MustCompile(`~?([\d,.]+)\s*/\s*~?([\d,.]+)`)
)

// MealRawDeclaresMargin is true when the meal cell ends with an explicit +N% margin (= …) total.
func MealRawDeclaresMargin(raw string) bool {
	raw = strings.TrimSpace(raw)
	return raw != "" && mealMarginRE.MatchString(raw)
}

// ShouldApplyMealSlotMargin applies +margin to catalog line sums unless the markdown cell
// already ends with an unmargined **~mg / ~kcal** total (early-month logs). Empty raw (UI/API) still gets margin.
func ShouldApplyMealSlotMargin(meal *MealSlot, marginPercent float64) bool {
	if meal == nil || marginPercent <= 0 {
		return false
	}
	raw := strings.TrimSpace(meal.Raw)
	if raw == "" {
		return true
	}
	if MealRawDeclaresMargin(raw) {
		return true
	}
	if mealTotalRE.MatchString(raw) {
		return false
	}
	if mealRawHasTrailingSlotTotal(raw) {
		return false
	}
	return true
}

// mealRawHasTrailingSlotTotal detects Aug–Oct style cells that end with ~mg / ~kcal (no ** wrapper).
func mealRawHasTrailingSlotTotal(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	ms := pairRE.FindAllStringSubmatch(raw, -1)
	if len(ms) == 0 {
		return false
	}
	last := ms[len(ms)-1]
	sod, kc := parseFloatLoose(last[1]), parseFloatLoose(last[2])
	if sod <= 0 && kc <= 0 {
		return false
	}
	// Snack cells often append " + 4.66 km" after the total pair — still a declared slot total.
	if kc > 50 && kc < 8000 {
		return true
	}
	return sod > 50 && kc > 0 && kc < 8000
}

func parseMealTotalFromRaw(raw string) (sod, kc float64) {
	if ms := mealMarginRE.FindAllStringSubmatch(raw, -1); len(ms) > 0 {
		last := ms[len(ms)-1]
		return parseFloatLoose(last[1]), parseFloatLoose(last[3])
	}
	if ms := mealTotalRE.FindAllStringSubmatch(raw, -1); len(ms) > 0 {
		last := ms[len(ms)-1]
		return parseFloatLoose(last[1]), parseFloatLoose(last[2])
	}
	if ms := pairRE.FindAllStringSubmatch(raw, -1); len(ms) > 0 {
		last := ms[len(ms)-1]
		return parseFloatLoose(last[1]), parseFloatLoose(last[2])
	}
	return 0, 0
}

func parseDayTotalCell(s string) (sod, kc float64) {
	sod, kc = parseMealTotalFromRaw(s)
	if sod > 0 || kc > 0 {
		return sod, kc
	}
	m2 := pairRE.FindStringSubmatch(s)
	if len(m2) >= 3 {
		return parseFloatLoose(m2[1]), parseFloatLoose(m2[2])
	}
	return 0, 0
}

func looksLikeBP(s string) bool {
	ok, _ := regexp.MatchString(`^\d{2,3}\s*/\s*\d{2,3}$`, strings.TrimSpace(s))
	return ok
}

func looksLikeOnlyTotal(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > 48 {
		return false
	}
	sod, kc := parseDayTotalCell(s)
	return sod > 0 || kc > 0
}

func relocateMisplacedMealText(d *Day) {
	move := func(s *string) {
		v := strings.TrimSpace(*s)
		if v == "" {
			return
		}
		if looksLikeOnlyTotal(v) {
			if d.Totals.Kcal == 0 && d.Totals.SodiumMg == 0 {
				d.Totals.SodiumMg, d.Totals.Kcal = parseDayTotalCell(v)
			}
			*s = ""
			return
		}
		if len(v) > 60 && strings.Contains(v, "/") {
			putExtraMealRaw(d, v)
			*s = ""
		}
	}
	move(&d.BP)
	move(&d.Walking)
}

func putExtraMealRaw(d *Day, raw string) {
	if d.Meals == nil {
		d.Meals = map[string]MealSlot{}
	}
	for _, slot := range []string{"dinner", "snack", "lunch"} {
		m := d.Meals[slot]
		if strings.TrimSpace(m.Raw) == "" {
			m.Raw = raw
			d.Meals[slot] = m
			return
		}
	}
	m := d.Meals["snack"]
	m.Raw = strings.TrimSpace(m.Raw + " + " + raw)
	d.Meals["snack"] = m
}

// hydrateDayFromRaw fills slot/day totals from markdown meal text when numeric fields are empty.
func hydrateDayFromRaw(d *Day) {
	if d == nil {
		return
	}
	relocateMisplacedMealText(d)
	if d.LegacyTotals.Kcal == 0 && d.LegacyTotals.SodiumMg == 0 &&
		(d.Totals.Kcal > 0 || d.Totals.SodiumMg > 0) {
		d.LegacyTotals = d.Totals
	}
	for _, slot := range []string{"lunch", "dinner", "snack"} {
		m := d.Meals[slot]
		before := m.Raw
		m.Raw = stripMisplacedDayTotalFromRaw(m.Raw)
		if strings.TrimSpace(m.Raw) != "" && (m.Raw != before || MealItemsNeedReparse(m.Items, m.Raw)) {
			if parsed := ParseMealStructureFromRaw(m.Raw); len(parsed) > 0 {
				m.Items = parsed
			}
		}
		d.Meals[slot] = m
	}
}

var mealRawDayTotalSuffixRE = regexp.MustCompile(`(?i)\s*\+\s*(?:→|->)?\s*\*\*~?[\d,.]+\s*/\s*~?[\d,.]+\*\*.*$`)

// stripMisplacedDayTotalFromRaw removes a day-total column wrongly appended to a meal cell during import.
func stripMisplacedDayTotalFromRaw(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	return strings.TrimSpace(mealRawDayTotalSuffixRE.ReplaceAllString(raw, ""))
}

func mealRawHasMisplacedDayTotalSuffix(raw string) bool {
	return stripMisplacedDayTotalFromRaw(raw) != strings.TrimSpace(raw)
}

// looksLikeDayTotalColumn detects the day-total markdown column even with long trailing notes (over 48 chars).
func looksLikeDayTotalColumn(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || !strings.Contains(s, "**") {
		return false
	}
	sod, kc := parseDayTotalCell(s)
	return sod >= 400 && kc >= 400
}

// DayHasMisplacedDayTotalInMeals reports import artifact where the day total column was glued to a meal slot.
func DayHasMisplacedDayTotalInMeals(d *Day) bool {
	if d == nil {
		return false
	}
	for _, m := range d.Meals {
		if mealRawHasMisplacedDayTotalSuffix(m.Raw) {
			return true
		}
	}
	return false
}

// RepairDayMealImportArtifacts strips misplaced day totals and re-parses items when needed.
func RepairDayMealImportArtifacts(d *Day) bool {
	if d == nil {
		return false
	}
	changed := false
	for slot, m := range d.Meals {
		clean := stripMisplacedDayTotalFromRaw(m.Raw)
		if clean != m.Raw {
			m.Raw = clean
			changed = true
		}
		if strings.TrimSpace(m.Raw) != "" {
			if changed || MealItemsNeedReparse(m.Items, m.Raw) {
				if parsed := ParseMealStructureFromRaw(m.Raw); len(parsed) > 0 {
					m.Items = parsed
					changed = true
				}
			}
		}
		d.Meals[slot] = m
	}
	return changed
}

func parseFloatLoose(s string) float64 {
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func splitTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
