package health

import "testing"

func TestParseEggsSunnySide3Eggs(t *testing.T) {
	it, ok := parseOneMealChunk("eggs sunny side 3 eggs 165 g")
	if !ok {
		t.Fatal("parse fail")
	}
	if it.Count != 3 {
		t.Fatalf("count=%v want 3", it.Count)
	}
	if it.Grams != 165 {
		t.Fatalf("grams=%v want 165", it.Grams)
	}
}

func TestParseOmelet2Eggs(t *testing.T) {
	it, ok := parseOneMealChunk("omelet 2 eggs 110 g")
	if !ok {
		t.Fatal("parse fail")
	}
	if it.Count != 2 {
		t.Fatalf("count=%v want 2", it.Count)
	}
	if it.Grams != 110 {
		t.Fatalf("grams=%v want 110", it.Grams)
	}
}

func TestParseOmelet2EggsNoGrams(t *testing.T) {
	it, ok := parseOneMealChunk("omelet 2 eggs")
	if !ok {
		t.Fatal("parse fail")
	}
	if it.Count != 2 {
		t.Fatalf("count=%v want 2", it.Count)
	}
	if it.Grams != 110 {
		t.Fatalf("grams=%v want 110 (2×55)", it.Grams)
	}
}

func TestParseRusksPerUnitTimes3(t *testing.T) {
	it, ok := parseOneMealChunk("Rusks no salt 12 g ×3")
	if !ok {
		t.Fatal("parse fail")
	}
	if it.Count != 3 {
		t.Fatalf("count=%v want 3", it.Count)
	}
	if it.Grams != 36 {
		t.Fatalf("grams=%v want 36", it.Grams)
	}
}

func TestMealItemKcalFromFood_usesCountField(t *testing.T) {
	k := 78.0
	f := &Food{Name: "Egg (1 large, ~55 g)", KcalPer100g: &k}
	it := &MealItem{Name: "eggs sunny side", Grams: 165, Count: 3, Note: "eggs sunny side 3 eggs 165 g"}
	got := mealItemKcalFromFood(it, f, &k)
	if got < 230 || got > 240 {
		t.Fatalf("kcal %v want ~234", got)
	}
}
