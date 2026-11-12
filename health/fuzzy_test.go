package health

import "testing"

func TestRankFoodsPasta(t *testing.T) {
	foods := []Food{
		{ID: "1", Name: "Potato (boiled)"},
		{ID: "2", Name: "Pasta (cooked, no salt)"},
		{ID: "3", Name: "White rice (cooked, no salt)"},
	}
	ranked := RankFoods(foods, "pasta", 5)
	if len(ranked) != 1 || ranked[0].Name != "Pasta (cooked, no salt)" {
		t.Fatalf("got %+v", ranked)
	}
}

func TestBestFoodMatchRoastedChickenSingleCookedOnly(t *testing.T) {
	foods := []Food{{ID: "1", Name: "Chicken breast (cooked, no salt)"}}
	_, ok := BestFoodMatch(foods, "roasted chicken")
	if ok {
		t.Fatal("cooked breast must not match roasted chicken shorthand")
	}
}

func TestBestFoodMatchRoastedChickenAmbiguousCooked(t *testing.T) {
	foods := []Food{
		{ID: "1", Name: "Chicken breast (cooked, no salt)"},
		{ID: "2", Name: "Chicken wings (cooked, no salt)"},
	}
	_, ok := BestFoodMatch(foods, "roasted chicken")
	if ok {
		t.Fatal("should not guess between cooked cuts when query says roasted")
	}
}

func TestBestFoodMatchRoastedChicken(t *testing.T) {
	foods := []Food{
		{ID: "1", Name: "Chicken breast (cooked, no salt)"},
		{ID: "2", Name: "Chicken thigh (roasted, air fryer)"},
	}
	f, ok := BestFoodMatch(foods, "roasted chicken")
	if !ok || f.ID != "2" {
		t.Fatalf("got %v ok=%v", f, ok)
	}
}

func TestRankFoodsTypo(t *testing.T) {
	foods := []Food{{ID: "1", Name: "Vitalina FRUIT"}}
	ranked := RankFoods(foods, "vitalia", 5)
	if len(ranked) == 0 {
		t.Fatal("expected match")
	}
}
