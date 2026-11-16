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

func TestDeleteFoodAlias(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	k := 30.0
	if _, err := st.AddFood(Food{Name: "Bell pepper", KcalPer100g: &k}); err != nil {
		t.Fatal(err)
	}
	foods := st.ListFoods()
	id := foods[0].ID
	if _, err := st.AddFoodAlias(id, "peppers"); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteFoodAlias(id, "peppers"); err != nil {
		t.Fatal(err)
	}
	if len(st.ListFoodAliasesForFood(id)) != 0 {
		t.Fatal("alias still listed")
	}
}

func TestFuzzyAliasResolvesMealLine(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	k := 30.0
	if _, err := st.AddFood(Food{Name: "Bell pepper", KcalPer100g: &k}); err != nil {
		t.Fatal(err)
	}
	foods := st.ListFoods()
	if _, err := st.AddFoodAlias(foods[0].ID, "hot horn pepper"); err != nil {
		t.Fatal(err)
	}
	it := MealItem{Name: "hot peppers", Grams: 10}
	if err := st.ScaleItemFromFood(&it); err != nil {
		t.Fatal(err)
	}
	if it.FoodID != foods[0].ID {
		t.Fatalf("food_id %q want bell pepper", it.FoodID)
	}
	if it.Kcal < 2.8 || it.Kcal > 3.2 {
		t.Fatalf("kcal %v want ~3", it.Kcal)
	}
}
