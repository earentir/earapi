package health

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestLegacyDiffBandZeroIsGreen(t *testing.T) {
	s := DefaultKcalBudgetSettings()
	if legacyDiffBand(1091, 1091, s) != "green" {
		t.Fatalf("want green for exact match")
	}
}

func TestNutritionComputeDayHTTP(t *testing.T) {
	r, svc := testServer(t)
	st, err := svc.StoreForUser("u")
	if err != nil {
		t.Fatal(err)
	}
	k := 155.0
	sod := 1.0
	if _, _, err := st.EnsureFood(Food{ID: "pasta", Name: "pasta", KcalPer100g: &k, SodiumMgPer100g: &sod}); err != nil {
		t.Fatal(err)
	}
	body := Day{
		Date: "2026-11-14",
		Meals: map[string]MealSlot{
			"lunch": {Items: []MealItem{{Name: "pasta", FoodID: "pasta", Grams: 100}}},
		},
	}
	w := doJSON(t, r, http.MethodPost, "/health/v1/nutrition/compute/day", body)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d %s", w.Code, w.Body.String())
	}
	var out ComputeDayResult
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Day.Totals.Kcal <= 0 {
		t.Fatalf("kcal %v", out.Day.Totals.Kcal)
	}
	if out.Display.IntakeBarBand == "none" {
		t.Fatalf("intake band")
	}
}
