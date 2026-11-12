package health

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestImportScalesFytroAndEVOOOnNov12(t *testing.T) {
	dir := filepath.Join("..", "mddata")
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
	if d == nil {
		t.Fatal("no nov 12")
	}
	ScaleDayMealsFromCatalog(d, foods, stock, aliasFoodsFromMDDir(dir, foods))
	var fytroNa, evooK float64
	var foundF, foundE bool
	var walk func([]MealItem)
	walk = func(items []MealItem) {
		for _, it := range items {
			if len(it.Parts) > 0 {
				walk(it.Parts)
				continue
			}
			n := strings.TrimSpace(it.Name)
			if strings.EqualFold(n, "fytro") {
				foundF = true
				fytroNa = it.SodiumMg
				if it.FoodID == "" {
					t.Fatal("fytro food_id empty")
				}
			}
			if strings.EqualFold(n, "evoo") {
				foundE = true
				evooK = it.Kcal
				if it.FoodID == "" {
					t.Fatal("evoo food_id empty")
				}
			}
		}
	}
	for _, m := range d.Meals {
		walk(m.Items)
	}
	if !foundF || !foundE {
		t.Fatalf("missing lines f=%v e=%v", foundF, foundE)
	}
	if fytroNa <= 0 {
		t.Fatalf("fytro sodium want >0 got %v", fytroNa)
	}
	if evooK <= 0 {
		t.Fatalf("evoo kcal want >0 got %v", evooK)
	}
}
