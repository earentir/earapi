package health

import (
	"math"
	"strings"
)

// BuildAliasFoodMap resolves mddata alias rows to catalog Food values for import scaling.
func BuildAliasFoodMap(specs []FoodAliasFromMD, foods []Food) map[string]Food {
	out := map[string]Food{}
	for _, sp := range specs {
		fid, ok := foodIDByDisplayName(foods, sp.CatalogFood)
		if !ok {
			continue
		}
		for _, f := range foods {
			if f.ID == fid {
				out[normalizeFoodAlias(sp.Alias)] = f
				break
			}
		}
	}
	return out
}

func foodForMealLine(name string, catalog []Food, byAlias map[string]Food) (*Food, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, false
	}
	if byAlias != nil {
		if f, ok := byAlias[normalizeFoodAlias(name)]; ok {
			ff := f
			return &ff, true
		}
	}
	name = canonicalMealFoodName(name)
	if f, ok := foodByExactName(catalog, name); ok {
		return f, true
	}
	return BestFoodMatch(catalog, name)
}

func foodByExactName(catalog []Food, name string) (*Food, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, false
	}
	for i := range catalog {
		if strings.EqualFold(catalog[i].Name, name) {
			f := catalog[i]
			return &f, true
		}
	}
	return nil, false
}

// stalePer100KcalLine reports DB rows where kcal equals per-100g on a small portion (Reggiano-style bug).
func stalePer100KcalLine(it *MealItem, kcalPer100 float64) bool {
	if it == nil || it.Grams <= 0 || it.Grams >= 100 || it.Kcal <= 0 {
		return false
	}
	expect := kcalPer100 * it.Grams / 100
	if math.Abs(it.Kcal-expect) < 2 {
		return false
	}
	return math.Abs(it.Kcal-kcalPer100) < 2
}


func isFytroFood(f *Food) bool {
	if f == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(f.Name), "fytro") ||
		strings.Contains(strings.ToLower(f.Name), "fytro")
}
