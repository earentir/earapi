package health

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func (s *Store) ListStockLots(activeOnly bool) ([]StockLot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	q := `SELECT id, name, prepared_date, batch_grams, remaining_grams, active, items_json, notes, kcal_per_100g, sodium_mg_per_100g
		FROM stock_lots WHERE user_id = ?`
	if activeOnly {
		q += ` AND active = 1`
	}
	q += ` ORDER BY active DESC, prepared_date DESC, name`
	rows, err := s.db.Query(q, s.userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out, err := scanStockLots(rows)
	if err != nil {
		return nil, err
	}
	byLot, err := s.listAllStockMovementsUnlocked()
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Usage = byLot[out[i].ID]
	}
	return out, nil
}

func scanStockLots(rows *sql.Rows) ([]StockLot, error) {
	var out []StockLot
	for rows.Next() {
		lot, err := scanStockFields(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, lot)
	}
	return out, rows.Err()
}

type scanFn func(dest ...any) error

func scanStockFields(scan scanFn) (StockLot, error) {
	var lot StockLot
	var prep, notes sql.NullString
	var itemsJSON string
	var kcal, sod sql.NullFloat64
	var active int
	if err := scan(&lot.ID, &lot.Name, &prep, &lot.BatchGrams, &lot.RemainingGrams, &active, &itemsJSON, &notes, &kcal, &sod); err != nil {
		return StockLot{}, err
	}
	stockLotSetActive(&lot, active != 0)
	if prep.Valid {
		lot.PreparedDate = prep.String
	}
	if notes.Valid {
		lot.Notes = notes.String
	}
	if itemsJSON != "" {
		_ = json.Unmarshal([]byte(itemsJSON), &lot.Items)
	}
	if kcal.Valid {
		v := kcal.Float64
		lot.KcalPer100g = &v
	}
	if sod.Valid {
		v := sod.Float64
		lot.SodiumMgPer100g = &v
	}
	return lot, nil
}

func (s *Store) StockLotByID(id string) (*StockLot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	lot, ok := s.stockLotByIDUnlocked(id)
	if !ok {
		return nil, false
	}
	out := *lot
	movs, _ := s.listStockMovementsUnlocked(id)
	out.Usage = movs
	return &out, true
}

func (s *Store) stockLotByIDUnlocked(id string) (*StockLot, bool) {
	row := s.db.QueryRow(`SELECT id, name, prepared_date, batch_grams, remaining_grams, active, items_json, notes, kcal_per_100g, sodium_mg_per_100g
		FROM stock_lots WHERE user_id = ? AND id = ?`, s.userID, id)
	lot, err := scanStockFields(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false
		}
		return nil, false
	}
	return &lot, true
}

func stockLotIsActive(lot StockLot) bool {
	if lot.RemainingGrams <= 0.01 {
		return false
	}
	if lot.Active != nil {
		return *lot.Active
	}
	return true
}

func stockLotSetActive(lot *StockLot, on bool) {
	lot.Active = &on
}

// refreshStockLotNutrientsFromCatalog scales set ingredients without DB reads (safe inside an open tx on a single connection).
func refreshStockLotNutrientsFromCatalog(lot *StockLot, catalog []Food) {
	if lot == nil || lot.BatchGrams <= 0 {
		return
	}
	if len(lot.Items) > 0 {
		virtual := MealItem{Name: lot.Name, Grams: lot.BatchGrams, Parts: lot.Items}
		_ = scaleDishWithCatalog(&virtual, catalog)
		k := virtual.Kcal / lot.BatchGrams * 100
		n := virtual.SodiumMg / lot.BatchGrams * 100
		lot.KcalPer100g = &k
		lot.SodiumMgPer100g = &n
		return
	}
	if nk, ns := parseNutrientsFromNotes(lot.Notes); nk != nil || ns != nil {
		lot.KcalPer100g = nk
		lot.SodiumMgPer100g = ns
	}
}

func (s *Store) refreshStockLotNutrients(lot *StockLot) {
	if lot == nil || lot.BatchGrams <= 0 {
		return
	}
	if len(lot.Items) > 0 {
		virtual := MealItem{Name: lot.Name, Grams: lot.BatchGrams, Parts: lot.Items}
		_ = s.scaleDishUnlocked(&virtual)
		k := virtual.Kcal / lot.BatchGrams * 100
		n := virtual.SodiumMg / lot.BatchGrams * 100
		lot.KcalPer100g = &k
		lot.SodiumMgPer100g = &n
		return
	}
	if nk, ns := parseNutrientsFromNotes(lot.Notes); nk != nil || ns != nil {
		lot.KcalPer100g = nk
		lot.SodiumMgPer100g = ns
	}
}

func (s *Store) UpsertStockLot(lot StockLot) (StockLot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name := strings.TrimSpace(lot.Name)
	if name == "" {
		return StockLot{}, fmt.Errorf("%w: stock name required", ErrInvalidInput)
	}
	if lot.BatchGrams <= 0 {
		return StockLot{}, fmt.Errorf("%w: batch_grams required", ErrInvalidInput)
	}
	isNew := lot.ID == ""
	var cur *StockLot
	if existing, ok := s.stockLotByIDUnlocked(lot.ID); ok {
		isNew = false
		cur = existing
		if lot.Name == "" {
			lot.Name = cur.Name
		}
		if lot.PreparedDate == "" {
			lot.PreparedDate = cur.PreparedDate
		}
		if lot.Notes == "" && len(lot.Items) == 0 {
			lot.Items = cur.Items
			lot.Notes = cur.Notes
		}
	} else if !isNew {
		return StockLot{}, ErrNotFound
	}
	if isNew && lot.RemainingGrams <= 0 {
		lot.RemainingGrams = lot.BatchGrams
	}
	if lot.RemainingGrams > lot.BatchGrams+0.01 {
		return StockLot{}, fmt.Errorf("%w: remaining cannot exceed batch", ErrInvalidInput)
	}
	activeFlag := lot.RemainingGrams > 0.01
	if lot.Active != nil {
		activeFlag = *lot.Active
	} else if !isNew && cur != nil {
		activeFlag = stockLotIsActive(*cur)
	}
	if lot.RemainingGrams <= 0.01 {
		activeFlag = false
	}
	stockLotSetActive(&lot, activeFlag)
	s.refreshStockLotNutrients(&lot)
	if lot.ID == "" {
		taken := map[string]bool{}
		rows, _ := s.db.Query(`SELECT id FROM stock_lots WHERE user_id = ?`, s.userID)
		if rows != nil {
			for rows.Next() {
				var id string
				_ = rows.Scan(&id)
				taken[id] = true
			}
			rows.Close()
		}
		lot.ID = UniqueSlug(name, taken)
	}
	lot.Name = name
	items, _ := json.Marshal(lot.Items)
	if items == nil {
		items = []byte("[]")
	}
	active := 0
	if stockLotIsActive(lot) {
		active = 1
	}
	_, err := s.db.Exec(`INSERT INTO stock_lots (user_id, id, name, prepared_date, batch_grams, remaining_grams, active, items_json, notes, kcal_per_100g, sodium_mg_per_100g)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, id) DO UPDATE SET
			name=excluded.name, prepared_date=excluded.prepared_date,
			batch_grams=excluded.batch_grams, remaining_grams=excluded.remaining_grams,
			active=excluded.active,
			items_json=excluded.items_json, notes=excluded.notes,
			kcal_per_100g=excluded.kcal_per_100g, sodium_mg_per_100g=excluded.sodium_mg_per_100g`,
		s.userID, lot.ID, lot.Name, nullStr(lot.PreparedDate), lot.BatchGrams, lot.RemainingGrams, active,
		string(items), lot.Notes, nullFloat(lot.KcalPer100g), nullFloat(lot.SodiumMgPer100g))
	if err != nil {
		return StockLot{}, err
	}
	if isNew {
		if err := s.seedStockImportUsageUnlocked(lot); err != nil {
			return StockLot{}, err
		}
	} else if cur != nil {
		var used float64
		_ = s.db.QueryRow(`SELECT COALESCE(SUM(grams),0) FROM stock_movements WHERE user_id = ? AND lot_id = ?`,
			s.userID, lot.ID).Scan(&used)
		targetUsed := lot.BatchGrams - lot.RemainingGrams
		if diff := targetUsed - used; diff > 0.01 || diff < -0.01 {
			if err := s.recordStockMovementUnlocked(lot.ID, diff, "manual", "", "", "Adjusted remaining"); err != nil {
				return StockLot{}, err
			}
		}
	}
	if err := s.recomputeStockRemainingFromLedgerUnlocked(lot.ID); err != nil {
		return StockLot{}, err
	}
	got, ok := s.stockLotByIDUnlocked(lot.ID)
	if !ok {
		return lot, nil
	}
	out := *got
	movs, _ := s.listStockMovementsUnlocked(lot.ID)
	out.Usage = movs
	return out, nil
}

func (s *Store) DeleteStockLot(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec(`DELETE FROM stock_lots WHERE user_id = ? AND id = ?`, s.userID, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DrawFromStock(lotID string, req StockDraw) (*Day, *StockLot, error) {
	slot := strings.ToLower(strings.TrimSpace(req.Slot))
	if !ValidMealSlots[slot] {
		return nil, nil, fmt.Errorf("%w: invalid slot", ErrInvalidInput)
	}
	date := stringsTrimDate(req.Date)
	if date == "" {
		date = todayAthens()
	}
	if req.Grams <= 0 {
		return nil, nil, fmt.Errorf("%w: grams required", ErrInvalidInput)
	}
	lot, ok := s.StockLotByID(lotID)
	if !ok {
		return nil, nil, ErrNotFound
	}
	if !stockLotIsActive(*lot) {
		return nil, nil, fmt.Errorf("%w: batch is not active", ErrInvalidInput)
	}
	if req.Grams > lot.RemainingGrams+0.01 {
		return nil, nil, fmt.Errorf("%w: only %.0f g left in stock", ErrInvalidInput, lot.RemainingGrams)
	}
	item := mealItemFromStockLot(*lot, req.Grams)
	d, found := s.GetDay(date)
	if !found {
		d = &Day{Date: date, Meals: map[string]MealSlot{}}
	}
	if d.Meals == nil {
		d.Meals = map[string]MealSlot{}
	}
	m := d.Meals[slot]
	m.Items = append(m.Items, item)
	if _, err := s.PrepareMealSlot(&m, nil); err != nil {
		return nil, nil, err
	}
	day, err := s.UpsertMealSlot(date, slot, m, true)
	if err != nil {
		return nil, nil, err
	}
	outLot, _ := s.StockLotByID(lotID)
	return day, outLot, nil
}

// AdjustFromStock decreases remaining without touching the meal log (household ate it / discarded).
func (s *Store) AdjustFromStock(lotID string, req StockAdjust) (*StockLot, error) {
	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode != "household" && mode != "discarded" {
		return nil, fmt.Errorf("%w: mode must be household or discarded", ErrInvalidInput)
	}
	lot, ok := s.StockLotByID(lotID)
	if !ok {
		return nil, ErrNotFound
	}
	grams := req.Grams
	if grams <= 0 {
		return nil, fmt.Errorf("%w: grams required", ErrInvalidInput)
	}
	if lot.RemainingGrams <= 0.01 {
		return nil, fmt.Errorf("%w: nothing left to adjust", ErrInvalidInput)
	}
	if grams > lot.RemainingGrams+0.01 {
		return nil, fmt.Errorf("%w: only %.0f g left in stock", ErrInvalidInput, lot.RemainingGrams)
	}
	if err := s.applyStockUsage(lotID, grams, mode, "", "", ""); err != nil {
		return nil, err
	}
	out, _ := s.StockLotByID(lotID)
	return out, nil
}

func (s *Store) applyStockUsage(lotID string, grams float64, kind, mealDate, mealSlot, note string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.applyStockUsageUnlocked(lotID, grams, kind, mealDate, mealSlot, note)
}

func (s *Store) SetStockLotActive(id string, active bool) (*StockLot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.stockLotByIDUnlocked(id); !ok {
		return nil, ErrNotFound
	}
	activeVal := 0
	if active {
		activeVal = 1
	}
	if _, err := s.db.Exec(`UPDATE stock_lots SET active=? WHERE user_id=? AND id=?`, activeVal, s.userID, id); err != nil {
		return nil, err
	}
	lot, ok := s.stockLotByIDUnlocked(id)
	if !ok {
		return nil, ErrNotFound
	}
	out := *lot
	stockLotSetActive(&out, active)
	return &out, nil
}

func (s *Store) applyStockRemaining(id string, grams float64, active ...bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.applyStockRemainingUnlocked(id, grams, active...)
}

func (s *Store) applyStockRemainingUnlocked(id string, grams float64, active ...bool) error {
	var q string
	var args []any
	if len(active) > 0 {
		act := 0
		if active[0] {
			act = 1
		}
		q = `UPDATE stock_lots SET remaining_grams=?, active=? WHERE user_id=? AND id=?`
		args = []any{grams, act, s.userID, id}
	} else {
		act := 1
		if grams <= 0.01 {
			act = 0
		}
		q = `UPDATE stock_lots SET remaining_grams=?, active=? WHERE user_id=? AND id=?`
		args = []any{grams, act, s.userID, id}
	}
	res, err := s.db.Exec(q, args...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) setStockRemaining(id string, grams float64) error {
	return s.applyStockRemaining(id, grams)
}

func mealItemFromStockLot(lot StockLot, grams float64) MealItem {
	it := MealItem{
		Name:       lot.Name,
		StockLotID: lot.ID,
		Grams:      grams,
		Note:       "from stock",
	}
	if lot.KcalPer100g != nil || lot.SodiumMgPer100g != nil {
		it.KcalPer100g = lot.KcalPer100g
		it.SodiumMgPer100g = lot.SodiumMgPer100g
	}
	return it
}

func nullStr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
