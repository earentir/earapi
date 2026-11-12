package health

import "testing"

func TestParseEUCroissant(t *testing.T) {
	text := `411 kcal / fat 21 g (sat 11 g) / carbs 49 g (sugars 23 g) / protein 5.6 g / salt 0.42 g (~168 mg sodium) per 100 g`
	f := ParseEULabelFromText(text)
	if f.KcalPer100g == nil || *f.KcalPer100g != 411 {
		t.Fatalf("kcal %v", f.KcalPer100g)
	}
	if f.FatGPer100g == nil || *f.FatGPer100g != 21 {
		t.Fatalf("fat %v", f.FatGPer100g)
	}
	if f.SaturatesGPer100g == nil || *f.SaturatesGPer100g != 11 {
		t.Fatalf("sat %v", f.SaturatesGPer100g)
	}
	if f.SaltGPer100g == nil || *f.SaltGPer100g != 0.42 {
		t.Fatalf("salt %v", f.SaltGPer100g)
	}
}

func TestApplyLabelSimpleTableFood(t *testing.T) {
	k, s := 130.0, 1.0
	f := Food{Name: "White rice", KcalPer100g: &k, SodiumMgPer100g: &s, Notes: "No salt when cooking"}
	ApplyLabelFromNotes(&f)
	if f.SaltGPer100g == nil || *f.SaltGPer100g < 0.002 || *f.SaltGPer100g > 0.003 {
		t.Fatalf("salt %v", f.SaltGPer100g)
	}
	if f.Notes != "No salt when cooking" {
		t.Fatalf("notes %q", f.Notes)
	}
}
