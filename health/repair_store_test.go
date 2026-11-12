package health

import (
	"testing"
)

func TestParseCountOnCornWafers(t *testing.T) {
	raw := "Corn wafers ×4 ~160 / ~166"
	items := ParseMealItemsFromRaw(raw)
	if len(items) != 1 {
		t.Fatalf("items %+v", items)
	}
	if items[0].Count != 4 {
		t.Fatalf("count %v", items[0].Count)
	}
	if items[0].Grams <= 0 {
		t.Fatalf("grams %v", items[0].Grams)
	}
}
