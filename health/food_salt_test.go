package health

import "testing"

func TestSaltSodiumEUConversion(t *testing.T) {
	// Typical Greek retail chicken souvlaki (Megas Yeeros et al.): salt ~1.3 g / 100 g.
	sod := SodiumMgFromSaltPer100g(1.3)
	if sod < 510 || sod > 512 {
		t.Fatalf("sodium %v want ~511 mg per 100 g", sod)
	}
	back := SaltGFromSodiumMgPer100g(sod)
	if back < 1.29 || back > 1.31 {
		t.Fatalf("salt round-trip %v", back)
	}
}

func TestParseSaltFromNotesNotTimes1000(t *testing.T) {
	_, s := parseNutrientsFromNotes(`120 kcal / salt 1.3 g per 100 g`)
	if s == nil || *s < 510 || *s > 512 {
		t.Fatalf("sod from salt in notes %v want ~511", s)
	}
}