package health

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestImportNovemberNoZeroKcalSkewers(t *testing.T) {
	dir := filepath.Join("..", "mddata")
	_, foods, days, stock, err := ImportFromMDDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	aliases := aliasFoodsFromMDDir(dir, foods)
	for i := range days {
		if len(days[i].Date) < 10 || days[i].Date[:8] != "2026-11" {
			continue
		}
		ScaleDayMealsFromCatalog(&days[i], foods, stock, aliases)
		for slot, m := range days[i].Meals {
			var walk func([]MealItem)
			walk = func(items []MealItem) {
				for _, it := range items {
					if len(it.Parts) > 0 {
						walk(it.Parts)
						continue
					}
					n := strings.ToLower(it.Name)
					if it.Grams >= 20 && strings.Contains(n, "skewer") && it.Kcal <= 0 {
						t.Errorf("%s %s: %q %.0fg has no kcal", days[i].Date, slot, it.Name, it.Grams)
					}
					if it.Grams >= 5 && strings.Contains(n, "rusk") && it.Kcal <= 0 {
						t.Errorf("%s %s: %q %.0fg has no kcal", days[i].Date, slot, it.Name, it.Grams)
					}
				}
			}
			walk(m.Items)
		}
	}
}
