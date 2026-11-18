package health

import "strings"

func variantCutProfile(v *FoodVariants, cut string) FoodCutProfile {
	if v == nil {
		return FoodCutProfile{}
	}
	switch strings.ToLower(strings.TrimSpace(cut)) {
	case "with_skin", "withskin", "with-skin":
		return v.WithSkin
	default:
		return v.Skinless
	}
}

func edibleFractionFromFood(f *Food) float64 {
	if f == nil {
		return 1
	}
	if f.Variants != nil && f.Variants.Enabled {
		cut := f.Variants.DefaultCut
		if cut == "" {
			cut = "skinless"
		}
		prof := variantCutProfile(f.Variants, cut)
		if prof.EdibleFraction != nil {
			return clampEdibleFraction(*prof.EdibleFraction)
		}
	}
	if f.EdibleFraction != nil {
		return clampEdibleFraction(*f.EdibleFraction)
	}
	return 1
}

func edibleFractionForItem(f *Food, it *MealItem) float64 {
	if it != nil && it.EdibleFraction != nil {
		return clampEdibleFraction(*it.EdibleFraction)
	}
	return edibleFractionFromFood(f)
}

func clampEdibleFraction(x float64) float64 {
	if x <= 0 || x > 1 {
		if x > 1 && x <= 100 {
			return x / 100
		}
		return 1
	}
	return x
}

// nutritionGramsForItem scales weighed grams by edible yield (per-100 g refs edible meat).
func nutritionGramsForItem(it *MealItem, f *Food) float64 {
	if it == nil || it.Grams <= 0 {
		return 0
	}
	return it.Grams * edibleFractionForItem(f, it)
}