package health

import (
	"strings"
)

func stockGramsByLotID(items []MealItem) map[string]float64 {
	out := map[string]float64{}
	var walk func([]MealItem)
	walk = func(list []MealItem) {
		for _, it := range list {
			if it.StockLotID != "" && it.Grams > 0 {
				out[it.StockLotID] += it.Grams
			}
			if len(it.Parts) > 0 {
				walk(it.Parts)
			}
		}
	}
	walk(items)
	return out
}

// linkMealItemsToStock attaches stock_lot_id when the line name clearly matches an active lot.
func (s *Store) linkMealItemsToStock(items *[]MealItem) {
	if items == nil || len(*items) == 0 {
		return
	}
	lots, err := s.ListStockLots(true)
	if err != nil || len(lots) == 0 {
		return
	}
	for i := range *items {
		linkOneMealItemToStock(&(*items)[i], lots)
	}
}

func linkOneMealItemToStock(it *MealItem, lots []StockLot) {
	if it == nil {
		return
	}
	if len(it.Parts) > 0 {
		for j := range it.Parts {
			linkOneMealItemToStock(&it.Parts[j], lots)
		}
		return
	}
	if it.StockLotID != "" || it.Grams <= 0 {
		return
	}
	if id := matchMealItemToStockLot(it.Name, lots); id != "" {
		it.StockLotID = id
	}
}

func matchMealItemToStockLot(name string, lots []StockLot) string {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return ""
	}
	if strings.Contains(n, "soup-tomato") || strings.Contains(n, "soup tomato") {
		for _, lot := range lots {
			if strings.Contains(strings.ToLower(lot.Name), "soup-tomato") {
				return lot.ID
			}
		}
	}
	if strings.Contains(n, "non-tomato") {
		for _, lot := range lots {
			if strings.Contains(strings.ToLower(lot.Name), "non-tomato") {
				return lot.ID
			}
		}
	}
	for _, lot := range lots {
		ln := strings.ToLower(strings.TrimSpace(lot.Name))
		if ln == "" {
			continue
		}
		if n == ln {
			return lot.ID
		}
		if len(ln) >= 10 && strings.Contains(n, ln) {
			return lot.ID
		}
		if len(n) >= 10 && strings.Contains(ln, n) {
			return lot.ID
		}
	}
	return ""
}

// syncStockForMealSlotChange records meal ledger lines when stock_lot_id grams change (caller must hold s.mu).
func (s *Store) syncStockForMealSlotChange(date, slot string, oldItems, newItems []MealItem) error {
	oldG := stockGramsByLotID(oldItems)
	newG := stockGramsByLotID(newItems)
	ids := map[string]bool{}
	for id := range oldG {
		ids[id] = true
	}
	for id := range newG {
		ids[id] = true
	}
	slot = strings.ToLower(strings.TrimSpace(slot))
	for id := range ids {
		delta := newG[id] - oldG[id]
		if delta <= 0.01 && delta >= -0.01 {
			continue
		}
		lot, ok := s.stockLotByIDUnlocked(id)
		if !ok {
			continue
		}
		used := delta
		note := lot.Name
		if used < 0 {
			note = "meal edit removed"
		}
		if err := s.applyStockUsageUnlocked(id, used, "meal", date, slot, note); err != nil {
			return err
		}
		_ = lot
	}
	return nil
}

func mergeImportedStockWithDB(existing []StockLot, imported []StockLot) []StockLot {
	byID := map[string]StockLot{}
	for _, ex := range existing {
		byID[ex.ID] = ex
	}
	out := make([]StockLot, len(imported))
	for i, lot := range imported {
		out[i] = lot
		ex, ok := byID[lot.ID]
		if !ok {
			if out[i].RemainingGrams <= 0.01 {
				stockLotSetActive(&out[i], false)
			}
			continue
		}
		if ex.RemainingGrams <= 0.01 {
			out[i].RemainingGrams = 0
			stockLotSetActive(&out[i], false)
			continue
		}
		if ex.RemainingGrams < lot.RemainingGrams-0.01 {
			out[i].RemainingGrams = ex.RemainingGrams
			out[i].Active = ex.Active
		}
		if len(ex.Items) > 0 && len(out[i].Items) == 0 {
			out[i].Items = ex.Items
		}
		if out[i].RemainingGrams <= 0.01 {
			stockLotSetActive(&out[i], false)
		}
	}
	return out
}
