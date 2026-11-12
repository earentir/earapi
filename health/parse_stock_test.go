package health

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseStockMarkdown_activeLots(t *testing.T) {
	path := filepath.Join("..", "mddata", "Health Tracker - Misc.md")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Skip("mddata not present")
	}
	lots := parseStockMarkdown(string(b))
	if len(lots) < 8 {
		t.Fatalf("expected active stock lots, got %d", len(lots))
	}
	var active, depleted int
	names := map[string]float64{}
	for _, lot := range lots {
		if stockLotIsActive(lot) {
			active++
			if lot.RemainingGrams > 0.01 {
				names[lot.Name] = lot.RemainingGrams
			}
		} else {
			depleted++
		}
	}
	if active < 7 {
		t.Fatalf("expected several active lots, got %d", active)
	}
	if names["Sole-tomato sauce (Oct 30 batch)"] < 300 || names["Sole-tomato sauce (Oct 30 batch)"] > 310 {
		t.Fatalf("sole-tomato remaining: %v", names)
	}
	if yogurt, ok := names["Yogurt stock (Marata 1.7%)"]; !ok || yogurt < 590 || yogurt > 610 {
		t.Fatalf("yogurt stock: %v", names)
	}
	if soup, ok := names["soup-tomato mix"]; !ok || soup < 330 || soup > 350 {
		t.Fatalf("soup-tomato: %v", names)
	}
	var bean *StockLot
	for i := range lots {
		if strings.Contains(lots[i].Name, "Bean salad") {
			bean = &lots[i]
			break
		}
	}
	if bean == nil || bean.BatchGrams < 720 || bean.BatchGrams > 740 {
		t.Fatalf("bean batch %v", bean)
	}
	if len(bean.Items) < 2 {
		t.Fatalf("bean ingredients %v", bean.Items)
	}
	if _, ok := names["tikka-rice-chicken mix"]; ok {
		t.Fatalf("tikka sub-lot should not import as active fridge stock: %v", names)
	}
	_ = depleted
}

func TestStillInStockSubLotInactiveWhenParentBatchFinished(t *testing.T) {
	text := `Vegetable-curry sauce (Sep 13 batch): pepper 300 g + mushroom 300 g , **BATCH FINISHED Sep 18** , **tikka-rice-chicken mix 323 g saved (Sep 14 dinner) - STILL IN STOCK**`
	lots := parseStockBullet(text, "", map[string]bool{})
	var tikka *StockLot
	for i := range lots {
		if strings.Contains(strings.ToLower(lots[i].Name), "tikka") {
			tikka = &lots[i]
			break
		}
	}
	if tikka == nil {
		t.Fatal("expected tikka sub-lot")
	}
	if stockLotIsActive(*tikka) {
		t.Fatal("saved sub-portion on finished batch should import inactive")
	}
	if tikka.RemainingGrams != 323 {
		t.Fatalf("remaining %v", tikka.RemainingGrams)
	}
	if len(tikka.Notes) >= len(text) {
		t.Fatalf("notes should be snippet not whole bullet: len=%d", len(tikka.Notes))
	}
}

func TestSkipFoodBulletForCatalog(t *testing.T) {
	if !skipFoodBulletForCatalog("**YOGURT STOCK (Nov 7):** cups") {
		t.Fatal("expected yogurt stock skipped")
	}
	if skipFoodBulletForCatalog("**Marata Επιδόρπιο** temporary stock (4 cups)") {
		t.Fatal("catalog Marata must stay")
	}
	if !skipFoodBulletForCatalog("Salt in logs means Fytro unless real salt is explicit") {
		t.Fatal("expected rule bullet skipped")
	}
	if !skipFoodBulletForCatalog("Yellow curry sauce for mush (recipe, Sep 10): batch ~215 mg") {
		t.Fatal("recipe batch doc should skip")
	}
	if !skipFoodBulletForCatalog("Approved ready-meal rebuy list (salt g/100 g): moussaka") {
		t.Fatal("rebuy list policy should skip")
	}
}

func TestFilterStockFromFoods(t *testing.T) {
	foods := []Food{
		{ID: "a", Name: "YOGURT STOCK (Nov 7)", Notes: "cups"},
		{ID: "b", Name: "Marata yogurt", Notes: "label"},
	}
	out := filterStockFromFoods(foods)
	if len(out) != 1 || !strings.Contains(out[0].Name, "Marata") {
		t.Fatalf("filter: %+v", out)
	}
}
