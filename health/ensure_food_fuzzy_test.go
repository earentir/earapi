package health

import "testing"

func TestEnsureFoodFuzzyNoDuplicate(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	notes := `Marata Επιδόρπιο Γιαουρτιού 1.7% με Φράουλα (label verified Nov 7): 71 kcal / salt 0.12 g (~48 mg) per 100 g`
	k71, s48 := 71.0, 48.0
	if _, err := st.AddFood(Food{
		ID:              "marata-straw",
		Name:            "Marata Επιδόρπιο Γιαουρτιού 1.7% με Φράουλα",
		Notes:           notes,
		KcalPer100g:     &k71,
		SodiumMgPer100g: &s48,
	}); err != nil {
		t.Fatal(err)
	}
	k72 := 72.0
	got, created, err := st.EnsureFood(Food{Name: "Marata 1.7%", KcalPer100g: &k72, SodiumMgPer100g: &s48})
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatalf("expected fuzzy merge, created new food %q", got.Name)
	}
	if got.ID != "marata-straw" {
		t.Fatalf("got id %q", got.ID)
	}
	if st.FoodCount() != 1 {
		t.Fatalf("foods %d", st.FoodCount())
	}
}
