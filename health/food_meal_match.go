package health

import "strings"

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
	return BestFoodMatch(catalog, name)
}

func isFytroFood(f *Food) bool {
	if f == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(f.Name), "fytro") ||
		strings.Contains(strings.ToLower(f.Name), "fytro")
}
