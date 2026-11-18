package health

import "testing"

func TestEdibleFractionScalesKcal(t *testing.T) {
	k := 200.0
	frac := 0.7
	f := Food{
		Name:           "Chicken wings",
		KcalPer100g:    &k,
		EdibleFraction: &frac,
	}
	it := MealItem{Name: "Chicken wings", FoodID: f.ID, Grams: 1000}
	got := mealItemKcalFromFood(&it, &f, &k)
	want := 200.0 * 1000 * 0.7 / 100
	if got < want-0.01 || got > want+0.01 {
		t.Fatalf("kcal %v want %v", got, want)
	}
}

func TestEdibleFractionPerCutInVariants(t *testing.T) {
	k := 100.0
	frac := 0.57
	f := Food{
		Name:        "Chicken leg",
		KcalPer100g: &k,
		Variants: &FoodVariants{
			Enabled:     true,
			DefaultCut:  "skinless",
			DefaultPrep: "cooked",
			Skinless: FoodCutProfile{
				EdibleFraction: &frac,
				Cooked:         &FoodNutritionPer100{KcalPer100g: &k},
			},
		},
	}
	it := MealItem{Grams: 270}
	got := nutritionGramsForItem(&it, &f)
	if got < 153.8 || got > 154.1 {
		t.Fatalf("nutrition grams %v want ~154", got)
	}
}