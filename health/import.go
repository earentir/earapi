package health

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ImportFromMDDir loads weight, foods, and monthly day logs from mddata/.
func ImportFromMDDir(dir string) ([]WeighIn, []Food, []Day, error) {
	var weights []WeighIn
	var foods []Food
	var days []Day

	wpath := filepath.Join(dir, "Health Tracker - Weight.md")
	if b, err := os.ReadFile(wpath); err == nil {
		weights = parseWeightMarkdown(string(b), 2026)
	}

	fpath := filepath.Join(dir, "Health Tracker - Food Data.md")
	if b, err := os.ReadFile(fpath); err == nil {
		foods = parseFoodMarkdown(string(b))
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
	return weights, foods, days, nil
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
	var lastMonth int
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
		iso, month := parseTrackerDate(dateStr, defaultYear, lastMonth)
		if month > 0 {
			lastMonth = month
		}
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
			w.Note = strings.TrimSpace(kgRaw + " · " + note)
		}
		out = append(out, w)
	}
	return out
}

func parseKgCell(s string) (float64, string) {
	s = strings.TrimSpace(s)
	if s == "" || s == "—" || s == "-" {
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
	if s == "" || s == "—" || s == "-" {
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

func parseTrackerDate(s string, year int, lastMonth int) (string, int) {
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
		if strings.Contains(line, "Reference kcal") {
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
				kcal := parseApproxNumber(cells[1])
				sod := parseApproxNumber(cells[2])
				notes := ""
				if len(cells) > 3 {
					notes = strings.TrimSpace(cells[3])
				}
				id := UniqueSlug(name, taken)
				taken[id] = true
				f := Food{ID: id, Name: name, Notes: notes}
				if kcal != nil {
					f.KcalPer100g = kcal
				}
				if sod != nil {
					f.SodiumMgPer100g = sod
				}
				out = append(out, f)
			}
		}
		// bullet lines as note-only foods
		if strings.HasPrefix(line, "- ") && !strings.HasPrefix(line, "- |") {
			text := strings.TrimPrefix(line, "- ")
			if len(text) < 8 {
				continue
			}
			name := text
			if idx := strings.Index(text, ":"); idx > 0 && idx < 60 {
				name = strings.TrimSpace(text[:idx])
			} else if idx := strings.Index(text, " ("); idx > 0 && idx < 60 {
				name = strings.TrimSpace(text[:idx])
			}
			if len(name) > 80 {
				name = name[:80]
			}
			id := UniqueSlug(name, taken)
			taken[id] = true
			out = append(out, Food{ID: id, Name: name, Notes: text})
		}
	}
	return out
}

func parseApproxNumber(s string) *float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "—" || s == "-" {
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
		iso, _ := parseTrackerDate(dateRaw, year, 0)
		if iso == "" {
			// try "Nov 1" with only month in file
			continue
		}
		d := Day{
			Date:  iso,
			Meals: map[string]MealSlot{},
		}
		d.Meals["lunch"] = MealSlot{Raw: strings.TrimSpace(cells[1])}
		d.Meals["dinner"] = MealSlot{Raw: strings.TrimSpace(cells[2])}
		d.Meals["snack"] = MealSlot{Raw: strings.TrimSpace(cells[3])}
		if tot := strings.TrimSpace(cells[4]); tot != "" {
			sod, kc := parseDayTotalCell(tot)
			d.Totals = DayTotals{SodiumMg: sod, Kcal: kc}
		}
		if len(cells) > 5 {
			d.BP = strings.TrimSpace(cells[5])
		}
		if len(cells) > 6 {
			d.Walking = strings.TrimSpace(cells[6])
		}
		for slot, key := range map[string]string{"lunch": "lunch", "dinner": "dinner", "snack": "snack"} {
			if sod, kc := parseMealTotalFromRaw(d.Meals[key].Raw); sod > 0 || kc > 0 {
				m := d.Meals[key]
				m.SodiumMg = sod
				m.Kcal = kc
				d.Meals[slot] = m
			}
		}
		if d.Totals.Kcal == 0 && d.Totals.SodiumMg == 0 {
			RecomputeDayTotals(&d)
		}
		out = append(out, d)
	}
	return out
}

var mealTotalRE = regexp.MustCompile(`\*\*~?([\d,.]+)\s*/\s*~?([\d,.]+)\*\*`)

func parseMealTotalFromRaw(raw string) (sod, kc float64) {
	m := mealTotalRE.FindStringSubmatch(raw)
	if len(m) < 3 {
		return 0, 0
	}
	sod = parseFloatLoose(m[1])
	kc = parseFloatLoose(m[2])
	return sod, kc
}

func parseDayTotalCell(s string) (sod, kc float64) {
	m := mealTotalRE.FindStringSubmatch(s)
	if len(m) >= 3 {
		return parseFloatLoose(m[1]), parseFloatLoose(m[2])
	}
	// e.g. **~360 / ~857** without bold variants
	re := regexp.MustCompile(`~?([\d,.]+)\s*/\s*~?([\d,.]+)`)
	m2 := re.FindStringSubmatch(s)
	if len(m2) >= 3 {
		return parseFloatLoose(m2[1]), parseFloatLoose(m2[2])
	}
	return 0, 0
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
