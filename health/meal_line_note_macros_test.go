package health

import "testing"

func TestApplyMealItemNoteMacrosLabelPer100(t *testing.T) {
	it := MealItem{Name: "rusks", Grams: 36, Note: `Rusks no salt 12 g ×3 (36 g, label 387 kcal / ~15 mg per 100 g)`}
	applyMealItemNoteMacros(&it)
	if it.Kcal < 138 || it.Kcal > 140 {
		t.Fatalf("kcal %v want ~139", it.Kcal)
	}
}

func TestCalibrateMealSlotsFromRawTotals(t *testing.T) {
	d := &Day{Meals: map[string]MealSlot{
		"lunch": {
			Raw: `Pasta 120 g → **~198 / ~380**`,
			Items: []MealItem{
				{Name: "Pasta", Grams: 120, Kcal: 417, SodiumMg: 10},
			},
		},
	}}
	CalibrateMealSlotsFromRawTotals(d, 10)
	if d.Meals["lunch"].Items[0].Kcal < 379 || d.Meals["lunch"].Items[0].Kcal > 381 {
		t.Fatalf("scaled kcal %v want ~380", d.Meals["lunch"].Items[0].Kcal)
	}
}
