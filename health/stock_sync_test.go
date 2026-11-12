package health

import "testing"

func TestMealSaveDeductsStockByLotID(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	lot, err := st.UpsertStockLot(StockLot{Name: "Soup-tomato mix (carcass #3)", BatchGrams: 1900, RemainingGrams: 700})
	if err != nil {
		t.Fatal(err)
	}
	meal := MealSlot{
		Items: []MealItem{
			{Name: "Soup-tomato mix (carcass #3)", StockLotID: lot.ID, Grams: 400},
		},
	}
	if _, err := st.PrepareMealSlot(&meal, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpsertMealSlot("2026-11-08", "dinner", meal, true); err != nil {
		t.Fatal(err)
	}
	got, ok := st.StockLotByID(lot.ID)
	if !ok || got.RemainingGrams != 300 {
		t.Fatalf("remaining %v", got.RemainingGrams)
	}
	if stockLotIsActive(*got) == false {
		t.Fatal("expected still active")
	}
	if len(got.Usage) == 0 {
		t.Fatal("expected usage ledger line")
	}
}

func TestStockInactiveWhenEmpty(t *testing.T) {
	on := true
	lot := StockLot{Name: "X", BatchGrams: 100, RemainingGrams: 100, Active: &on}
	stockLotSetActive(&lot, true)
	lot.RemainingGrams = 0
	if stockLotIsActive(lot) {
		t.Fatal("zero remaining must not be active")
	}
}
