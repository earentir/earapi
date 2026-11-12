package health

import "testing"

func TestFilterStockLotsForImport_dropsPurchasedKeepsSoup(t *testing.T) {
	lots := []StockLot{
		{ID: "y", Name: "Yogurt stock (Marata 1.7%)", BatchGrams: 800, RemainingGrams: 600},
		{ID: "s", Name: "soup-tomato mix", BatchGrams: 1900, RemainingGrams: 340},
		{Name: "Bean salad", BatchGrams: 730, Items: []MealItem{{Name: "beans", Grams: 500}}},
	}
	out := filterStockLotsForImport(lots)
	if len(out) != 2 {
		t.Fatalf("got %d lots %+v", len(out), out)
	}
	for _, lot := range out {
		if lot.ID == "y" {
			t.Fatal("yogurt should be dropped")
		}
	}
}
