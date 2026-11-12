package health

import "testing"

func TestGapsAndFill(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	k := 70.0
	n := 70.0
	if _, err := st.AddFood(Food{Name: "Perch", KcalPer100g: &k, SodiumMgPer100g: &n}); err != nil {
		t.Fatal(err)
	}
	day := Day{
		Date: "2026-11-11",
		Meals: map[string]MealSlot{
			"lunch": {
				Items: []MealItem{
					{Name: "Perch", Grams: 400},
					{Name: "Unknown spice", Grams: 5},
				},
			},
		},
	}
	if err := st.PutDay(day); err != nil {
		t.Fatal(err)
	}
	gaps := st.ListNutritionGaps("2026-11-11", "2026-11-11", 50)
	if len(gaps.Items) != 1 || gaps.Items[0].Name != "Unknown spice" {
		t.Fatalf("items %+v", gaps.Items)
	}
	spK, spN := 10.0, 400.0
	out, err := st.ApplyNutritionFill(NutritionFill{
		Foods:         []Food{{Name: "Unknown spice", KcalPer100g: &spK, SodiumMgPer100g: &spN}},
		RecomputeDays: []string{"2026-11-11"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.FoodsEnsured) != 1 {
		t.Fatalf("%+v", out.FoodsEnsured)
	}
	d, ok := st.GetDay("2026-11-11")
	if !ok {
		t.Fatal("day missing")
	}
	lunch := d.Meals["lunch"]
	if lunch.Kcal <= 0 {
		t.Fatalf("lunch not recomputed %+v", lunch)
	}
	gaps2 := st.ListNutritionGaps("2026-11-11", "2026-11-11", 50)
	if len(gaps2.Items) != 0 {
		t.Fatalf("still gaps %+v", gaps2.Items)
	}
}
