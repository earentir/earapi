package health

import "strings"

// EffectiveMealItemGrams returns grams for scaling: explicit grams, or count × unit default.
func EffectiveMealItemGrams(it *MealItem, foods []Food) float64 {
	if it == nil {
		return 0
	}
	if it.Grams > 0.001 {
		return it.Grams
	}
	if it.Count <= 0 {
		return 0
	}
	hint := strings.TrimSpace(it.Name + " " + it.Note)
	if g := itemCountGuessGrams(it.Count, hint); g > 0 {
		return g
	}
	_ = foods
	return 0
}

// FillMealItemGramsFromCount sets it.Grams from count when grams are missing.
func FillMealItemGramsFromCount(it *MealItem, foods []Food) {
	if it == nil {
		return
	}
	if g := EffectiveMealItemGrams(it, foods); g > 0 {
		it.Grams = g
	}
}
