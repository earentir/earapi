package health

import "strings"

// RepairAllDaysFromStoredRaw re-parses every saved day's meal raw_text (strip import glitches),
// re-scales from catalog/stock, and recomputes totals. Optional mdDays backfills legacy_totals by date.
func (s *Store) RepairAllDaysFromStoredRaw(mdDays []Day) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	legacyByDate := map[string]DayTotals{}
	for i := range mdDays {
		lt := mdDays[i].LegacyTotals
		if lt.Kcal > 0 || lt.SodiumMg > 0 {
			legacyByDate[mdDays[i].Date] = lt
		}
	}
	foods := s.listFoodsUnlocked()
	stock, _ := s.listStockLotsUnlocked()
	rows, err := s.db.Query(`SELECT date FROM days WHERE user_id = ? ORDER BY date ASC`, s.userID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var dates []string
	for rows.Next() {
		var dt string
		if rows.Scan(&dt) == nil {
			dates = append(dates, dt)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	repaired := 0
	for _, date := range dates {
		d, ok := s.readDayUnlocked(date)
		if !ok {
			continue
		}
		keepLegacy := d.LegacyTotals
		if lt, ok := legacyByDate[date]; ok {
			keepLegacy = lt
		}
		if !rebuildDayMealsFromRaw(d, foods, stock) {
			continue
		}
		d.LegacyTotals = keepLegacy
		if err := s.saveDayLocked(d); err != nil {
			return repaired, err
		}
		repaired++
	}
	return repaired, nil
}

func rebuildDayMealsFromRaw(d *Day, foods []Food, stock []StockLot) bool {
	if d == nil {
		return false
	}
	for slot, m := range d.Meals {
		m.Raw = stripMisplacedDayTotalFromRaw(m.Raw)
		if strings.TrimSpace(m.Raw) == "" {
			m.Items = nil
			d.Meals[slot] = m
			continue
		}
		if parsed := ParseMealStructureFromRaw(m.Raw); len(parsed) > 0 {
			m.Items = parsed
		}
		d.Meals[slot] = m
	}
	clearDayItemMacros(d)
	ScaleDayMealsFromCatalog(d, foods, stock, nil)
	RecomputeDayTotals(d)
	return true
}
