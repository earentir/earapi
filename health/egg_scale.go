package health

import (
	"math"
	"strings"
)

func isEggFood(f *Food) bool {
	if f == nil {
		return false
	}
	return strings.Contains(strings.ToLower(f.Name), "egg")
}

func eggCountFromMealLine(name, note string) float64 {
	s := strings.TrimSpace(name + " " + note)
	if m := itemEggCountRe.FindStringSubmatch(s); len(m) > 1 {
		return parseFloatLoose(m[1])
	}
	return 0
}

// mealItemKcalFromFood scales catalog kcal. Egg rows are per one large egg (~55 g), not EU per 100 g.
func mealItemKcalFromFood(it *MealItem, f *Food, kcalPer100 *float64) float64 {
	if it == nil || f == nil || kcalPer100 == nil || it.Grams <= 0 {
		return 0
	}
	if isEggFood(f) {
		n := it.Count
		if n <= 0 {
			n = eggCountFromMealLine(it.Name, it.Note)
		}
		if n <= 0 {
			n = math.Round(it.Grams / 55)
		}
		if n < 1 {
			n = 1
		}
		return *kcalPer100 * n
	}
	return *kcalPer100 * it.Grams / 100
}
