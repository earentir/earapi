package health

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFoodAliasesFromMDDir(t *testing.T) {
	dir := filepath.Join("..", "mddata")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("no mddata")
	}
	specs, err := LoadFoodAliasesFromMDDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) < 5 {
		t.Fatalf("specs %d", len(specs))
	}
	found := false
	for _, sp := range specs {
		if sp.Alias == "roasted wings" {
			found = true
			if sp.CatalogFood != "Roasted chicken wings (edible)" {
				t.Fatalf("%+v", sp)
			}
		}
	}
	if !found {
		t.Fatal("roasted wings alias missing")
	}
}
