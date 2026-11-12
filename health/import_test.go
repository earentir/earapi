package health

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseWeightSnippet(t *testing.T) {
	snippet := `| Date | kg | RHR (bpm) | Note |
| --- | --- | --- | --- |
| Oct 23 | 144.3 | - | **PROTOCOL ✓** |
| Oct 12 | 148.3 | - | evening |
`
	rows := parseWeightMarkdown(snippet, 2026)
	if len(rows) != 2 {
		t.Fatalf("want 2 rows got %d", len(rows))
	}
	if rows[0].Date != "2026-10-23" || rows[0].Kg != 144.3 {
		t.Fatalf("oct 23: %+v", rows[0])
	}
	if rows[1].ID != "w_2026-10-12_1" {
		t.Fatalf("id: %s", rows[1].ID)
	}
}

func TestParseFoodSnippet(t *testing.T) {
	snippet := `## Reference kcal / sodium per 100 g (whole foods in regular rotation)

| Food | kcal / 100 g | sodium mg / 100 g | Notes |
| --- | --- | --- | --- |
| Pasta (cooked, no salt) | ~155 | ~1 | Bow pasta |
`
	foods := parseFoodMarkdown(snippet)
	if len(foods) != 1 {
		t.Fatalf("want 1 food got %d", len(foods))
	}
	if foods[0].Name != "Pasta (cooked, no salt)" {
		t.Fatalf("name %q", foods[0].Name)
	}
	if foods[0].KcalPer100g == nil || *foods[0].KcalPer100g != 155 {
		t.Fatalf("kcal %+v", foods[0].KcalPer100g)
	}
}

func TestParseMealTotalWithMargin(t *testing.T) {
	raw := `Soup-tomato mix 400 g ~92 / ~112 + moussaka 200 g ~440 / ~390 → **~573 / ~554 → +10% margin = ~609**`
	sod, kc := parseMealTotalFromRaw(raw)
	if sod != 573 || kc != 609 {
		t.Fatalf("got sod=%v kc=%v", sod, kc)
	}
}

func TestHydrateDayFromRaw(t *testing.T) {
	d := Day{
		Date: "2026-11-10",
		Meals: map[string]MealSlot{
			"lunch":  {Raw: "- (no lunch, user)"},
			"dinner": {Raw: `soup → **~573 / ~554 → +10% margin = ~609**`},
			"snack":  {Raw: `eggs → **~1,129 / ~857 → +10% margin = ~926**`},
		},
		BP: "**~1,702 / ~1,535**",
	}
	hydrateDayFromRaw(&d)
	if d.BP != "" {
		t.Fatalf("bp should be cleared, got %q", d.BP)
	}
	if d.Meals["dinner"].Kcal != 0 || d.Meals["snack"].Kcal != 0 {
		t.Fatalf("meal slot totals should not load legacy; meals %+v", d.Meals)
	}
}

func TestParseMonthSnippet(t *testing.T) {
	snippet := `| Date | Lunch (mg / kcal) | Dinner (mg / kcal) | Snack (mg / kcal) | Day total (mg / kcal) | BP | Walking |
| Nov 1 | lunch text **~198 / ~380** | | snack **~162 / ~477** | **~360 / ~857** | | |
`
	days := parseMonthMarkdown(snippet, 2026)
	if len(days) != 1 {
		t.Fatalf("want 1 day got %d", len(days))
	}
	if days[0].Date != "2026-11-01" {
		t.Fatalf("date %s", days[0].Date)
	}
	if days[0].LegacyTotals.Kcal != 857 || days[0].LegacyTotals.SodiumMg != 360 {
		t.Fatalf("legacy totals %+v", days[0].LegacyTotals)
	}
}

func TestImportFromRepoMDData(t *testing.T) {
	dir := filepath.Join("..", "mddata")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("mddata not present")
	}
	w, f, d, stock, err := ImportFromMDDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(w) < 10 {
		t.Fatalf("expected many weights got %d", len(w))
	}
	if len(f) < 20 {
		t.Fatalf("expected many foods got %d", len(f))
	}
	if len(d) < 5 {
		t.Fatalf("expected many days got %d", len(d))
	}
	if len(stock) < 5 {
		t.Fatalf("expected stock lots from Misc got %d", len(stock))
	}
	active := 0
	for _, lot := range stock {
		if stockLotIsActive(lot) {
			active++
		}
	}
	if active < 3 {
		t.Fatalf("expected several active stock lots got %d", active)
	}
	// fuzzy sanity
	names := strings.Join([]string{f[0].Name}, "")
	if names == "" {
		t.Fatal("empty food")
	}
	var hasMoussaka bool
	for _, food := range f {
		low := strings.ToLower(food.Name)
		if strings.Contains(low, "rebuy list") || strings.Contains(low, "approved ready-meal rebuy") {
			t.Fatalf("policy doc must not become a catalog food: %q", food.Name)
		}
		if strings.Contains(low, "moussaka ready meal") {
			hasMoussaka = true
		}
	}
	if !hasMoussaka {
		t.Fatal("expected Moussaka ready meal from EU table row")
	}
	_ = w
}

func TestCatalogStaplesForNovMeals(t *testing.T) {
	dir := filepath.Join("..", "mddata")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("mddata not present")
	}
	_, foods, days, _, err := ImportFromMDDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"EVOO", "Fytro", "soup-tomato mix", "roasted wings", "Marata 1.7%"} {
		f, ok := BestFoodMatch(foods, n)
		if !ok {
			t.Fatalf("%q: no catalog match", n)
		}
		if f.KcalPer100g == nil || f.SodiumMgPer100g == nil {
			t.Fatalf("%q -> %q missing per 100 g", n, f.Name)
		}
	}
	var d *Day
	for i := range days {
		if days[i].Date == "2026-11-12" {
			d = &days[i]
			break
		}
	}
	if d == nil {
		t.Fatal("expected 2026-11-12 in month logs")
	}
	var walk func(items []MealItem, slot string)
	walk = func(items []MealItem, slot string) {
		for _, it := range items {
			if len(it.Parts) > 0 {
				walk(it.Parts, slot)
				continue
			}
			name := strings.TrimSpace(it.Name)
			if name == "" || it.Grams <= 0 {
				continue
			}
			f, ok := BestFoodMatch(foods, name)
			if !ok {
				t.Fatalf("%s unmatched: %q", slot, name)
			}
			if f.KcalPer100g == nil || f.SodiumMgPer100g == nil {
				t.Fatalf("%s %q -> %q incomplete per 100 g", slot, name, f.Name)
			}
		}
	}
	for slot, m := range d.Meals {
		walk(m.Items, slot)
	}
}
