package health

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadJSONBundleFromDir(t *testing.T) {
	dir := t.TempDir()
	w := []WeighIn{{ID: "w_2026-01-01_1", Date: "2026-01-01", Kg: 80}}
	b, _ := json.Marshal(w)
	_ = os.WriteFile(filepath.Join(dir, "weight.json"), b, 0644)
	bundle, err := LoadJSONBundleFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Weights) != 1 || bundle.Weights[0].Kg != 80 {
		t.Fatalf("%+v", bundle.Weights)
	}
}

func TestImportBundleSQLite(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "testuser")
	if err != nil {
		t.Fatal(err)
	}
	bundle := JSONBundle{
		Weights: []WeighIn{{ID: "w1", Date: "2026-01-02", Kg: 75}},
		Foods:   []Food{{ID: "apple", Name: "Apple", KcalPer100g: ptrF(52)}},
	}
	counts, err := st.ImportBundle(bundle, false)
	if err != nil {
		t.Fatal(err)
	}
	if counts["weights"] != 1 || counts["foods"] != 1 {
		t.Fatalf("%v", counts)
	}
	if st.IsEmpty() {
		t.Fatal("db should not be empty")
	}
}

func ptrF(v float64) *float64 { return &v }
