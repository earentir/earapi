package health

import (
	"regexp"
	"strings"
)

// EU mandatory nutrition declaration per 100 g / 100 ml (Reg. 1169/2011).
var (
	reLabelKcal = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*kcal`)
	reLabelFat = regexp.MustCompile(`(?i)\bfat\s*(\d+(?:\.\d+)?)\s*g`)
	reLabelSat = regexp.MustCompile(`(?i)(?:sat(?:urates)?|of which saturates)\s*(\d+(?:\.\d+)?)\s*g`)
	reLabelCarb = regexp.MustCompile(`(?i)\bcarbs?\s*(\d+(?:\.\d+)?)\s*g`)
	reLabelSugar = regexp.MustCompile(`(?i)(?:sugars?|of which sugars)\s*(\d+(?:\.\d+)?)\s*g`)
	reLabelProtein = regexp.MustCompile(`(?i)\bprotein\s*(\d+(?:\.\d+)?)\s*g`)
	reLabelSalt = regexp.MustCompile(`(?i)\bsalt\s*(\d+(?:\.\d+)?)\s*g`)
	rePer100 = regexp.MustCompile(`(?i)per\s*100\s*g`)
)

// ParseEULabelFromText fills per-100g label fields from free text (notes, catalog bullets).
func ParseEULabelFromText(text string) Food {
	var f Food
	text = strings.TrimSpace(text)
	if text == "" {
		return f
	}
	if m := reLabelKcal.FindStringSubmatch(text); len(m) > 1 {
		v := parseFloatLoose(m[1])
		f.KcalPer100g = &v
	}
	if m := reLabelFat.FindStringSubmatch(text); len(m) > 1 {
		v := parseFloatLoose(m[1])
		f.FatGPer100g = &v
	}
	if m := reLabelSat.FindStringSubmatch(text); len(m) > 1 {
		v := parseFloatLoose(m[1])
		f.SaturatesGPer100g = &v
	}
	if m := reLabelCarb.FindStringSubmatch(text); len(m) > 1 {
		v := parseFloatLoose(m[1])
		f.CarbsGPer100g = &v
	}
	if m := reLabelSugar.FindStringSubmatch(text); len(m) > 1 {
		v := parseFloatLoose(m[1])
		f.SugarsGPer100g = &v
	}
	if m := reLabelProtein.FindStringSubmatch(text); len(m) > 1 {
		v := parseFloatLoose(m[1])
		f.ProteinGPer100g = &v
	}
	if m := reLabelSalt.FindStringSubmatch(text); len(m) > 1 {
		v := parseFloatLoose(m[1])
		f.SaltGPer100g = &v
	}
	if f.SaltGPer100g == nil {
		nk, ns := parseNutrientsFromNotes(text)
		if f.KcalPer100g == nil {
			f.KcalPer100g = nk
		}
		if ns != nil {
			salt := *ns / 400
			f.SaltGPer100g = &salt
		}
	} else if f.KcalPer100g == nil {
		nk, _ := parseNutrientsFromNotes(text)
		f.KcalPer100g = nk
	}
	if f.SaltGPer100g != nil && f.SodiumMgPer100g == nil {
		sod := *f.SaltGPer100g * 400
		f.SodiumMgPer100g = &sod
	}
	return f
}

// NormalizeFoodLabel syncs kJ, salt ↔ sodium from kcal/salt fields.
func NormalizeFoodLabel(f *Food) {
	if f == nil {
		return
	}
	if f.KcalPer100g != nil && f.EnergyKjPer100g == nil {
		kj := *f.KcalPer100g * 4.184
		f.EnergyKjPer100g = &kj
	}
	if f.SaltGPer100g != nil && f.SodiumMgPer100g == nil {
		sod := *f.SaltGPer100g * 400
		f.SodiumMgPer100g = &sod
	} else if f.SodiumMgPer100g != nil && f.SaltGPer100g == nil {
		salt := *f.SodiumMgPer100g / 400
		f.SaltGPer100g = &salt
	}
}

// ApplyLabelFromNotes parses label numbers from notes, moves non-label text to residual notes.
func ApplyLabelFromNotes(f *Food) {
	if f == nil {
		return
	}
	NormalizeFoodLabel(f)
	notes := strings.TrimSpace(f.Notes)
	if notes == "" {
		return
	}
	if !reLabelKcal.MatchString(notes) && !rePer100.MatchString(notes) && !reLabelFat.MatchString(notes) {
		return
	}
	parsed := ParseEULabelFromText(notes)
	mergeFoodLabel(f, parsed)
	f.Notes = residualFoodNotes(notes)
	NormalizeFoodLabel(f)
}

func mergeFoodLabel(dst *Food, src Food) {
	if src.KcalPer100g != nil && dst.KcalPer100g == nil {
		dst.KcalPer100g = src.KcalPer100g
	}
	if src.EnergyKjPer100g != nil && dst.EnergyKjPer100g == nil {
		dst.EnergyKjPer100g = src.EnergyKjPer100g
	}
	if src.FatGPer100g != nil && dst.FatGPer100g == nil {
		dst.FatGPer100g = src.FatGPer100g
	}
	if src.SaturatesGPer100g != nil && dst.SaturatesGPer100g == nil {
		dst.SaturatesGPer100g = src.SaturatesGPer100g
	}
	if src.CarbsGPer100g != nil && dst.CarbsGPer100g == nil {
		dst.CarbsGPer100g = src.CarbsGPer100g
	}
	if src.SugarsGPer100g != nil && dst.SugarsGPer100g == nil {
		dst.SugarsGPer100g = src.SugarsGPer100g
	}
	if src.ProteinGPer100g != nil && dst.ProteinGPer100g == nil {
		dst.ProteinGPer100g = src.ProteinGPer100g
	}
	if src.SaltGPer100g != nil && dst.SaltGPer100g == nil {
		dst.SaltGPer100g = src.SaltGPer100g
	}
	if src.SodiumMgPer100g != nil && dst.SodiumMgPer100g == nil {
		dst.SodiumMgPer100g = src.SodiumMgPer100g
	}
}

// residualFoodNotes keeps user rules / approvals; drops dense label lines when structured data exists.
func residualFoodNotes(notes string) string {
	notes = strings.TrimSpace(notes)
	if notes == "" {
		return notes
	}
	if !rePer100.MatchString(notes) && !reLabelKcal.MatchString(notes) {
		return notes
	}
	// If the whole note is a label dump, keep only tail after "per 100 g" arrow or policy keywords.
	if idx := strings.Index(notes, "→"); idx >= 0 {
		tail := strings.TrimSpace(notes[idx+len("→"):])
		if tail != "" && !strings.Contains(strings.ToLower(tail), "per 100 g") {
			return trimLabelTail(tail)
		}
	}
	return trimLabelTail(notes)
}

func trimLabelTail(s string) string {
	s = strings.TrimSpace(s)
	for _, sep := range []string{" , ", " — ", " - APPROVED", " - REJECTED", " - **", ". **"} {
		if i := strings.Index(s, sep); i > 20 {
			return strings.TrimSpace(s[i+len(sep):])
		}
	}
	lower := strings.ToLower(s)
	for _, kw := range []string{"approved", "rejected", "only ", "rule:", "cap ", "never ", "don't", "not rebuy", "household", "verified", "label verified"} {
		if i := strings.Index(lower, kw); i >= 0 {
			return strings.TrimSpace(s[i:])
		}
	}
	if reLabelKcal.MatchString(s) && len(s) > 120 {
		return ""
	}
	return s
}
