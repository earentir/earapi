package health

import "testing"

func TestApplyLogMoveItemClearsSourceSlotRaw(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	date := "2026-11-25"
	dinnerRaw := "chicken 200g + rice 150g"
	if _, err := st.UpsertMealSlot(date, "dinner", MealSlot{
		Raw: dinnerRaw,
		Items: []MealItem{
			{Name: "chicken", Grams: 200, Kcal: 300},
			{Name: "rice", Grams: 150, Kcal: 180},
		},
	}, true); err != nil {
		t.Fatal(err)
	}
	out, err := st.ApplyLog(DayLog{
		Date: date,
		Meals: []mealWriteBody{
			{Slot: "lunch", Replace: true, Items: []MealItem{{Name: "chicken", Grams: 200, Kcal: 300}}},
			{Slot: "dinner", Replace: true, Items: nil},
			{Slot: "snack", Replace: true, Items: nil},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Day.Meals["lunch"].Items) != 1 || out.Day.Meals["lunch"].Items[0].Name != "chicken" {
		t.Fatalf("lunch %+v", out.Day.Meals["lunch"].Items)
	}
	if len(out.Day.Meals["dinner"].Items) != 0 {
		t.Fatalf("dinner items %+v", out.Day.Meals["dinner"].Items)
	}
	if raw := out.Day.Meals["dinner"].Raw; raw != "" {
		t.Fatalf("dinner raw %q should be cleared", raw)
	}
	got, ok := st.GetDay(date)
	if !ok {
		t.Fatal("missing day")
	}
	if len(got.Meals["dinner"].Items) != 0 {
		t.Fatalf("after get dinner items %+v", got.Meals["dinner"].Items)
	}
	if len(got.Meals["lunch"].Items) != 1 {
		t.Fatalf("after get lunch %+v", got.Meals["lunch"].Items)
	}
}