package health

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNov12TotalsAfterBoneAndMillhouseFix(t *testing.T) {
	dir := filepath.Join("..", "mddata")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("no mddata")
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
	CalibrateMealSlotsFromRawTotals(d, 10)
	RecomputeDayTotalsWithMargin(d, 10)
	dinner := d.Meals["dinner"]
	for _, it := range dinner.Items {
		if strings.Contains(strings.ToLower(it.Name), "wing") && it.Grams != 175 {
			t.Fatalf("wings grams %v want 175", it.Grams)
		}
	}
	if diff := d.Totals.Kcal - d.LegacyTotals.Kcal; math.Abs(diff) > 25 {
		t.Fatalf("day kcal %.0f legacy %.0f (diff %.0f)", d.Totals.Kcal, d.LegacyTotals.Kcal, diff)
	}
	_ = foods
	_ = stock
}
