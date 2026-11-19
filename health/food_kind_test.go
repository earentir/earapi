package health

import "testing"

func TestInferFoodKindPackedSingleSlot(t *testing.T) {
	k := 130.0
	f := &Food{
		Variants: &FoodVariants{
			Enabled:     true,
			DefaultCut:  "skinless",
			DefaultPrep: "raw",
			Skinless:    FoodCutProfile{Raw: &FoodNutritionPer100{KcalPer100g: &k}},
		},
	}
	if got := inferFoodKind(f); got != FoodKindPacked {
		t.Fatalf("kind %q want packed", got)
	}
}

func TestInferFoodKindFromTopLevelYield(t *testing.T) {
	frac := 0.7
	f := &Food{EdibleFraction: &frac}
	if got := inferFoodKind(f); got != FoodKindRaw {
		t.Fatalf("kind %q want raw", got)
	}
}

func TestInferFoodKindRawDualPrep(t *testing.T) {
	raw, cooked := 120.0, 165.0
	f := &Food{
		Variants: &FoodVariants{
			Enabled:     true,
			DefaultCut:  "skinless",
			DefaultPrep: "cooked",
			Skinless: FoodCutProfile{
				Raw:    &FoodNutritionPer100{KcalPer100g: &raw},
				Cooked: &FoodNutritionPer100{KcalPer100g: &cooked},
			},
		},
	}
	if got := inferFoodKind(f); got != FoodKindRaw {
		t.Fatalf("kind %q want raw", got)
	}
}

func TestPackedIgnoresCatalogEdibleFraction(t *testing.T) {
	k := 200.0
	frac := 0.7
	f := Food{Kind: FoodKindPacked, KcalPer100g: &k, EdibleFraction: &frac}
	it := MealItem{Grams: 100}
	got := nutritionGramsForItem(&it, &f)
	if got < 99.9 || got > 100.1 {
		t.Fatalf("nutrition grams %v want 100", got)
	}
}

func TestPackedLineEdibleOverride(t *testing.T) {
	k := 200.0
	frac := 0.5
	f := Food{Kind: FoodKindPacked, KcalPer100g: &k}
	it := MealItem{Grams: 100, EdibleFraction: &frac}
	got := nutritionGramsForItem(&it, &f)
	if got < 49.9 || got > 50.1 {
		t.Fatalf("nutrition grams %v want 50", got)
	}
}

func TestPackedPatchKeepsVariantsAndUpdatesDefaultSlot(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	raw, cooked, extra := 120.0, 165.0, 197.0
	f, err := st.AddFood(Food{
		Name:        "Chicken",
		Kind:        FoodKindRaw,
		KcalPer100g: &raw,
		Variants: &FoodVariants{
			Enabled:     true,
			DefaultCut:  "skinless",
			DefaultPrep: "raw",
			Skinless: FoodCutProfile{
				Raw:    &FoodNutritionPer100{KcalPer100g: &raw},
				Cooked: &FoodNutritionPer100{KcalPer100g: &cooked},
			},
			WithSkin: FoodCutProfile{Cooked: &FoodNutritionPer100{KcalPer100g: &extra}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	packedK := 130.0
	labelG := 150.0
	got, err := st.UpdateFood(f.ID, Food{Kind: FoodKindPacked, KcalPer100g: &packedK, LabelGrams: &labelG})
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != FoodKindPacked {
		t.Fatalf("kind %q", got.Kind)
	}
	if got.LabelGrams == nil || *got.LabelGrams != 150 {
		t.Fatalf("packed pack grams %+v", got.LabelGrams)
	}
	if got.Variants == nil || !got.Variants.Enabled {
		t.Fatal("variants JSON dropped")
	}
	if got.Variants.Skinless.Raw == nil || got.Variants.Skinless.Raw.KcalPer100g == nil || *got.Variants.Skinless.Raw.KcalPer100g != 130 {
		t.Fatalf("default slot %+v", got.Variants.Skinless.Raw)
	}
	if got.Variants.Skinless.Cooked == nil || *got.Variants.Skinless.Cooked.KcalPer100g != 165 {
		t.Fatalf("cooked slot lost %+v", got.Variants.Skinless.Cooked)
	}
	if got.Variants.WithSkin.Cooked == nil || *got.Variants.WithSkin.Cooked.KcalPer100g != 197 {
		t.Fatalf("with_skin lost %+v", got.Variants.WithSkin.Cooked)
	}
	reload, ok := st.FoodByID(f.ID)
	if !ok {
		t.Fatal("reload")
	}
	if reload.Variants == nil || reload.Variants.WithSkin.Cooked == nil || *reload.Variants.WithSkin.Cooked.KcalPer100g != 197 {
		t.Fatalf("persisted variants %+v", reload.Variants)
	}
}

func TestNewFoodDefaultsPacked(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	k := 90.0
	f, err := st.AddFood(Food{Name: "Yogurt", KcalPer100g: &k})
	if err != nil {
		t.Fatal(err)
	}
	if f.Kind != FoodKindPacked {
		t.Fatalf("kind %q want packed", f.Kind)
	}
}
