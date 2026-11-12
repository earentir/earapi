package health

import (
	"testing"
	"time"
)

func TestScaleMealItemFromExplicitStockLotID(t *testing.T) {
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
	it := MealItem{Name: "soup-tomato mix", Grams: 360, StockLotID: lot.ID}
	if err := st.ScaleItemFromFood(&it); err != nil {
		t.Fatal(err)
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

func TestScaleStockLotSelfReferencingPartsNoOverflow(t *testing.T) {
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
	n := 100.0
	lot, err := st.UpsertStockLot(StockLot{
		Name:             "Soup-tomato mix (carcass #3)",
		BatchGrams:       500,
		RemainingGrams:   500,
		KcalPer100g:      &k,
		SodiumMgPer100g:  &n,
	})
	if err != nil {
		t.Fatal(err)
	}
	lot.Items = []MealItem{{Name: "soup-tomato mix", StockLotID: lot.ID, Grams: 500}}
	lot, err = st.UpsertStockLot(lot)
	if err != nil {
		t.Fatal(err)
	}
	it := MealItem{Name: "soup-tomato", StockLotID: lot.ID, Grams: 200}
	if err := st.ScaleItemFromFood(&it); err != nil {
		t.Fatal(err)
	}
	if it.Kcal < 95 || it.Kcal > 105 {
		t.Fatalf("kcal %v want ~100", it.Kcal)
	}
}

func TestUpsertStockLotSelfReferencingItemsNoHang(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	k := 10.0
	n := 20.0
	lot, err := st.UpsertStockLot(StockLot{
		Name: "Soup-tomato mix", BatchGrams: 500, RemainingGrams: 500,
		KcalPer100g: &k, SodiumMgPer100g: &n,
	})
	if err != nil {
		t.Fatal(err)
	}
	lot.Items = []MealItem{{Name: "soup-tomato mix", StockLotID: lot.ID, Grams: 500}}
	done := make(chan error, 1)
	go func() { _, err := st.UpsertStockLot(lot); done <- err }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("UpsertStockLot hung refreshing batch nutrients")
	}
}

func TestGetDayWithSelfReferencingStockLotNoOverflow(t *testing.T) {
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
	n := 100.0
	lot, err := st.UpsertStockLot(StockLot{
		Name: "Soup-tomato mix (carcass #3)", BatchGrams: 500, RemainingGrams: 500,
		KcalPer100g: &k, SodiumMgPer100g: &n,
	})
	if err != nil {
		t.Fatal(err)
	}
	lot.Items = []MealItem{{Name: "soup-tomato mix", StockLotID: lot.ID, Grams: 500}}
	lot, err = st.UpsertStockLot(lot)
	if err != nil {
		t.Fatal(err)
	}
	meal := MealSlot{
		Items: []MealItem{{Name: "soup-tomato", StockLotID: lot.ID, Grams: 200}},
	}
	if _, err := st.UpsertMealSlot("2026-11-12", "dinner", meal, true); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		_, ok := st.GetDay("2026-11-12")
		if !ok {
			t.Error("expected day")
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("GetDay hung (stock scale recursion?)")
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
