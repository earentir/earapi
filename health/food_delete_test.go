package health

import "testing"

func TestDeleteFoodWithReplacementRedirectsOldID(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	k := 200.0
	old, err := st.AddFood(Food{Name: "Old pasta", KcalPer100g: &k})
	if err != nil {
		t.Fatal(err)
	}
	rep, err := st.AddFood(Food{Name: "New pasta", KcalPer100g: &k})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddFoodAlias(old.ID, "pasta shorthand"); err != nil {
		t.Fatal(err)
	}
	it := MealItem{Name: "Old pasta", FoodID: old.ID, Grams: 100, Kcal: 200}
	if err := st.DeleteFood(old.ID, rep.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.ScaleItemFromFood(&it); err != nil {
		t.Fatal(err)
	}
	if it.FoodID != old.ID {
		t.Fatalf("meal line keeps stored food_id %q", it.FoodID)
	}
	if it.Kcal < 199 || it.Kcal > 201 {
		t.Fatalf("kcal after redirect %v", it.Kcal)
	}
	if _, ok := st.foodByAliasUnlocked(old.ID); !ok {
		t.Fatal("old id alias missing")
	}
	if f, ok := st.foodByAliasUnlocked("pasta shorthand"); !ok || f.ID != rep.ID {
		t.Fatalf("shorthand alias → %v ok=%v", f, ok)
	}
}

func TestDeleteFoodWithoutReplacementLeavesBrokenRef(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	k := 100.0
	old, err := st.AddFood(Food{Name: "Gone", KcalPer100g: &k})
	if err != nil {
		t.Fatal(err)
	}
	it := MealItem{Name: "Gone", FoodID: old.ID, Grams: 50, Kcal: 50}
	if err := st.DeleteFood(old.ID, ""); err != nil {
		t.Fatal(err)
	}
	if !itemBrokenFoodRef(it, st) {
		t.Fatal("expected broken food ref")
	}
}