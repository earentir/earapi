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

func TestRankFoodsTypo(t *testing.T) {
	foods := []Food{{ID: "1", Name: "Vitalina FRUIT"}}
	ranked := RankFoods(foods, "vitalia", 5)
	if len(ranked) == 0 {
		t.Fatal("expected match")
	}
}
