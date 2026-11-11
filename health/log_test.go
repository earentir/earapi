package health

import "testing"

func TestApplyLogMealAndNewFood(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	k, n := 131.0, 1.0
	if _, err := st.AddFood(Food{Name: "Pasta (cooked, no salt)", KcalPer100g: &k, SodiumMgPer100g: &n}); err != nil {
		t.Fatal(err)
	}
	nk := 400.0
	out, err := st.ApplyLog(DayLog{
		Date: "2026-11-11",
		Meals: []mealWriteBody{
			{
				Slot: "lunch", Replace: true, KcalMargin: 0.10,
				Items: []MealItem{{Name: "Pasta (cooked, no salt)", Grams: 100}},
			},
			{
				Slot:  "snack",
				Items: []MealItem{{Name: "Energy bite", Grams: 50, KcalPer100g: &nk, SodiumMgPer100g: ptrF(80)}},
			},
		},
		BP: strPtr("120/80"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Day == nil {
		t.Fatal("missing day")
	}
	if out.Day.BP != "120/80" {
		t.Fatalf("bp %q", out.Day.BP)
	}
	lunch := out.Day.Meals["lunch"]
	if lunch.Kcal < 144-0.2 || lunch.Kcal > 144.2 {
		t.Fatalf("lunch kcal %v", lunch.Kcal)
	}
	snack := out.Day.Meals["snack"]
	if snack.Kcal != 200 || snack.SodiumMg != 40 {
		t.Fatalf("snack %+v", snack)
	}
	if st.FoodCount() != 2 {
		t.Fatalf("foods %d", st.FoodCount())
	}
}

func TestApplyLogDeletes(t *testing.T) {
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
	f, err := st.AddFood(Food{Name: "X", KcalPer100g: &k})
	if err != nil {
		t.Fatal(err)
	}
	w, err := st.AddWeight(WeighIn{Date: "2026-11-11", Kg: 80})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.ApplyLog(DayLog{
		Date:  "2026-11-11",
		Slot:  "lunch",
		Items: []MealItem{{Name: "X", Grams: 100}},
	}); err != nil {
		t.Fatal(err)
	}
	out, err := st.ApplyLog(DayLog{
		Date:          "2026-11-11",
		DeleteMeals:   []string{"lunch"},
		DeleteFoods:   []string{f.ID},
		DeleteWeights: []string{w.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out.Day.Meals["lunch"]; ok {
		t.Fatal("lunch should be gone")
	}
	if st.FoodCount() != 0 {
		t.Fatal("food remains")
	}
	if len(st.ListWeights("", "")) != 0 {
		t.Fatal("weight remains")
	}
}

func TestFoodsBatchAndWeightUpdate(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	k := 50.0
	got, err := st.EnsureFood(Food{Name: "Apple", KcalPer100g: &k})
	if err != nil {
		t.Fatal(err)
	}
	k2 := 52.0
	got, err = st.EnsureFood(Food{ID: got.ID, KcalPer100g: &k2})
	if err != nil {
		t.Fatal(err)
	}
	if got.KcalPer100g == nil || *got.KcalPer100g != 52 {
		t.Fatalf("%+v", got)
	}
	w, err := st.AddWeight(WeighIn{Date: "2026-11-01", Kg: 90, Note: "a"})
	if err != nil {
		t.Fatal(err)
	}
	upd, err := st.UpdateWeight(w.ID, WeighIn{Kg: 89.5, Note: "b", Protocol: true})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Kg != 89.5 || upd.Note != "b" || !upd.Protocol {
		t.Fatalf("%+v", upd)
	}
}

func strPtr(s string) *string { return &s }
