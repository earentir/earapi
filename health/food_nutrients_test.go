package health

import "testing"

func TestParseNutrientsFromMarataNote(t *testing.T) {
	notes := `**Marata Επιδόρπιο Γιαουρτιού 1.7% με Φράουλα (label verified Nov 7): 71 kcal / salt 0.12 g (~48 mg) per 100 g → per 200 g cup ~143 kcal / ~96 mg`
	k, s := parseNutrientsFromNotes(notes)
	if k == nil || *k != 71 {
		t.Fatalf("kcal %v", k)
	}
	if s == nil || *s != 48 {
		t.Fatalf("sod %v", s)
	}
}

func TestScaleMarataMealItem(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	notes := `71 kcal / salt 0.12 g (~48 mg) per 100 g`
	if _, err := st.AddFood(Food{ID: "marata", Name: "Marata 1.7%", Notes: notes}); err != nil {
		t.Fatal(err)
	}
	it := MealItem{Name: "Marata 1.7%", Grams: 200}
	if err := st.ScaleItemFromFood(&it); err != nil {
		t.Fatal(err)
	}
	if it.Kcal < 141 || it.Kcal > 143 {
		t.Fatalf("kcal %v", it.Kcal)
	}
}
