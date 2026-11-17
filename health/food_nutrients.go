package health

import (
	"regexp"
	"strings"
)

var (
	noteKcalPer100gRe = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*kcal.+?per\s*100\s*g`)
	noteSodMgPer100gRe = regexp.MustCompile(`(?i)\(?~?(\d+(?:\.\d+)?)\s*mg\)?[^.]{0,24}per\s*100\s*g`)
	noteSaltGPer100gRe = regexp.MustCompile(`(?i)salt\s*(\d+(?:\.\d+)?)\s*g.+?per\s*100\s*g`)
)

// parseNutrientsFromNotes pulls per-100g numbers from markdown catalog notes (labels, bullets).
func parseNutrientsFromNotes(notes string) (kcal, sod *float64) {
	notes = strings.TrimSpace(notes)
	if notes == "" {
		return nil, nil
	}
	if m := noteKcalPer100gRe.FindStringSubmatch(notes); len(m) > 1 {
		v := parseFloatLoose(m[1])
		kcal = &v
	}
	if m := noteSodMgPer100gRe.FindStringSubmatch(notes); len(m) > 1 {
		v := parseFloatLoose(m[1])
		sod = &v
	} else if m := noteSaltGPer100gRe.FindStringSubmatch(notes); len(m) > 1 {
		v := SodiumMgFromSaltPer100g(parseFloatLoose(m[1]))
		sod = &v
	}
	return kcal, sod
}

func kcalPer100FromLabel(f *Food) *float64 {
	if f == nil || f.LabelGrams == nil || f.LabelKcal == nil {
		return nil
	}
	g, k := *f.LabelGrams, *f.LabelKcal
	if g <= 0 || k <= 0 {
		return nil
	}
	v := k / g * 100
	return &v
}

func sodPer100FromLabel(f *Food) *float64 {
	if f == nil || f.LabelGrams == nil || f.LabelSodiumMg == nil {
		return nil
	}
	g, s := *f.LabelGrams, *f.LabelSodiumMg
	if g <= 0 || s < 0 {
		return nil
	}
	v := s / g * 100
	return &v
}

func nutrientsForFood(f *Food) (kcal, sod *float64) {
	if f == nil {
		return nil, nil
	}
	if isFytroFood(f) {
		z := 0.0
		// ~120 mg Na per 1 g weighed on plate (see Food Data notes).
		sodPer100 := 12000.0
		return &z, &sodPer100
	}
	kcal, sod = f.KcalPer100g, f.SodiumMgPer100g
	if lk := kcalPer100FromLabel(f); lk != nil {
		kcal = lk
	}
	if ls := sodPer100FromLabel(f); ls != nil {
		sod = ls
	}
	nk, ns := parseNutrientsFromNotes(f.Notes)
	if kcal == nil {
		kcal = nk
	}
	if sod == nil {
		sod = ns
	}
	return kcal, sod
}

func cleanFoodDisplayName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimLeft(name, "*")
	name = strings.TrimRight(name, "*")
	return strings.TrimSpace(name)
}
