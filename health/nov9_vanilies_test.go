package health

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNov9VaniliesSnackKcal(t *testing.T) {
	dir := filepath.Join("..", "mddata")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("mddata")
	}
	_, foods, days, stock, err := ImportFromMDDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var d *Day
	for i := range days {
		if days[i].Date == "2026-11-09" {
			d = &days[i]
			break
		}
	}
	if d == nil {
		t.Fatal("nov 9 missing")
	}
	aliases := aliasFoodsFromMDDir(dir, foods)
	ScaleDayMealsFromCatalog(d, foods, stock, aliases)
	snackSlot := d.Meals["snack"]
	RecomputeMealSlotTotalsFromItems(&snackSlot)
	d.Meals["snack"] = snackSlot
	RecomputeDayTotalsWithMargin(d, 10)
	snack := d.Meals["snack"]
	var vanK, grapeK float64
	var foundVan bool
	for _, it := range snack.Items {
		n := strings.ToLower(strings.TrimSpace(it.Name))
		if strings.Contains(n, "vanil") {
			foundVan = true
			vanK = it.Kcal
			if it.Grams < 140 {
				t.Fatalf("vanilies grams %v", it.Grams)
			}
		}
		if strings.Contains(n, "grape") {
			grapeK += it.Kcal
		}
	}
	if !foundVan || vanK < 60 {
		t.Fatalf("vanilies line missing or low kcal: found=%v kcal=%.0f", foundVan, vanK)
	}
	// Snack legacy ~342 kcal; +10% margin → ~376; catalog should be within ~15% of legacy snack.
	if snack.Kcal < 300 || snack.Kcal > 420 {
		t.Fatalf("snack kcal %.0f", snack.Kcal)
	}
	if diff := d.Totals.Kcal - d.LegacyTotals.Kcal; diff < -120 || diff > 60 {
		t.Fatalf("day kcal %.0f legacy %.0f", d.Totals.Kcal, d.LegacyTotals.Kcal)
	}
}
