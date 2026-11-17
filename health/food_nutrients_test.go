package health

import "testing"

func TestNutrientsForFood_prefersPackLabel(t *testing.T) {
	// Sklavenitis tray: 380 kcal / 400 g → 95 kcal/100 g (not a stale 140 kcal/100 g column).
	stale := 140.0
	lg, lk := 400.0, 380.0
	f := Food{
		KcalPer100g: &stale,
		LabelGrams:  &lg,
		LabelKcal:   &lk,
	}
	k, _ := nutrientsForFood(&f)
	if k == nil || *k < 94.9 || *k > 95.1 {
		t.Fatalf("kcal/100g %v want 95 from pack label", k)
	}
}

func TestMealItemKcal_packLabelFood(t *testing.T) {
	k100 := 155.0
	f := Food{ID: "pasta", KcalPer100g: &k100, Name: "Pasta (cooked, no salt)"}
	it := MealItem{Name: "Pasta (cooked, no salt)", FoodID: "pasta", Grams: 300}
	got := mealItemKcalFromFood(&it, &f, &k100)
	if got < 464 || got > 466 {
		t.Fatalf("300 g pasta kcal %v want 465", got)
	}
}