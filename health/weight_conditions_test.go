package health

import "testing"

func TestLegacyProtocolSetsAllConditions(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	w, err := st.AddWeight(WeighIn{Date: "2026-11-19", Kg: 142, Protocol: true})
	if err != nil {
		t.Fatal(err)
	}
	if !w.PostBM || !w.PreCoffee || !w.PreFood || !w.Protocol {
		t.Fatalf("legacy protocol: %+v", w)
	}
	got, ok := st.WeightByID(w.ID)
	if !ok {
		t.Fatal("missing")
	}
	if !got.PostBM || !got.PreCoffee || !got.PreFood {
		t.Fatalf("reload %+v", got)
	}
}

func TestWeighInFlagsIndependent(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	w, err := st.AddWeight(WeighIn{Date: "2026-11-19", Kg: 142, PostBM: true, PreCoffee: false, PreFood: true})
	if err != nil {
		t.Fatal(err)
	}
	if w.Protocol {
		t.Fatal("protocol should be false unless all three")
	}
	upd, err := st.UpdateWeight(w.ID, WeighIn{Kg: 142, PostBM: true, PreCoffee: true, PreFood: true})
	if err != nil {
		t.Fatal(err)
	}
	if !upd.Protocol || !upd.PreCoffee {
		t.Fatalf("%+v", upd)
	}
}
