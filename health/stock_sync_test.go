package health

import "testing"

func TestScaleMealItemFromStockByShortName(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	k := 23.0
	n := 28.0
	lot, err := st.UpsertStockLot(StockLot{
		Name: "Soup-tomato mix (carcass #3)", BatchGrams: 1900, RemainingGrams: 700,
		KcalPer100g: &k, SodiumMgPer100g: &n,
	})
	if err != nil {
		t.Fatal(err)
	}
	it := MealItem{Name: "soup-tomato mix", Grams: 360}
	if err := st.ScaleItemFromFood(&it); err != nil {
		t.Fatal(err)
	}
	if it.StockLotID != lot.ID {
		t.Fatalf("lot id %q want %q", it.StockLotID, lot.ID)
	}
	if it.Kcal < 80 || it.Kcal > 85 {
		t.Fatalf("kcal %v", it.Kcal)
	}
}

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
