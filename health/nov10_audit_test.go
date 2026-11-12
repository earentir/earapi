package health

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditNov10SnackGap(t *testing.T) {
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
		if days[i].Date == "2026-11-10" {
			d = &days[i]
			break
		}
	}
	if d == nil {
		t.Fatal("no nov 10")
	}
	aliases := aliasFoodsFromMDDir(dir, foods)
	ScaleDayMealsFromCatalog(d, foods, stock, aliases)
	RecomputeDayTotalsWithMargin(d, 10)
	snack := d.Meals["snack"]
	fmt.Printf("snack +10%%: %.0f kcal legacy day %.0f\n", snack.Kcal, d.LegacyTotals.Kcal)
	for _, it := range snack.Items {
		if len(it.Parts) > 0 {
			fmt.Printf("  DISH %q %.0fg kcal=%.0f\n", it.Name, it.Grams, it.Kcal)
			for _, p := range it.Parts {
				fmt.Printf("    %q %.0fg kcal=%.0f\n", p.Name, p.Grams, p.Kcal)
			}
		} else {
			fmt.Printf("  %q %.0fg kcal=%.0f food=%q\n", it.Name, it.Grams, it.Kcal, it.FoodID)
		}
	}
	if d.Totals.Kcal < d.LegacyTotals.Kcal-120 {
		t.Logf("day kcal gap: catalog %.0f legacy %.0f", d.Totals.Kcal, d.LegacyTotals.Kcal)
	}
	_ = strings.TrimSpace
}
