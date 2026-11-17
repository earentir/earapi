package health

import (
	"math"
	"testing"
)

func TestApplyMealItemNoteMacrosLabelPer100(t *testing.T) {
	it := MealItem{Name: "rusks", Grams: 36, Note: `Rusks no salt 12 g ×3 (36 g, label 387 kcal / ~15 mg per 100 g)`}
	applyMealItemNoteMacros(&it)
	if it.Kcal < 138 || it.Kcal > 140 {
		t.Fatalf("kcal %v want ~139", it.Kcal)
	}
}

func TestCalibrateMealSlotsFromRawTotals_skipsCatalogLinkedStaleHighRaw(t *testing.T) {
	pastaK := 155.0
	d := &Day{Meals: map[string]MealSlot{
		"lunch": {
			Raw: `old total → **~3045 / ~1444**`,
			Items: []MealItem{
				{Name: "xirino me selino", FoodID: "xir", Grams: 400, Kcal: 380},
				{
					Name: "pasta",
					Parts: []MealItem{
						{Name: "Pasta (cooked, no salt)", FoodID: "pasta", Grams: 300, Kcal: pastaK / 100 * 300},
						{Name: "pesto", FoodID: "pesto", Grams: 22, Kcal: 160},
						{Name: "kefalotiri", FoodID: "kef", Grams: 7, Kcal: 37},
					},
					Kcal: pastaK/100*300 + 160 + 37,
				},
			},
		},
	}}
	was := 380.0
	CalibrateMealSlotsFromRawTotals(d, 10)
	if k := d.Meals["lunch"].Items[0].Kcal; math.Abs(k-was) > 1 {
		t.Fatalf("xirino kcal %v want ~380 unchanged, not scaled to stale raw 1444", k)
	}
	m := d.Meals["lunch"]
	RecomputeMealSlotTotalsFromItems(&m)
	if m.Kcal < 1040 || m.Kcal > 1045 {
		t.Fatalf("lunch catalog kcal %v want ~1042 before meal margin", m.Kcal)
	}
}

func TestCalibrateMealSlotsFromRawTotalsSkipsCatalogLinkedUI(t *testing.T) {
	d := &Day{Meals: map[string]MealSlot{
		"dinner": {
			Raw: `old blob → **~90 / ~312**`,
			Items: []MealItem{
				{Name: "ready meal", FoodID: "meal", Grams: 450, Kcal: 527, SodiumMg: 1980},
				{
					Name: "salad",
					Grams: 289,
					Parts: []MealItem{
						{Name: "hot horn pepper", FoodID: "pepper", Grams: 15, Kcal: 5, SodiumMg: 1},
						{Name: "EVOO", FoodID: "evoo", Grams: 4, Kcal: 35, SodiumMg: 0},
					},
					Kcal: 40,
				},
			},
		},
	}}
	CalibrateMealSlotsFromRawTotals(d, 10)
	part := d.Meals["dinner"].Items[1].Parts[0]
	if part.Kcal < 4.5 || part.Kcal > 5.5 {
		t.Fatalf("pepper kcal %v want ~5 (not crushed by stale raw)", part.Kcal)
	}
	if d.Meals["dinner"].Items[0].Kcal < 520 {
		t.Fatalf("ready meal kcal %v want catalog (~527)", d.Meals["dinner"].Items[0].Kcal)
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
