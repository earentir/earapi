package health

import "strings"

const (
	FoodKindPacked = "packed"
	FoodKindRaw    = "raw"
)

// NormalizeFoodKind returns packed, raw, or empty if unknown.
func NormalizeFoodKind(k string) string {
	switch strings.ToLower(strings.TrimSpace(k)) {
	case FoodKindRaw:
		return FoodKindRaw
	case FoodKindPacked:
		return FoodKindPacked
	default:
		return ""
	}
}

func foodKind(f *Food) string {
	if f == nil {
		return FoodKindPacked
	}
	if k := NormalizeFoodKind(f.Kind); k != "" {
		return k
	}
	return inferFoodKind(f)
}

func nutritionBlockHasData(n *FoodNutritionPer100) bool {
	if n == nil {
		return false
	}
	return n.KcalPer100g != nil || n.SodiumMgPer100g != nil || n.ProteinGPer100g != nil
}

// inferFoodKind treats leftover single-slot variants (old always-on form) as packed.
func inferFoodKind(f *Food) string {
	if f == nil {
		return FoodKindPacked
	}
	if f.EdibleFraction != nil {
		e := clampEdibleFraction(*f.EdibleFraction)
		if e > 0 && e < 1 {
			return FoodKindRaw
		}
	}
	if f.Variants == nil || !f.Variants.Enabled {
		return FoodKindPacked
	}
	v := f.Variants
	cut := strings.ToLower(strings.TrimSpace(v.DefaultCut))
	if cut == "with_skin" || cut == "withskin" || cut == "with-skin" {
		return FoodKindRaw
	}
	if nutritionBlockHasData(v.WithSkin.Raw) || nutritionBlockHasData(v.WithSkin.Cooked) {
		return FoodKindRaw
	}
	if v.WithSkin.EdibleFraction != nil || v.Skinless.EdibleFraction != nil {
		return FoodKindRaw
	}
	if nutritionBlockHasData(v.Skinless.Raw) && nutritionBlockHasData(v.Skinless.Cooked) {
		return FoodKindRaw
	}
	return FoodKindPacked
}

func setVariantNutrition(v *FoodVariants, cut, prep string, n *FoodNutritionPer100) {
	if v == nil || n == nil {
		return
	}
	var prof *FoodCutProfile
	switch strings.ToLower(strings.TrimSpace(cut)) {
	case "with_skin", "withskin", "with-skin":
		prof = &v.WithSkin
	default:
		prof = &v.Skinless
	}
	if strings.ToLower(strings.TrimSpace(prep)) == "cooked" {
		prof.Cooked = n
		return
	}
	prof.Raw = n
}

// applyPackedNutritionToDefaultVariant copies top-level macros into the default cut/prep only.
func applyPackedNutritionToDefaultVariant(f *Food) {
	if f == nil || f.Variants == nil || !f.Variants.Enabled {
		return
	}
	n := foodNutritionFromFood(f)
	if n == nil {
		return
	}
	cut := f.Variants.DefaultCut
	if cut == "" {
		cut = "skinless"
	}
	prep := f.Variants.DefaultPrep
	if prep == "" {
		prep = "raw"
	}
	setVariantNutrition(f.Variants, cut, prep, n)
}

func prepareFoodForWrite(f *Food) {
	if f == nil {
		return
	}
	f.Kind = foodKind(f)
	if f.Kind == FoodKindPacked {
		applyPackedNutritionToDefaultVariant(f)
		return
	}
	SyncFoodFromVariantDefault(f)
}
