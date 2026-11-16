package health

import "testing"

func TestExportImportFoodsScope(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	k := 30.0
	f, err := st.AddFood(Food{Name: "Bell pepper", KcalPer100g: &k})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddFoodAlias(f.ID, "peppers"); err != nil {
		t.Fatal(err)
	}
	exp, err := st.ExportBundle("foods")
	if err != nil {
		t.Fatal(err)
	}
	if len(exp.Foods) != 1 || len(exp.Days) != 0 {
		t.Fatalf("foods export %+v", exp)
	}

	root2 := t.TempDir()
	db2, _ := OpenSharedDB(root2)
	st2, _ := NewStore(db2, root2, "u2")
	counts, err := st2.ImportBundleScope(exp, "foods", true)
	if err != nil {
		t.Fatal(err)
	}
	if counts["foods"] != 1 || counts["food_aliases"] != 1 {
		t.Fatalf("counts %v", counts)
	}
	if len(st2.ListFoods()) != 1 {
		t.Fatal("food missing")
	}
}