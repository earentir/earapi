package health

import "testing"

func TestIsEggFood_notReggiano(t *testing.T) {
	k := 380.0
	f := &Food{Name: "Parmigiano Reggiano", KcalPer100g: &k}
	if isEggFood(f) {
		t.Fatal("Reggiano must not match egg food")
	}
}

func TestScaleParmigiano7g(t *testing.T) {
	k := 380.0
	n := 800.0
	f := Food{ID: "parmigiano-reggiano", Name: "Parmigiano Reggiano", KcalPer100g: &k, SodiumMgPer100g: &n}
	it := MealItem{Name: "Parmigiano Reggiano", Grams: 7, Note: "Parmigiano Reggiano 7 g"}
	if err := scaleItemFromFoodWithCatalog(&it, []Food{f}, nil); err != nil {
		t.Fatal(err)
	}
	if it.Kcal < 25 || it.Kcal > 28 {
		t.Fatalf("kcal %v want ~26.6", it.Kcal)
	}
}

func TestMealItemKcalFromFood_threeEggs(t *testing.T) {
	k := 78.0
	f := &Food{Name: "Egg (1 large, ~55 g)", KcalPer100g: &k}
	it := &MealItem{Name: "eggs sunny side 3 eggs", Grams: 165, Note: "eggs sunny side 3 eggs"}
	got := mealItemKcalFromFood(it, f, &k)
	if got < 230 || got > 240 {
		t.Fatalf("kcal %v want ~234", got)
	}
}
