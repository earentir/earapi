package health

import "strings"

// filterStockLotsForImport keeps home-prepped batches only (soups, sauces with set ingredients).
// Purchased pantry counts (yogurt cups, cereal boxes, etc.) are not imported.
func filterStockLotsForImport(lots []StockLot) []StockLot {
	if len(lots) == 0 {
		return nil
	}
	out := make([]StockLot, 0, len(lots))
	for _, lot := range lots {
		if stockLotKeepForImport(lot) {
			out = append(out, lot)
		}
	}
	return out
}

func stockLotKeepForImport(lot StockLot) bool {
	if stockLotIsPurchasedInventory(lot) {
		return false
	}
	if len(lot.Items) > 0 {
		return true
	}
	name := strings.ToLower(strings.TrimSpace(lot.Name))
	for _, hint := range prepBatchStockNameHints {
		if strings.Contains(name, hint) {
			return true
		}
	}
	return false
}

var prepBatchStockNameHints = []string{
	"soup-tomato", "soup tomato", "carcass soup", "carcass #",
	"bean salad", "sole-tomato", "tomato mix", "non-tomato",
}

func stockLotIsPurchasedInventory(lot StockLot) bool {
	text := strings.TrimSpace(lot.Name + " " + lot.Notes)
	if text == "" {
		return true
	}
	low := strings.ToLower(text)
	for _, p := range []string{
		"yogurt stock",
		"cereal stock",
		"skewer stock",
		"wings stock",
		"kebab stock",
		"batch stock (",
		"temporary stock",
		" cups",
		" cup ",
		"bars ",
		"bar stock",
		"rebuy list",
		"ready meal",
		"ready-meal",
	} {
		if strings.Contains(low, p) {
			return true
		}
	}
	if len(lot.Items) == 0 && strings.Contains(low, " saved ") {
		return true
	}
	return false
}
