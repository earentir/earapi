package health

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNov12LunchSoupStockRescaleDoesNotDropKcal(t *testing.T) {
	dir := filepath.Join("..", "mddata")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("mddata")
	}
	db, err := OpenSharedDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	st, err := NewStore(db, t.TempDir(), "audit")
	if err != nil {
		t.Fatal(err)
	}
	_, foods, days, stock, err := ImportFromMDDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.ReplaceImported(dir, nil, foods, days, stock); err != nil {
		t.Fatal(err)
	}
	st.mu.Lock()
	d0, ok0 := st.readDayUnlocked("2026-11-12")
	st.mu.Unlock()
	if !ok0 {
		t.Fatal("read after import")
	}
	for _, it := range d0.Meals["lunch"].Items {
		if strings.Contains(strings.ToLower(it.Name), "soup-tomato") {
			t.Logf("soup in DB after import (pre-GetDay): kcal=%.1f na=%.0f stock=%q", it.Kcal, it.SodiumMg, it.StockLotID)
		}
	}
	for _, it := range d0.Meals["snack"].Items {
		t.Logf("snack pre-GetDay: %q kcal=%.0f", it.Name, it.Kcal)
	}
	d, ok := st.GetDay("2026-11-12")
	if !ok || d == nil {
		t.Fatal("no day after import")
	}
	lunch := d.Meals["lunch"]
	var soupK float64
	var walk func([]MealItem)
	walk = func(items []MealItem) {
		for _, it := range items {
			if len(it.Parts) > 0 {
				walk(it.Parts)
			}
			if strings.Contains(strings.ToLower(it.Name), "soup-tomato") {
				soupK = it.Kcal
			}
		}
	}
	walk(lunch.Items)
	if soupK < 90 {
		t.Fatalf("soup-tomato kcal after GetDay %.0f want ~101", soupK)
	}
	if lunch.Kcal < 400 {
		t.Fatalf("lunch slot kcal %.0f want ~423 with margin", lunch.Kcal)
	}
	for _, it := range d.Meals["snack"].Items {
		t.Logf("snack post-GetDay: %q kcal=%.0f stock=%q", it.Name, it.Kcal, it.StockLotID)
	}
	for _, slot := range []string{"lunch", "dinner", "snack"} {
		m := d.Meals[slot]
		t.Logf("%s: %.0f kcal %.0f mg", slot, m.Kcal, m.SodiumMg)
	}
	if diff := d.Totals.Kcal - d.LegacyTotals.Kcal; diff < -5 || diff > 5 {
		t.Fatalf("day kcal %.0f legacy %.0f (diff %.0f)", d.Totals.Kcal, d.LegacyTotals.Kcal, diff)
	}
}
