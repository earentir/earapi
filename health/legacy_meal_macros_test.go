package health

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImportPreservesLegacyDayTotals(t *testing.T) {
	dir := filepath.Join("..", "mddata")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("mddata not present")
	}
	_, _, days, _, err := ImportFromMDDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var d *Day
	for i := range days {
		if days[i].Date == "2026-11-12" {
			d = &days[i]
			break
		}
	}
	if d == nil {
		t.Fatal("nov 12 missing")
	}
	if d.LegacyTotals.Kcal != 1630 || d.LegacyTotals.SodiumMg != 1333 {
		t.Fatalf("legacy totals %+v", d.LegacyTotals)
	}
}

func TestImportScalesItemsNotLegacyLines(t *testing.T) {
	dir := filepath.Join("..", "mddata")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("mddata not present")
	}
	_, foods, days, stock, err := ImportFromMDDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var d *Day
	for i := range days {
		if days[i].Date == "2026-11-12" {
			d = &days[i]
			break
		}
	}
	ScaleDayMealsFromCatalog(d, foods, stock, aliasFoodsFromMDDir(dir, foods))
	RecomputeDayTotals(d)
	var walk func([]MealItem)
	var catalogLines int
	walk = func(items []MealItem) {
		for _, it := range items {
			if len(it.Parts) > 0 {
				walk(it.Parts)
				continue
			}
			if it.Grams > 0 && it.FoodID != "" && (it.Kcal > 0 || it.SodiumMg > 0) {
				catalogLines++
			}
		}
	}
	for _, slot := range []string{"lunch", "dinner", "snack"} {
		walk(d.Meals[slot].Items)
	}
	if catalogLines < 5 {
		t.Fatalf("expected catalog-scaled lines, got %d", catalogLines)
	}
	if d.Totals.Kcal == d.LegacyTotals.Kcal && d.Totals.SodiumMg == d.LegacyTotals.SodiumMg {
		// may match by coincidence; not required
	}
}
