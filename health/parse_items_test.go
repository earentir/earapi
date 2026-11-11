package health

import "testing"

func TestParseMealItemsFromRawDinner(t *testing.T) {
	raw := `Soup-tomato mix 400 g ~92 / ~112 + moussaka 200 g ~440 / ~390 + lettuce 100 g ~25 / ~20 + EVOO 2 g 0 / ~18 + balsamic cream 4 g ~16 / ~14 → **~573 / ~554 → +10% margin = ~609**`
	items := ParseMealItemsFromRaw(raw)
	if len(items) != 5 {
		t.Fatalf("got %d items %+v", len(items), items)
	}
	if items[0].Name != "Soup-tomato mix" || items[0].Grams != 400 {
		t.Fatalf("first %+v", items[0])
	}
	if items[1].Name != "moussaka" || items[1].Grams != 200 {
		t.Fatalf("moussaka %+v", items[1])
	}
	if items[3].Grams != 2 {
		t.Fatalf("evoo %+v", items[3])
	}
}

func TestParseMealItemsTimesAndParen(t *testing.T) {
	items := ParseMealItemsFromRaw(`Granola bars ×2 (42 g, label) ~104 / ~200 + rusks no salt 12 g ×3`)
	if len(items) != 2 {
		t.Fatalf("%+v", items)
	}
	if items[0].Grams != 42 {
		t.Fatalf("bars grams %v", items[0].Grams)
	}
	if items[1].Grams != 36 {
		t.Fatalf("rusks grams %v want 36", items[1].Grams)
	}
}

func TestScaleDish(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	k, n := 100.0, 200.0
	if _, err := st.AddFood(Food{Name: "Oil", KcalPer100g: &k, SodiumMgPer100g: &n}); err != nil {
		t.Fatal(err)
	}
	k2, n2 := 50.0, 10.0
	if _, err := st.AddFood(Food{Name: "Leaves", KcalPer100g: &k2, SodiumMgPer100g: &n2}); err != nil {
		t.Fatal(err)
	}
	meal := MealSlot{
		Items: []MealItem{{
			Name:  "Salad",
			Grams: 150,
			Parts: []MealItem{
				{Name: "Leaves", Grams: 200},
				{Name: "Oil", Grams: 10},
			},
		}},
	}
	if err := st.PrepareMealSlot(&meal, nil); err != nil {
		t.Fatal(err)
	}
	// batch 210g, eaten 150g; leaves 100 kcal, oil 10 kcal -> 110 * 150/210
	want := 110.0 * 150 / 210
	if meal.Items[0].Kcal < want-0.05 || meal.Items[0].Kcal > want+0.05 {
		t.Fatalf("kcal %v want %v", meal.Items[0].Kcal, want)
	}
}
