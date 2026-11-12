package health

import (
	"regexp"
	"strings"
)

var (
	stockLeftRe     = regexp.MustCompile(`(?i)~?\s*(\d+(?:\.\d+)?)\s*(g|kg|l)\s+left`)
	stockBatchRe    = regexp.MustCompile(`(?i)batch\s+~?\s*(\d+(?:\.\d+)?)\s*(g|kg|l)\b`)
	stockNewBatchRe = regexp.MustCompile(`(?i)new batch\s+~?\s*(\d+(?:\.\d+)?)\s*(g|kg|l)\b`)
	stockYieldGramsRe   = regexp.MustCompile(`(?i)→\s*~?\s*(\d+(?:\.\d+)?)\s*g\b`)
	stockBatchPlainRe   = regexp.MustCompile(`(?i)\bbatch\s+(\d+(?:\.\d+)?)\s*(g|kg|l)\b`)
	stockSetIngredients = regexp.MustCompile(`(?i)^set ingredients:\s*(.+)$`)
	stockCountLeft  = regexp.MustCompile(`(?i)(?:\*\*)?(\d+)\s+left(?:\*\*)?`)
	stockCupLeft    = regexp.MustCompile(`(?i)(\d+)\s+left[^.]{0,40}cup`)
	stockSavedG     = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*g\s+saved`)
)

func parseStockMarkdown(content string) []StockLot {
	lines := strings.Split(content, "\n")
	var out []StockLot
	taken := map[string]bool{}
	inSection := false
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.EqualFold(line, "## Current batch stock") {
			inSection = true
			continue
		}
		if inSection && strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "### ") {
			break
		}
		if !inSection || !strings.HasPrefix(line, "- ") {
			continue
		}
		if strings.HasPrefix(line, "- ~~") {
			continue
		}
		text := strings.TrimPrefix(line, "- ")
		setLine := ""
		for j := i + 1; j < len(lines); j++ {
			sub := strings.TrimSpace(lines[j])
			if strings.HasPrefix(sub, "- ") {
				sub = strings.TrimSpace(strings.TrimPrefix(sub, "- "))
			}
			if m := stockSetIngredients.FindStringSubmatch(sub); len(m) > 1 {
				setLine = "Set ingredients: " + strings.TrimSpace(m[1])
				i = j
				break
			}
			if sub == "" {
				continue
			}
			if strings.HasPrefix(sub, "- ") || strings.HasPrefix(sub, "### ") {
				break
			}
		}
		for _, lot := range parseStockBullet(text, setLine, taken) {
			if lot.Name == "" {
				continue
			}
			out = append(out, lot)
		}
	}
	return out
}

func parseStockBullet(text, setLine string, taken map[string]bool) []StockLot {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	low := strings.ToLower(text)
	depleted := stockBulletDepleted(text)
	if strings.Contains(low, "carcass soup #3") && strings.Contains(low, "night batch") {
		return parseCarcassSoup3Stock(text, taken, depleted)
	}
	lot := stockLotFromText(text, setLine, taken)
	if lot == nil {
		return extractStillInStockSubLots(text, taken, stockBulletDepleted(text))
	}
	var out []StockLot
	if depleted {
		lot.RemainingGrams = 0
		if lot.BatchGrams <= 0 {
			lot.BatchGrams = 1
		}
		stockLotSetActive(lot, false)
	} else {
		stockLotSetActive(lot, lot.RemainingGrams > 0.01)
	}
	out = append(out, *lot)
	out = append(out, extractStillInStockSubLots(text, taken, depleted)...)
	return out
}

func stockBulletDepleted(text string) bool {
	if strings.HasPrefix(text, "~~") {
		return true
	}
	low := strings.ToLower(text)
	for _, marker := range []string{
		"batch finished", "stock cleared", "depleted", "nothing in stock",
		"thrown away", "in stock: 0", "skewer stock depleted", "wings cleared",
		"rest thrown", "batch finished", "soup finished", "mush batch finished",
		"finished -", "finished sep", "finished oct", "finished aug",
	} {
		if strings.Contains(low, marker) {
			return true
		}
	}
	if stockLeftRe.FindStringSubmatch(text) != nil {
		return false
	}
	if strings.Contains(low, "still in stock") {
		return false
	}
	if stockCupLeft.FindStringSubmatch(text) != nil {
		return false
	}
	if stockSavedG.FindStringSubmatch(text) != nil {
		return false
	}
	return true
}

func stockItemsFromSetLine(setLine string) []MealItem {
	setLine = strings.TrimSpace(setLine)
	if setLine == "" {
		return nil
	}
	if m := stockSetIngredients.FindStringSubmatch(setLine); len(m) > 1 {
		setLine = strings.TrimSpace(m[1])
	}
	return ParseMealItemsFromRaw(setLine)
}

func stockLotFromText(text, setLine string, taken map[string]bool) *StockLot {
	name := stockExtractName(text)
	name = strings.Trim(name, "* ")
	if name == "" {
		return nil
	}
	notes := stockLotShortNotes(text)
	batchG := stockParseBatchGrams(text)
	remG := stockParseRemainingGrams(text)
	if remG <= 0 && !stockBulletDepleted(text) {
		remG = batchG
	}
	if batchG <= 0 {
		batchG = remG
	}
	if batchG <= 0 && stockBulletDepleted(text) {
		batchG = 1
	}
	if batchG <= 0 {
		return nil
	}
	if remG > batchG {
		remG = batchG
	}
	kcal, sod := parseNutrientsFromNotes(text)
	lot := StockLot{
		Name:            name,
		PreparedDate:    stockParsePreparedDate(text),
		BatchGrams:      batchG,
		RemainingGrams:  remG,
		Notes:           notes,
		Items:           stockItemsFromSetLine(setLine),
		KcalPer100g:     kcal,
		SodiumMgPer100g: sod,
	}
	id := UniqueSlug(name, taken)
	taken[id] = true
	lot.ID = id
	return &lot
}

func stockExtractName(text string) string {
	text = strings.TrimLeft(text, "~")
	if i := strings.Index(text, ":"); i > 0 && i < 80 {
		return cleanFoodDisplayName(strings.TrimSpace(text[:i]))
	}
	if i := strings.Index(text, " ("); i > 0 && i < 80 {
		return cleanFoodDisplayName(strings.TrimSpace(text[:i]))
	}
	return cleanFoodDisplayName(text)
}

func stockParsePreparedDate(text string) string {
	re := regexp.MustCompile(`(?i)\b(?:Nov|Oct|Sep|Aug|Jul|Jun|May|Apr|Mar|Feb|Jan)\s+\d{1,2}\b`)
	m := re.FindString(text)
	return strings.TrimSpace(m)
}

func stockLotShortNotes(text string) string {
	text = strings.TrimSpace(text)
	if i := strings.Index(strings.ToLower(text), "set ingredients"); i > 0 {
		text = strings.TrimSpace(text[:i])
	}
	if len(text) > 220 {
		text = text[:220] + "…"
	}
	return text
}

func stockParseBatchGrams(text string) float64 {
	if m := stockNewBatchRe.FindStringSubmatch(text); len(m) >= 3 {
		return stockToGrams(parseFloatLoose(m[1]), m[2])
	}
	if m := stockBatchPlainRe.FindStringSubmatch(text); len(m) >= 3 {
		return stockToGrams(parseFloatLoose(m[1]), m[2])
	}
	if m := stockBatchRe.FindStringSubmatch(text); len(m) >= 3 {
		return stockToGrams(parseFloatLoose(m[1]), m[2])
	}
	// e.g. bean salad "→ 730 g ≈", sole "~825 g"
	if m := stockYieldGramsRe.FindStringSubmatch(text); len(m) >= 2 {
		return parseFloatLoose(m[1])
	}
	return 0
}

func stockParseRemainingGrams(text string) float64 {
	if m := stockLeftRe.FindStringSubmatch(text); len(m) >= 3 {
		return stockToGrams(parseFloatLoose(m[1]), m[2])
	}
	if m := stockSavedG.FindStringSubmatch(text); len(m) >= 2 {
		return parseFloatLoose(m[1])
	}
	if m := stockCupLeft.FindStringSubmatch(text); len(m) >= 2 {
		n := parseFloatLoose(m[1])
		if strings.Contains(strings.ToLower(text), "200 g cup") {
			return n * 200
		}
		return n * 200
	}
	if m := stockCountLeft.FindStringSubmatch(text); len(m) >= 2 {
		n := parseFloatLoose(m[1])
		if strings.Contains(strings.ToLower(text), "cereal") || strings.Contains(strings.ToLower(text), "375 g") {
			return n * 375
		}
		return n * 200
	}
	if strings.Contains(strings.ToLower(text), "still in stock") {
		if m := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*g`).FindStringSubmatch(text); len(m) >= 2 {
			return parseFloatLoose(m[1])
		}
	}
	return 0
}

func stockToGrams(v float64, unit string) float64 {
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "kg":
		return v * 1000
	case "l":
		return v * 1000
	default:
		return v
	}
}

func parseCarcassSoup3Stock(text string, taken map[string]bool, depleted bool) []StockLot {
	notes := text
	kcal, sod := parseNutrientsFromNotes(text)
	var out []StockLot
	tomatoRem := 700.0
	plainRem := 1060.0
	lefts := stockLeftRe.FindAllStringSubmatch(text, -1)
	if len(lefts) >= 1 {
		tomatoRem = stockToGrams(parseFloatLoose(lefts[0][1]), lefts[0][2])
	}
	if len(lefts) >= 2 {
		plainRem = stockToGrams(parseFloatLoose(lefts[1][1]), lefts[1][2])
	}
	tomatoBatch := 1900.0
	if m := stockNewBatchRe.FindStringSubmatch(text); len(m) >= 3 {
		tomatoBatch = stockToGrams(parseFloatLoose(m[1]), m[2])
	}
	plainBatch := plainRem
	if depleted {
		tomatoRem, plainRem = 0, 0
	}
	for _, spec := range []struct{ name string; batch, rem float64 }{
		{"Soup-tomato mix (carcass #3)", tomatoBatch, tomatoRem},
		{"Carcass soup non-tomato (#3)", plainBatch, plainRem},
	} {
		id := UniqueSlug(spec.name, taken)
		taken[id] = true
		row := StockLot{
			ID: id, Name: spec.name, PreparedDate: "Nov 7",
			BatchGrams: spec.batch, RemainingGrams: spec.rem,
			Notes: notes, KcalPer100g: kcal, SodiumMgPer100g: sod,
		}
		stockLotSetActive(&row, spec.rem > 0.01)
		out = append(out, row)
	}
	return out
}

var stockSavedSubLot = regexp.MustCompile(`(?i)([a-z0-9][a-z0-9\-\s]+?)\s+(\d+(?:\.\d+)?)\s*g\s+saved`)

func extractStillInStockSubLots(text string, taken map[string]bool, parentFinished bool) []StockLot {
	if !strings.Contains(strings.ToLower(text), "still in stock") {
		return nil
	}
	var out []StockLot
	for _, m := range stockSavedSubLot.FindAllStringSubmatch(text, -1) {
		if len(m) < 3 {
			continue
		}
		match := m[0]
		if stockSavedMatchStruck(text, match) {
			continue
		}
		name := cleanFoodDisplayName(strings.TrimSpace(m[1]))
		if name == "" {
			continue
		}
		g := parseFloatLoose(m[2])
		if g <= 0 {
			continue
		}
		kcal, sod := parseNutrientsFromNotes(text)
		id := UniqueSlug(name, taken)
		taken[id] = true
		sub := StockLot{
			ID: id, Name: name, PreparedDate: stockParsePreparedDate(text),
			BatchGrams: g, RemainingGrams: g, Notes: stockSubLotNoteSnippet(text, match),
			KcalPer100g: kcal, SodiumMgPer100g: sod,
		}
		// "saved … STILL IN STOCK" inside a finished-batch paragraph is historical; verify in UI.
		stockLotSetActive(&sub, !parentFinished)
		out = append(out, sub)
	}
	return out
}

func stockSavedMatchStruck(text, match string) bool {
	idx := strings.Index(strings.ToLower(text), strings.ToLower(match))
	if idx < 0 {
		return false
	}
	before := text[:idx]
	return strings.Count(before, "~~")%2 == 1
}

func stockSubLotNoteSnippet(text, match string) string {
	low := strings.ToLower(text)
	mlow := strings.ToLower(match)
	idx := strings.Index(low, mlow)
	if idx < 0 {
		return match
	}
	start := idx
	if start > 60 {
		start -= 60
	} else {
		start = 0
	}
	end := idx + len(match) + 100
	if end > len(text) {
		end = len(text)
	}
	return strings.TrimSpace(text[start:end])
}

func parseYogurtStock(text string, taken map[string]bool) []StockLot {
	notes := text
	kcal, sod := parseNutrientsFromNotes(text)
	batch := 800.0
	rem := 600.0
	if m := stockCountLeft.FindStringSubmatch(text); len(m) >= 2 {
		n := parseFloatLoose(m[1])
		rem = n * 200
	}
	name := "Yogurt stock (Marata 1.7%)"
	id := UniqueSlug(name, taken)
	taken[id] = true
	lot := StockLot{
		ID: id, Name: name, PreparedDate: stockParsePreparedDate(text),
		BatchGrams: batch, RemainingGrams: rem, Notes: notes,
		KcalPer100g: kcal, SodiumMgPer100g: sod,
	}
	stockLotSetActive(&lot, true)
	return []StockLot{lot}
}

func parseCerealStock(text string, taken map[string]bool) []StockLot {
	specs := []struct{ name string }{
		{"Bonora Action Fruit"},
		{"Millhouse Super Fruits & Dark Choc"},
		{"Millhouse Line & Diet Dark Chocolate"},
	}
	kcal, sod := parseNutrientsFromNotes(text)
	var out []StockLot
	for _, spec := range specs {
		id := UniqueSlug(spec.name, taken)
		taken[id] = true
		row := StockLot{
			ID: id, Name: spec.name, PreparedDate: stockParsePreparedDate(text),
			BatchGrams: 375, RemainingGrams: 375, Notes: text,
			KcalPer100g: kcal, SodiumMgPer100g: sod,
		}
		stockLotSetActive(&row, true)
		out = append(out, row)
	}
	for i := range out {
		out[i].KcalPer100g = kcal
		out[i].SodiumMgPer100g = sod
	}
	return out
}

func filterStockFromFoods(foods []Food) []Food {
	var out []Food
	for _, f := range foods {
		if foodLooksLikeStockEntry(f) {
			continue
		}
		out = append(out, f)
	}
	return out
}

func foodLooksLikeStockEntry(f Food) bool {
	text := f.Name + " " + f.Notes
	return skipFoodBulletForCatalog(text)
}

func skipFoodBulletForCatalog(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	if strings.HasPrefix(text, "~~") {
		return true
	}
	low := strings.ToLower(text)
	if strings.Contains(low, "yogurt stock (") || strings.HasPrefix(low, "yogurt stock") {
		return true
	}
	if strings.Contains(low, "cereal stock (") || strings.HasPrefix(low, "cereal stock") {
		return true
	}
	if strings.Contains(low, "skewer") && strings.Contains(low, "wings stock") {
		return true
	}
	if strings.Contains(low, "kebab stock (") {
		return true
	}
	if strings.Contains(low, "batch stock (") && !strings.Contains(low, "temporary") {
		return true
	}
	if strings.Contains(low, "(recipe,") && strings.Contains(low, "batch") {
		return true
	}
	if strings.Contains(low, "homemade fasolada") && strings.Contains(low, "batch") {
		return true
	}
	if strings.HasPrefix(low, "sep 1 batch split:") {
		return true
	}
	return false
}
