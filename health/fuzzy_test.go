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
	f, ok := BestFoodMatch(foods, "roasted chicken")
	if !ok || f.ID != "1" {
		t.Fatalf("only chicken cut in catalog should link: %v ok=%v", f, ok)
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

func TestBestFoodMatchEggAndPotatoShorthand(t *testing.T) {
	foods := []Food{
		{ID: "egg", Name: "Egg (1 large, ~55 g)"},
		{ID: "rp", Name: "Roasted potato"},
		{ID: "pp", Name: "Potato (boiled)"},
	}
	if f, ok := BestFoodMatch(foods, "boiled egg"); !ok || f.ID != "egg" {
		t.Fatalf("boiled egg: %v ok=%v", f, ok)
	}
	if f, ok := BestFoodMatch(foods, "roasted potatoes"); !ok || f.ID != "rp" {
		t.Fatalf("roasted potatoes: %v ok=%v", f, ok)
	}
	if f, ok := BestFoodMatch(foods, "eggs sunny side 3 eggs"); !ok || f.ID != "egg" {
		t.Fatalf("sunny eggs: %v ok=%v", f, ok)
	}
	for _, name := range []string{"Baby potatoes (steamed/boiled)", "Potato (boiled)", "Βλίτα / χόρτα (boiled)"} {
		if catalogMatchesMealLine(name, "boiled egg") {
			t.Fatalf("%q must not match boiled egg", name)
		}
	}
	pool := filterFoodsForMealLine([]Food{
		{ID: "egg", Name: "Egg (1 large, ~55 g)"},
		{ID: "pp", Name: "Potato (boiled)"},
	}, "boiled egg")
	if len(pool) != 1 || pool[0].ID != "egg" {
		t.Fatalf("filter boiled egg: %+v", pool)
	}
}

func TestRankFoodsTypo(t *testing.T) {
	foods := []Food{{ID: "1", Name: "Vitalina FRUIT"}}
	ranked := RankFoods(foods, "vitalia", 5)
	if len(ranked) == 0 {
		t.Fatal("expected match")
	}
}
