package health

import "testing"

func TestMealItemKcalFromFood_threeEggs(t *testing.T) {
	k := 78.0
	f := &Food{Name: "Egg (1 large, ~55 g)", KcalPer100g: &k}
	it := &MealItem{Name: "eggs sunny side 3 eggs", Grams: 165, Note: "eggs sunny side 3 eggs"}
	got := mealItemKcalFromFood(it, f, &k)
	if got < 230 || got > 240 {
		t.Fatalf("kcal %v want ~234", got)
	}
}
