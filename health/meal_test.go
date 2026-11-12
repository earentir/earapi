package health

import "testing"

func TestFinalizeMealTotalsSumsItems(t *testing.T) {
	m := MealSlot{
		Items: []MealItem{
			{Kcal: 100, SodiumMg: 50},
			{Kcal: 200, SodiumMg: 30},
		},
	}
	FinalizeMealTotals(&m)
	if m.SodiumMg != 80 {
		t.Fatalf("sodium %v", m.SodiumMg)
	}
	if m.Kcal != 300 {
		t.Fatalf("kcal %v", m.Kcal)
	}
}

func TestPrepareMealSlotCatalog(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u1")
	if err != nil {
		t.Fatal(err)
	}
	k, s := 131.0, 1.0
	if _, err := st.AddFood(Food{ID: "pasta", Name: "Pasta (cooked, no salt)", KcalPer100g: &k, SodiumMgPer100g: &s}); err != nil {
		t.Fatal(err)
	}
	meal := MealSlot{
		Items: []MealItem{{FoodID: "pasta", Grams: 100}},
	}
	if _, err := st.PrepareMealSlot(&meal, nil); err != nil {
		t.Fatal(err)
	}
	if meal.Kcal < 130.99 || meal.Kcal > 131.01 {
		t.Fatalf("kcal %v", meal.Kcal)
	}
	if meal.SodiumMg != 1 {
		t.Fatalf("sod %v", meal.SodiumMg)
	}
}

func TestEnsureFoodAndLogNewItem(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u2")
	if err != nil {
		t.Fatal(err)
	}
	k := 400.0
	meal := MealSlot{
		Items: []MealItem{{Name: "Energy bite", Grams: 50, KcalPer100g: &k, SodiumMgPer100g: ptrF(80)}},
	}
	if _, err := st.PrepareMealSlot(&meal, nil); err != nil {
		t.Fatal(err)
	}
	if meal.Kcal != 200 {
		t.Fatalf("kcal %v", meal.Kcal)
	}
	if meal.SodiumMg != 40 {
		t.Fatalf("sod %v", meal.SodiumMg)
	}
	if foods := st.ListFoods(); len(foods) != 1 {
		t.Fatalf("foods %d", len(foods))
	}
}
