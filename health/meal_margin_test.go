package health

import "testing"

func TestRecomputeDayTotalsWithMealMargin(t *testing.T) {
	d := &Day{
		Date:  "2026-01-01",
		Meals: map[string]MealSlot{},
	}
	d.Meals["lunch"] = MealSlot{
		Items: []MealItem{{Name: "x", Grams: 100, Kcal: 100, SodiumMg: 50}},
	}
	d.Meals["snack"] = MealSlot{
		Items: []MealItem{{Name: "y", Grams: 100, Kcal: 200, SodiumMg: 100}},
	}
	RecomputeDayTotalsWithMargin(d, 10)
	if d.Meals["lunch"].Kcal != 110 || d.Meals["lunch"].SodiumMg != 55 {
		t.Fatalf("lunch %+v", d.Meals["lunch"])
	}
	if d.Meals["snack"].Kcal != 220 || d.Meals["snack"].SodiumMg != 110 {
		t.Fatalf("snack %+v", d.Meals["snack"])
	}
	if d.Totals.Kcal != 330 || d.Totals.SodiumMg != 165 {
		t.Fatalf("day %+v", d.Totals)
	}
}
