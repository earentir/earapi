package health

import (
	"encoding/json"
	"strings"
)

// FoodNutritionPer100 is one nutrition profile stored per 100 g.
type FoodNutritionPer100 struct {
	KcalPer100g       *float64 `json:"kcal_per_100g,omitempty"`
	SodiumMgPer100g   *float64 `json:"sodium_mg_per_100g,omitempty"`
	EnergyKjPer100g   *float64 `json:"energy_kj_per_100g,omitempty"`
	FatGPer100g       *float64 `json:"fat_g_per_100g,omitempty"`
	SaturatesGPer100g *float64 `json:"saturates_g_per_100g,omitempty"`
	CarbsGPer100g     *float64 `json:"carbs_g_per_100g,omitempty"`
	SugarsGPer100g    *float64 `json:"sugars_g_per_100g,omitempty"`
	ProteinGPer100g   *float64 `json:"protein_g_per_100g,omitempty"`
	SaltGPer100g      *float64 `json:"salt_g_per_100g,omitempty"`
}

// FoodCutProfile holds raw and cooked numbers for one cut (skinless / with skin).
type FoodCutProfile struct {
	// EdibleFraction is the share of as-weighed grams that count toward per-100 g nutrition (bones, seeds, peel).
	EdibleFraction *float64             `json:"edible_fraction,omitempty"`
	Raw            *FoodNutritionPer100 `json:"raw,omitempty"`
	Cooked         *FoodNutritionPer100 `json:"cooked,omitempty"`
}

// FoodVariants groups cut profiles; default_cut + default_prep drive top-level catalog fields.
type FoodVariants struct {
	Enabled     bool           `json:"enabled,omitempty"`
	DefaultCut  string         `json:"default_cut,omitempty"`  // skinless | with_skin
	DefaultPrep string         `json:"default_prep,omitempty"` // raw | cooked
	Skinless    FoodCutProfile `json:"skinless,omitempty"`
	WithSkin    FoodCutProfile `json:"with_skin,omitempty"`
}

func foodNutritionFromFood(f *Food) *FoodNutritionPer100 {
	if f == nil {
		return nil
	}
	n := &FoodNutritionPer100{
		KcalPer100g:       f.KcalPer100g,
		SodiumMgPer100g:   f.SodiumMgPer100g,
		EnergyKjPer100g:   f.EnergyKjPer100g,
		FatGPer100g:       f.FatGPer100g,
		SaturatesGPer100g: f.SaturatesGPer100g,
		CarbsGPer100g:     f.CarbsGPer100g,
		SugarsGPer100g:    f.SugarsGPer100g,
		ProteinGPer100g:   f.ProteinGPer100g,
		SaltGPer100g:      f.SaltGPer100g,
	}
	if n.KcalPer100g == nil && n.SodiumMgPer100g == nil && n.ProteinGPer100g == nil {
		return nil
	}
	return n
}

func foodLooksCooked(f *Food) bool {
	if f == nil {
		return false
	}
	s := strings.ToLower(f.Name + " " + f.Notes)
	return strings.Contains(s, "cooked") || strings.Contains(s, "grilled") || strings.Contains(s, "roasted")
}

// SeedFoodVariants builds variants from legacy single-profile foods (skinless + guessed prep).
func SeedFoodVariants(f *Food) *FoodVariants {
	if f == nil {
		return nil
	}
	prep := "raw"
	if foodLooksCooked(f) {
		prep = "cooked"
	}
	block := foodNutritionFromFood(f)
	v := &FoodVariants{
		Enabled:     block != nil,
		DefaultCut:  "skinless",
		DefaultPrep: prep,
	}
	if block != nil {
		if prep == "cooked" {
			v.Skinless.Cooked = block
		} else {
			v.Skinless.Raw = block
		}
	}
	return v
}

func variantNutrition(v *FoodVariants, cut, prep string) *FoodNutritionPer100 {
	if v == nil {
		return nil
	}
	var prof FoodCutProfile
	switch strings.ToLower(strings.TrimSpace(cut)) {
	case "with_skin", "withskin", "with-skin":
		prof = v.WithSkin
	default:
		prof = v.Skinless
	}
	switch strings.ToLower(strings.TrimSpace(prep)) {
	case "cooked":
		return prof.Cooked
	default:
		return prof.Raw
	}
}

// SyncFoodFromVariantDefault copies the chosen default profile onto top-level Food (per 100 g + label ref 100 g).
func SyncFoodFromVariantDefault(f *Food) {
	if f == nil || f.Variants == nil || !f.Variants.Enabled {
		return
	}
	cut := f.Variants.DefaultCut
	prep := f.Variants.DefaultPrep
	if cut == "" {
		cut = "skinless"
	}
	if prep == "" {
		prep = "cooked"
	}
	n := variantNutrition(f.Variants, cut, prep)
	if n == nil {
		return
	}
	f.KcalPer100g = n.KcalPer100g
	f.SodiumMgPer100g = n.SodiumMgPer100g
	f.EnergyKjPer100g = n.EnergyKjPer100g
	f.FatGPer100g = n.FatGPer100g
	f.SaturatesGPer100g = n.SaturatesGPer100g
	f.CarbsGPer100g = n.CarbsGPer100g
	f.SugarsGPer100g = n.SugarsGPer100g
	f.ProteinGPer100g = n.ProteinGPer100g
	f.SaltGPer100g = n.SaltGPer100g
	g := 100.0
	f.LabelGrams = &g
	if n.KcalPer100g != nil {
		k := *n.KcalPer100g
		f.LabelKcal = &k
	}
	if n.SodiumMgPer100g != nil {
		s := *n.SodiumMgPer100g
		f.LabelSodiumMg = &s
	}
	NormalizeFoodLabel(f)
}

func encodeFoodVariantsJSON(v *FoodVariants) string {
	if v == nil || !v.Enabled {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func decodeFoodVariantsJSON(raw string) *FoodVariants {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var v FoodVariants
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return nil
	}
	if !v.Enabled {
		return nil
	}
	return &v
}