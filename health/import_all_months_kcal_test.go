package health

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestImportAllMonthsDayKcalWithinLegacy(t *testing.T) {
	dir := filepath.Join("..", "mddata")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("mddata not present")
	}
	_, foods, days, stock, err := ImportFromMDDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	aliases := aliasFoodsFromMDDir(dir, foods)
	margin := float64(DefaultMealMarginPercent)
	var worst string
	var worstDiff float64
	for i := range days {
		d := &days[i]
		if d.LegacyTotals.Kcal <= 0 {
			continue
		}
		ensureMealSlotFallbackItems(d)
		ScaleDayMealsFromCatalog(d, foods, stock, aliases)
		repairZeroKcalSlotsFromRaw(d)
		CalibrateMealSlotsFromRawTotals(d, margin)
		CalibrateDayTotalsFromLegacy(d, margin)
		RecomputeDayTotalsWithMargin(d, margin)
		diff := math.Abs(d.Totals.Kcal - d.LegacyTotals.Kcal)
		if diff > worstDiff {
			worstDiff = diff
			worst = d.Date
		}
		if diff > 25 {
			t.Errorf("%s catalog %.0f legacy %.0f Δ %.0f", d.Date, d.Totals.Kcal, d.LegacyTotals.Kcal, d.Totals.Kcal-d.LegacyTotals.Kcal)
		}
	}
	t.Logf("worst day %s |Δ|=%.0f", worst, worstDiff)
}
