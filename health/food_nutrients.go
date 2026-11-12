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
		v := parseFloatLoose(m[1]) * 1000 // g salt → mg sodium (NaCl approx)
		sod = &v
	}
	return kcal, sod
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
