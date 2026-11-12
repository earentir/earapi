package health

import "testing"

func TestCanonicalMealFoodName_souvlaki(t *testing.T) {
	if got := canonicalMealFoodName("pork souvlaki"); got != "pork skewer" {
		t.Fatalf("got %q", got)
	}
	if got := canonicalMealFoodName("chicken souvlaki"); got != "chicken souvlaki" {
		t.Fatalf("got %q", got)
	}
	if got := canonicalMealFoodName("charcoal chicken skewers"); got != "chicken souvlaki" {
		t.Fatalf("got %q", got)
	}
	if got := canonicalMealFoodName("chicken skewer"); got != "chicken souvlaki" {
		t.Fatalf("chicken skewer got %q", got)
	}
	if got := canonicalMealFoodName("Bonora milk chocolate bar"); got != "Bonora milk chocolate bar" {
		t.Fatalf("got %q", got)
	}
	if got := canonicalMealFoodName("rusk no salt"); got != "rusks no salt" {
		t.Fatalf("got %q", got)
	}
}

func TestFoodForMealLine_porkSouvlaki(t *testing.T) {
	catalog := []Food{
		{ID: "pork-skewer", Name: "pork skewer", KcalPer100g: ptr(250.0)},
	}
	f, ok := foodForMealLine("pork souvlaki", catalog, nil)
	if !ok || f.ID != "pork-skewer" {
		t.Fatalf("got %+v ok=%v", f, ok)
	}
}

func ptr(v float64) *float64 { return &v }

func TestStalePer100KcalLine_reggiano(t *testing.T) {
	it := &MealItem{Grams: 10, Kcal: 380}
	if !stalePer100KcalLine(it, 380) {
		t.Fatal("expected stale")
	}
	it.Kcal = 38
	if stalePer100KcalLine(it, 380) {
		t.Fatal("expected fresh")
	}
}
