package health

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOct27SnackNotDayTotal(t *testing.T) {
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
		if days[i].Date == "2026-10-27" {
			d = &days[i]
			break
		}
	}
	if d == nil {
		t.Fatal("missing oct 27")
	}
	snack := d.Meals["snack"]
	if DayHasMisplacedDayTotalInMeals(d) {
		t.Fatalf("snack raw still has day total: %q", snack.Raw)
	}
	ScaleDayMealsFromCatalog(d, foods, stock)
	RecomputeDayTotals(d)
	if d.Totals.Kcal > 1200 {
		t.Fatalf("day kcal should be catalog sum not markdown day column, got %v", d.Totals.Kcal)
	}
	if d.LegacyTotals.Kcal != 2002 || d.LegacyTotals.SodiumMg != 3178 {
		t.Fatalf("legacy %+v", d.LegacyTotals)
	}
	s := d.Meals["snack"]
	if s.Kcal > 200 {
		t.Fatalf("snack kcal %v (expected ~166 from wafers)", s.Kcal)
	}
	for _, it := range s.Items {
		if strings.EqualFold(strings.TrimSpace(it.Name), "Corn wafers") {
			if it.Count != 4 {
				t.Fatalf("corn wafers count %v", it.Count)
			}
			if it.Grams < 35 || it.Grams > 45 {
				t.Fatalf("corn wafers grams %v", it.Grams)
			}
		}
	}
}
