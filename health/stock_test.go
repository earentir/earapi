package health

import "testing"

func TestStockDrawDecrementsRemaining(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	k, n := 23.0, 4.0
	lot, err := st.UpsertStockLot(StockLot{
		Name:            "Soup mix",
		BatchGrams:      1000,
		RemainingGrams:  1000,
		KcalPer100g:     &k,
		SodiumMgPer100g: &n,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, out, err := st.DrawFromStock(lot.ID, StockDraw{Date: "2026-11-12", Slot: "lunch", Grams: 400})
	if err != nil {
		t.Fatal(err)
	}
	if out == nil || out.RemainingGrams != 600 {
		t.Fatalf("remaining %v", out.RemainingGrams)
	}
	d, ok := st.GetDay("2026-11-12")
	if !ok {
		t.Fatal("day missing")
	}
	lunch := d.Meals["lunch"]
	if len(lunch.Items) != 1 || lunch.Items[0].Grams != 400 {
		t.Fatalf("lunch %+v", lunch.Items)
	}
	if lunch.Items[0].Kcal < 90 || lunch.Items[0].Kcal > 95 {
		t.Fatalf("kcal %v", lunch.Items[0].Kcal)
	}
}

func TestUpsertStockPreservesZeroRemaining(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	on := true
	lot, err := st.UpsertStockLot(StockLot{Name: "Done soup", BatchGrams: 500, RemainingGrams: 500, Active: &on})
	if err != nil {
		t.Fatal(err)
	}
	off := false
	_, err = st.UpsertStockLot(StockLot{ID: lot.ID, Name: lot.Name, BatchGrams: 500, RemainingGrams: 0, Active: &off})
	if err != nil {
		t.Fatal(err)
	}
	got, ok := st.StockLotByID(lot.ID)
	if !ok || got.RemainingGrams != 0 || stockLotIsActive(*got) {
		t.Fatalf("got %+v", got)
	}
}

func TestAdjustFromStockNoDay(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	lot, err := st.UpsertStockLot(StockLot{Name: "Salad", BatchGrams: 400, RemainingGrams: 400})
	if err != nil {
		t.Fatal(err)
	}
	out, err := st.AdjustFromStock(lot.ID, StockAdjust{Grams: 150, Mode: "household"})
	if err != nil {
		t.Fatal(err)
	}
	if out.RemainingGrams != 250 {
		t.Fatalf("remaining %v", out.RemainingGrams)
	}
	_, found := st.GetDay("2026-11-12")
	if found {
		t.Fatal("adjust must not create a day log")
	}
}
