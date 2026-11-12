package health

import "testing"

func TestFoodAliasResolvesMealLine(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	k := 190.0
	if _, err := st.AddFood(Food{Name: "Chicken thigh (roasted, air fryer)", KcalPer100g: &k}); err != nil {
		t.Fatal(err)
	}
	foods := st.ListFoods()
	if _, err := st.AddFoodAlias(foods[0].ID, "roasted chicken"); err != nil {
		t.Fatal(err)
	}
	it := MealItem{Name: "roasted chicken", Grams: 50}
	if err := st.ScaleItemFromFood(&it); err != nil {
		t.Fatal(err)
	}
	if it.FoodID != foods[0].ID {
		t.Fatalf("food_id %q", it.FoodID)
	}
	if it.Kcal < 94 || it.Kcal > 96 {
		t.Fatalf("kcal %v", it.Kcal)
	}
}
