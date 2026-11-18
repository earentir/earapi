package health

import "testing"

func TestSyncFoodFromVariantDefault(t *testing.T) {
	kcal := 165.0
	sod := 74.0
	f := &Food{
		Name: "Chicken breast",
		Variants: &FoodVariants{
			Enabled:     true,
			DefaultCut:  "skinless",
			DefaultPrep: "cooked",
			Skinless: FoodCutProfile{
				Cooked: &FoodNutritionPer100{KcalPer100g: &kcal, SodiumMgPer100g: &sod},
			},
		},
	}
	SyncFoodFromVariantDefault(f)
	if f.KcalPer100g == nil || *f.KcalPer100g != 165 {
		t.Fatalf("kcal: %+v", f.KcalPer100g)
	}
	if f.LabelGrams == nil || *f.LabelGrams != 100 {
		t.Fatalf("label g: %+v", f.LabelGrams)
	}
}