package health

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrInvalidInput = errors.New("invalid input")
)

// Store is one user's slice of the shared SQLite database.
type Store struct {
	mu      sync.Mutex
	db      *sql.DB
	userID  string
	userDir string
}

// NewStore attaches a user to the shared DB (healthdata/health.db).
func NewStore(db *sql.DB, dataRoot, username string) (*Store, error) {
	_ = os.MkdirAll(dataRoot, 0755)
	return &Store{db: db, userID: username, userDir: dataRoot}, nil
}

func (s *Store) UserDir() string { return s.userDir }

func (s *Store) isEmptyDB() (bool, error) {
	var n int
	err := s.db.QueryRow(`
		SELECT
		  (SELECT COUNT(*) FROM weigh_ins WHERE user_id = ?) +
		  (SELECT COUNT(*) FROM foods WHERE user_id = ?) +
		  (SELECT COUNT(*) FROM days WHERE user_id = ?)`,
		s.userID, s.userID, s.userID).Scan(&n)
	if err != nil {
		return true, err
	}
	return n == 0, nil
}

func (s *Store) IsEmpty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	ok, err := s.isEmptyDB()
	return err != nil || ok
}

// ReplaceImported replaces all rows for this user.
func (s *Store) ReplaceImported(weights []WeighIn, foods []Food, days []Day, stock []StockLot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existingStock, _ := s.listStockLotsUnlocked()
	movSnap, _ := s.listAllStockMovementsUnlocked()
	stock = mergeImportedStockWithDB(existingStock, stock)
	for i := range stock {
		refreshStockLotNutrientsFromCatalog(&stock[i], foods)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	uid := s.userID
	for _, q := range []string{
		`DELETE FROM meal_slots WHERE user_id = ?`,
		`DELETE FROM days WHERE user_id = ?`,
		`DELETE FROM weigh_ins WHERE user_id = ?`,
		`DELETE FROM foods WHERE user_id = ?`,
		`DELETE FROM stock_lots WHERE user_id = ?`,
	} {
		if _, err := tx.Exec(q, uid); err != nil {
			return err
		}
	}

	for _, w := range weights {
		if err := insertWeighInUser(tx, uid, w); err != nil {
			return err
		}
	}
	seenFood := map[string]bool{}
	for _, f := range foods {
		key := strings.ToLower(strings.TrimSpace(f.Name))
		if key == "" || seenFood[key] {
			continue
		}
		seenFood[key] = true
		if err := insertFoodUser(tx, uid, f); err != nil {
			return err
		}
	}
	for _, lot := range stock {
		if err := insertStockLotUser(tx, uid, lot); err != nil {
			return err
		}
		if prev, ok := movSnap[lot.ID]; ok && len(prev) > 0 {
			for _, m := range prev {
				if err := insertStockMovementUser(tx, uid, m); err != nil {
					return err
				}
			}
		} else if used := lot.BatchGrams - lot.RemainingGrams; used > 0.01 {
			m := StockMovement{
				LotID:     lot.ID,
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
				Grams:     used,
				Kind:      "import",
				Note:      "Used before ledger (Misc / import)",
			}
			m.ID = UniqueSlug(lot.ID+"-import", map[string]bool{})
			if err := insertStockMovementUser(tx, uid, m); err != nil {
				return err
			}
		}
	}
	for i := range days {
		dd := days[i]
		if dd.LegacyTotals.Kcal == 0 && dd.LegacyTotals.SodiumMg == 0 &&
			(dd.Totals.Kcal > 0 || dd.Totals.SodiumMg > 0) {
			dd.LegacyTotals = dd.Totals
		}
		ScaleDayMealsFromCatalog(&dd, foods, stock)
		RecomputeDayTotals(&dd)
		if err := saveDayTxUser(tx, uid, &dd); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	for _, lot := range stock {
		_ = s.recomputeStockRemainingFromLedgerUnlocked(lot.ID)
	}
	return nil
}

func nullFloat(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

func (s *Store) preserveLegacyTotalsLocked(d *Day) {
	if d == nil || d.Date == "" {
		return
	}
	if d.LegacyTotals.Kcal > 0 || d.LegacyTotals.SodiumMg > 0 {
		return
	}
	ex, ok := s.readDayUnlocked(d.Date)
	if !ok {
		return
	}
	if ex.LegacyTotals.Kcal > 0 || ex.LegacyTotals.SodiumMg > 0 {
		d.LegacyTotals = ex.LegacyTotals
	}
}

func saveDayTxUser(tx *sql.Tx, userID string, d *Day) error {
	if d.Meals == nil {
		d.Meals = map[string]MealSlot{}
	}
	RecomputeDayTotals(d)
	_, err := tx.Exec(`INSERT INTO days (user_id, date, bp, walking, total_sodium_mg, total_kcal, legacy_total_sodium_mg, legacy_total_kcal)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, date) DO UPDATE SET
		  bp=excluded.bp, walking=excluded.walking,
		  total_sodium_mg=excluded.total_sodium_mg, total_kcal=excluded.total_kcal,
		  legacy_total_sodium_mg=excluded.legacy_total_sodium_mg, legacy_total_kcal=excluded.legacy_total_kcal`,
		userID, d.Date, d.BP, d.Walking, d.Totals.SodiumMg, d.Totals.Kcal,
		d.LegacyTotals.SodiumMg, d.LegacyTotals.Kcal)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM meal_slots WHERE user_id = ? AND date = ?`, userID, d.Date); err != nil {
		return err
	}
	for slot, m := range d.Meals {
		items, _ := json.Marshal(m.Items)
		if items == nil {
			items = []byte("[]")
		}
		_, err := tx.Exec(`INSERT INTO meal_slots (user_id, date, slot, raw_text, sodium_mg, kcal, kcal_margin, items_json)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			userID, d.Date, slot, m.Raw, m.SodiumMg, m.Kcal, m.KcalMargin, string(items))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) saveDayTx(tx *sql.Tx, d *Day) error {
	s.preserveLegacyTotalsLocked(d)
	return saveDayTxUser(tx, s.userID, d)
}

// --- weight ---

func (s *Store) ListWeights(from, to string) []WeighIn {
	s.mu.Lock()
	defer s.mu.Unlock()
	q := `SELECT id, date, kg, rhr_bpm, note, protocol, kg_raw FROM weigh_ins WHERE user_id = ?`
	args := []any{s.userID}
	if from != "" {
		q += ` AND date >= ?`
		args = append(args, from)
	}
	if to != "" {
		q += ` AND date <= ?`
		args = append(args, to)
	}
	q += ` ORDER BY date ASC, id ASC`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanWeights(rows)
}

func scanWeights(rows *sql.Rows) []WeighIn {
	var out []WeighIn
	for rows.Next() {
		var w WeighIn
		var rhr sql.NullInt64
		var prot int
		if err := rows.Scan(&w.ID, &w.Date, &w.Kg, &rhr, &w.Note, &prot, &w.KgRaw); err != nil {
			continue
		}
		if rhr.Valid {
			v := int(rhr.Int64)
			w.RHRBpm = &v
		}
		w.Protocol = prot != 0
		out = append(out, w)
	}
	return out
}

func (s *Store) LatestWeight() (*WeighIn, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row := s.db.QueryRow(`SELECT id, date, kg, rhr_bpm, note, protocol, kg_raw FROM weigh_ins
		WHERE user_id = ? ORDER BY date DESC, id DESC LIMIT 1`, s.userID)
	var w WeighIn
	var rhr sql.NullInt64
	var prot int
	if err := row.Scan(&w.ID, &w.Date, &w.Kg, &rhr, &w.Note, &prot, &w.KgRaw); err != nil {
		return nil, false
	}
	if rhr.Valid {
		v := int(rhr.Int64)
		w.RHRBpm = &v
	}
	w.Protocol = prot != 0
	return &w, true
}

func (s *Store) AddWeight(w WeighIn) (WeighIn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if w.Date == "" {
		w.Date = todayAthens()
	}
	if w.Kg <= 0 {
		return WeighIn{}, fmt.Errorf("%w: kg required", ErrInvalidInput)
	}
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM weigh_ins WHERE user_id = ? AND date = ?`, s.userID, w.Date).Scan(&n)
	w.ID = fmt.Sprintf("w_%s_%d", w.Date, n+1)
	if err := insertWeighInUser(s.db, s.userID, w); err != nil {
		return WeighIn{}, err
	}
	return w, nil
}

func (s *Store) WeightByID(id string) (*WeighIn, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.weightByIDUnlocked(id)
}

func (s *Store) weightByIDUnlocked(id string) (*WeighIn, bool) {
	row := s.db.QueryRow(`SELECT id, date, kg, rhr_bpm, note, protocol, kg_raw FROM weigh_ins
		WHERE user_id = ? AND id = ?`, s.userID, id)
	var w WeighIn
	var rhr sql.NullInt64
	var prot int
	if err := row.Scan(&w.ID, &w.Date, &w.Kg, &rhr, &w.Note, &prot, &w.KgRaw); err != nil {
		return nil, false
	}
	if rhr.Valid {
		v := int(rhr.Int64)
		w.RHRBpm = &v
	}
	w.Protocol = prot != 0
	return &w, true
}

func (s *Store) UpdateWeight(id string, patch WeighIn) (*WeighIn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.weightByIDUnlocked(id)
	if !ok {
		return nil, ErrNotFound
	}
	if patch.Date != "" {
		cur.Date = patch.Date
	}
	if patch.Kg > 0 {
		cur.Kg = patch.Kg
	}
	cur.Note = patch.Note
	cur.KgRaw = patch.KgRaw
	cur.Protocol = patch.Protocol
	if patch.RHRBpm != nil {
		cur.RHRBpm = patch.RHRBpm
	}
	if err := s.writeWeightUnlocked(*cur); err != nil {
		return nil, err
	}
	return cur, nil
}

func (s *Store) writeWeightUnlocked(w WeighIn) error {
	_, err := s.db.Exec(`UPDATE weigh_ins SET date=?, kg=?, rhr_bpm=?, note=?, protocol=?, kg_raw=? WHERE user_id = ? AND id=?`,
		w.Date, w.Kg, nullInt(w.RHRBpm), w.Note, boolInt(w.Protocol), w.KgRaw, s.userID, w.ID)
	return err
}

func (s *Store) DeleteWeight(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec(`DELETE FROM weigh_ins WHERE user_id = ? AND id = ?`, s.userID, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func nullInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// --- foods ---

func (s *Store) ListFoods() []Food {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listFoodsUnlocked()
}

func (s *Store) listFoodsUnlocked() []Food {
	rows, err := s.db.Query(`SELECT id, name, kcal_per_100g, sodium_mg_per_100g, notes,
		energy_kj_per_100g, fat_g_per_100g, saturates_g_per_100g, carbs_g_per_100g, sugars_g_per_100g, protein_g_per_100g, salt_g_per_100g
		FROM foods WHERE user_id = ? ORDER BY name`, s.userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanFoods(rows)
}

func scanFoods(rows *sql.Rows) []Food {
	var out []Food
	for rows.Next() {
		f, ok := scanFoodRow(rows)
		if ok {
			out = append(out, f)
		}
	}
	return out
}

func scanFoodRow(rows *sql.Rows) (Food, bool) {
	var f Food
	var kcal, sod, kj, fat, sat, carb, sugar, protein, salt sql.NullFloat64
	if err := rows.Scan(&f.ID, &f.Name, &kcal, &sod, &f.Notes,
		&kj, &fat, &sat, &carb, &sugar, &protein, &salt); err != nil {
		return Food{}, false
	}
	assignNullFloat(&f.KcalPer100g, kcal)
	assignNullFloat(&f.SodiumMgPer100g, sod)
	assignNullFloat(&f.EnergyKjPer100g, kj)
	assignNullFloat(&f.FatGPer100g, fat)
	assignNullFloat(&f.SaturatesGPer100g, sat)
	assignNullFloat(&f.CarbsGPer100g, carb)
	assignNullFloat(&f.SugarsGPer100g, sugar)
	assignNullFloat(&f.ProteinGPer100g, protein)
	assignNullFloat(&f.SaltGPer100g, salt)
	return f, true
}

func assignNullFloat(dst **float64, n sql.NullFloat64) {
	if n.Valid {
		v := n.Float64
		*dst = &v
	}
}

func (s *Store) FoodByID(id string) (*Food, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.foodByIDUnlocked(id)
}

// EnsureFood upserts a catalog food. When name is new, BestFoodMatch is tried before insert.
// The bool is true only when a new DB row was created.
func (s *Store) EnsureFood(f Food) (Food, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name := strings.TrimSpace(f.Name)
	if f.ID != "" {
		if cur, ok := s.foodByIDUnlocked(f.ID); ok {
			applyFoodPatch(cur, f)
			if err := s.writeFoodUnlocked(*cur); err != nil {
				return Food{}, false, err
			}
			return *cur, false, nil
		}
		if name == "" {
			return Food{}, false, fmt.Errorf("%w: unknown food id %q and no name", ErrInvalidInput, f.ID)
		}
	}
	if name == "" {
		return Food{}, false, fmt.Errorf("%w: name required", ErrInvalidInput)
	}
	var existingID string
	err := s.db.QueryRow(`SELECT id FROM foods WHERE user_id = ? AND name = ? COLLATE NOCASE`, s.userID, name).Scan(&existingID)
	if err == nil {
		cur, ok := s.foodByIDUnlocked(existingID)
		if !ok {
			return Food{}, false, err
		}
		applyFoodPatch(cur, f)
		if err := s.writeFoodUnlocked(*cur); err != nil {
			return Food{}, false, err
		}
		return *cur, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Food{}, false, err
	}
	if match, ok := BestFoodMatch(s.listFoodsUnlocked(), name); ok {
		cur, ok := s.foodByIDUnlocked(match.ID)
		if ok {
			patch := f
			patch.Name = ""
			patch.ID = ""
			applyFoodPatch(cur, patch)
			if err := s.writeFoodUnlocked(*cur); err != nil {
				return Food{}, false, err
			}
			return *cur, false, nil
		}
	}
	if f.KcalPer100g == nil && f.SodiumMgPer100g == nil {
		return Food{}, false, fmt.Errorf("%w: new food %q needs kcal_per_100g and/or sodium_mg_per_100g", ErrInvalidInput, name)
	}
	taken := map[string]bool{}
	rows, _ := s.db.Query(`SELECT id FROM foods WHERE user_id = ?`, s.userID)
	if rows != nil {
		for rows.Next() {
			var id string
			_ = rows.Scan(&id)
			taken[id] = true
		}
		rows.Close()
	}
	f.Name = name
	if f.ID == "" {
		f.ID = UniqueSlug(name, taken)
	} else if taken[f.ID] {
		return Food{}, false, ErrConflict
	}
	if err := insertFoodUser(s.db, s.userID, f); err != nil {
		return Food{}, false, err
	}
	return f, true, nil
}

func applyFoodPatch(cur *Food, patch Food) {
	if patch.Name != "" {
		cur.Name = strings.TrimSpace(patch.Name)
	}
	if patch.KcalPer100g != nil {
		cur.KcalPer100g = patch.KcalPer100g
	}
	if patch.SodiumMgPer100g != nil {
		cur.SodiumMgPer100g = patch.SodiumMgPer100g
	}
	if patch.EnergyKjPer100g != nil {
		cur.EnergyKjPer100g = patch.EnergyKjPer100g
	}
	if patch.FatGPer100g != nil {
		cur.FatGPer100g = patch.FatGPer100g
	}
	if patch.SaturatesGPer100g != nil {
		cur.SaturatesGPer100g = patch.SaturatesGPer100g
	}
	if patch.CarbsGPer100g != nil {
		cur.CarbsGPer100g = patch.CarbsGPer100g
	}
	if patch.SugarsGPer100g != nil {
		cur.SugarsGPer100g = patch.SugarsGPer100g
	}
	if patch.ProteinGPer100g != nil {
		cur.ProteinGPer100g = patch.ProteinGPer100g
	}
	if patch.SaltGPer100g != nil {
		cur.SaltGPer100g = patch.SaltGPer100g
	}
	if patch.Notes != "" {
		cur.Notes = patch.Notes
	}
	NormalizeFoodLabel(cur)
}

func (s *Store) writeFoodUnlocked(f Food) error {
	return execFoodUpdate(s.db, s.userID, f)
}

func (s *Store) AddFood(f Food) (Food, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name := strings.TrimSpace(f.Name)
	if name == "" {
		return Food{}, fmt.Errorf("%w: name required", ErrInvalidInput)
	}
	var existingID string
	err := s.db.QueryRow(`SELECT id FROM foods WHERE user_id = ? AND name = ? COLLATE NOCASE`, s.userID, name).Scan(&existingID)
	if err == nil {
		return Food{}, ErrConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Food{}, err
	}
	taken := map[string]bool{}
	rows, _ := s.db.Query(`SELECT id FROM foods WHERE user_id = ?`, s.userID)
	if rows != nil {
		for rows.Next() {
			var id string
			_ = rows.Scan(&id)
			taken[id] = true
		}
		rows.Close()
	}
	f.Name = name
	if f.ID == "" {
		f.ID = UniqueSlug(name, taken)
	} else if taken[f.ID] {
		return Food{}, ErrConflict
	}
	if err := insertFoodUser(s.db, s.userID, f); err != nil {
		return Food{}, err
	}
	return f, nil
}

func (s *Store) UpdateFood(id string, patch Food) (*Food, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.foodByIDUnlocked(id)
	if !ok {
		return nil, ErrNotFound
	}
	applyFoodPatch(cur, patch)
	if err := execFoodUpdate(s.db, s.userID, *cur); err != nil {
		return nil, err
	}
	return cur, nil
}

func (s *Store) DeleteFood(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec(`DELETE FROM foods WHERE user_id = ? AND id = ?`, s.userID, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) FoodCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM foods WHERE user_id = ?`, s.userID).Scan(&n)
	return n
}

func (s *Store) foodByIDUnlocked(id string) (*Food, bool) {
	row := s.db.QueryRow(`SELECT id, name, kcal_per_100g, sodium_mg_per_100g, notes,
		energy_kj_per_100g, fat_g_per_100g, saturates_g_per_100g, carbs_g_per_100g, sugars_g_per_100g, protein_g_per_100g, salt_g_per_100g
		FROM foods WHERE user_id = ? AND id = ?`, s.userID, id)
	var f Food
	var kcal, sod, kj, fat, sat, carb, sugar, protein, salt sql.NullFloat64
	if err := row.Scan(&f.ID, &f.Name, &kcal, &sod, &f.Notes,
		&kj, &fat, &sat, &carb, &sugar, &protein, &salt); err != nil {
		return nil, false
	}
	assignNullFloat(&f.KcalPer100g, kcal)
	assignNullFloat(&f.SodiumMgPer100g, sod)
	assignNullFloat(&f.EnergyKjPer100g, kj)
	assignNullFloat(&f.FatGPer100g, fat)
	assignNullFloat(&f.SaturatesGPer100g, sat)
	assignNullFloat(&f.CarbsGPer100g, carb)
	assignNullFloat(&f.SugarsGPer100g, sugar)
	assignNullFloat(&f.ProteinGPer100g, protein)
	assignNullFloat(&f.SaltGPer100g, salt)
	return &f, true
}

// --- days ---

func (s *Store) readDayUnlocked(date string) (*Day, bool) {
	row := s.db.QueryRow(`SELECT date, bp, walking, total_sodium_mg, total_kcal, legacy_total_sodium_mg, legacy_total_kcal FROM days WHERE user_id = ? AND date = ?`, s.userID, date)
	var d Day
	if err := row.Scan(&d.Date, &d.BP, &d.Walking, &d.Totals.SodiumMg, &d.Totals.Kcal, &d.LegacyTotals.SodiumMg, &d.LegacyTotals.Kcal); err != nil {
		return nil, false
	}
	d.Meals = map[string]MealSlot{}
	rows, err := s.db.Query(`SELECT slot, raw_text, sodium_mg, kcal, kcal_margin, items_json FROM meal_slots WHERE user_id = ? AND date = ?`, s.userID, date)
	if err != nil {
		return &d, true
	}
	defer rows.Close()
	for rows.Next() {
		var slot, raw, itemsJSON string
		var m MealSlot
		if err := rows.Scan(&slot, &raw, &m.SodiumMg, &m.Kcal, &m.KcalMargin, &itemsJSON); err != nil {
			continue
		}
		m.Raw = raw
		_ = json.Unmarshal([]byte(itemsJSON), &m.Items)
		d.Meals[slot] = m
	}
	if err := rows.Err(); err != nil {
		return nil, false
	}
	return &d, true
}

// scaleMealSlotsLenient fills item kcal/sodium from the food catalog when possible (read-only; errors ignored).
// Caller must not hold s.mu; uses unlocked DB reads only.
func (s *Store) scaleMealSlotsLenient(d *Day) {
	if d == nil || d.Meals == nil {
		return
	}
	active := make(map[string]bool)
	var scale func(*MealItem)
	scale = func(it *MealItem) {
		if len(it.Parts) > 0 {
			for i := range it.Parts {
				scale(&it.Parts[i])
			}
			_ = s.scaleDishUnlocked(it, active)
			return
		}
		_ = s.scaleItemFromFoodUnlocked(it, active)
	}
	for slot, m := range d.Meals {
		for i := range m.Items {
			scale(&m.Items[i])
		}
		d.Meals[slot] = m
	}
}

func (s *Store) saveDayLocked(d *Day) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.saveDayTx(tx, d); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) GetDay(date string) (*Day, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.readDayUnlocked(date)
	if !ok {
		return nil, false
	}
	dd := *d
	if RepairDayMealImportArtifacts(&dd) {
		foods := s.listFoodsUnlocked()
		stock, _ := s.listStockLotsUnlocked()
		rebuildDayMealsFromRaw(&dd, foods, stock)
		_ = s.saveDayLocked(&dd)
		return &dd, true
	}
	s.scaleMealSlotsLenient(&dd)
	for slot, m := range dd.Meals {
		RecomputeMealSlotTotalsFromItems(&m)
		dd.Meals[slot] = m
	}
	RecomputeDayTotals(&dd)
	return &dd, true
}

func (s *Store) ListDayDates() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT date FROM days WHERE user_id = ? ORDER BY date ASC`, s.userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var dates []string
	for rows.Next() {
		var d string
		if rows.Scan(&d) == nil {
			dates = append(dates, d)
		}
	}
	if err := rows.Err(); err != nil {
		return nil
	}
	return dates
}

func (s *Store) PatchDayMeta(date string, bp, walking *string) (*Day, error) {
	if date == "" {
		return nil, fmt.Errorf("%w: date required", ErrInvalidInput)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.readDayUnlocked(date)
	if !ok {
		d = &Day{Date: date, Meals: map[string]MealSlot{}}
	}
	if bp != nil {
		d.BP = *bp
	}
	if walking != nil {
		d.Walking = *walking
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := s.saveDayTx(tx, d); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Store) PutDay(d Day) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d.Date == "" {
		return fmt.Errorf("%w: date required", ErrInvalidInput)
	}
	if d.Meals == nil {
		d.Meals = map[string]MealSlot{}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.saveDayTx(tx, &d); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) DeleteDay(date string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec(`DELETE FROM days WHERE user_id = ? AND date = ?`, s.userID, date)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) UpsertMealSlot(date, slot string, meal MealSlot, replace bool) (*Day, error) {
	slot = strings.ToLower(strings.TrimSpace(slot))
	if !ValidMealSlots[slot] {
		return nil, fmt.Errorf("%w: invalid slot", ErrInvalidInput)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.readDayUnlocked(date)
	if !ok {
		d = &Day{Date: date, Meals: map[string]MealSlot{}}
	}
	var oldItems []MealItem
	if cur, exists := d.Meals[slot]; exists {
		oldItems = append([]MealItem(nil), cur.Items...)
	}
	cur, exists := d.Meals[slot]
	if replace || !exists {
		d.Meals[slot] = meal
	} else {
		if meal.Raw != "" {
			if cur.Raw != "" {
				cur.Raw += " + " + meal.Raw
			} else {
				cur.Raw = meal.Raw
			}
		}
		cur.Items = append(cur.Items, meal.Items...)
		if meal.SodiumMg > 0 {
			cur.SodiumMg += meal.SodiumMg
		}
		if meal.Kcal > 0 {
			cur.Kcal += meal.Kcal
		}
		d.Meals[slot] = cur
	}
	mealAfter := d.Meals[slot]
	if err := s.syncStockForMealSlotChange(date, slot, oldItems, mealAfter.Items); err != nil {
		return nil, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := s.saveDayTx(tx, d); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Store) SetMealSlot(date, slot string, meal MealSlot) (*Day, error) {
	return s.UpsertMealSlot(date, slot, meal, true)
}

func (s *Store) DeleteMealSlot(date, slot string) (*Day, error) {
	slot = strings.ToLower(strings.TrimSpace(slot))
	if !ValidMealSlots[slot] {
		return nil, fmt.Errorf("%w: invalid slot", ErrInvalidInput)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.readDayUnlocked(date)
	if !ok {
		return nil, ErrNotFound
	}
	var oldItems []MealItem
	if m, ok := d.Meals[slot]; ok {
		oldItems = m.Items
	}
	delete(d.Meals, slot)
	if err := s.syncStockForMealSlotChange(date, slot, oldItems, nil); err != nil {
		return nil, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := s.saveDayTx(tx, d); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Store) listStockLotsUnlocked() ([]StockLot, error) {
	rows, err := s.db.Query(`SELECT id, name, prepared_date, batch_grams, remaining_grams, active, items_json, notes, kcal_per_100g, sodium_mg_per_100g
		FROM stock_lots WHERE user_id = ? ORDER BY prepared_date DESC, name`, s.userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStockLots(rows)
}

// RecomputeDayTotals sums meal slots into day totals.
func RecomputeDayTotals(d *Day) {
	var sod, kc float64
	for _, slot := range []string{"lunch", "dinner", "snack"} {
		m, ok := d.Meals[slot]
		if !ok {
			continue
		}
		FinalizeMealTotals(&m)
		d.Meals[slot] = m
		sod += m.SodiumMg
		kc += m.Kcal
	}
	d.Totals = DayTotals{SodiumMg: sod, Kcal: kc}
}

func todayAthens() string {
	loc, err := time.LoadLocation("Europe/Athens")
	if err != nil {
		loc = time.UTC
	}
	return time.Now().In(loc).Format("2006-01-02")
}

func (s *Store) FilterDays(from, to, exactDate, q, slot string, limit int) []Day {
	s.mu.Lock()
	defer s.mu.Unlock()
	sqlQ := `SELECT date FROM days WHERE user_id = ?`
	args := []any{s.userID}
	if exactDate != "" {
		sqlQ += ` AND date = ?`
		args = append(args, exactDate)
	}
	if from != "" {
		sqlQ += ` AND date >= ?`
		args = append(args, from)
	}
	if to != "" {
		sqlQ += ` AND date <= ?`
		args = append(args, to)
	}
	sqlQ += ` ORDER BY date DESC`
	rows, err := s.db.Query(sqlQ, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var dates []string
	for rows.Next() {
		var dt string
		if rows.Scan(&dt) != nil {
			continue
		}
		dates = append(dates, dt)
	}
	if err := rows.Err(); err != nil {
		return nil
	}
	var out []Day
	for _, dt := range dates {
		d, ok := s.readDayUnlocked(dt)
		if !ok {
			continue
		}
		if slot != "" {
			slot = strings.ToLower(slot)
			m, has := d.Meals[slot]
			if !has || mealEmpty(m) {
				continue
			}
			if q != "" && !mealMatchesQuery(m, q) {
				continue
			}
		} else if q != "" && !dayMatchesQuery(d, q) {
			continue
		}
		out = append(out, *d)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	if limit <= 0 && from == "" && to == "" && exactDate == "" && q == "" && slot == "" {
		if len(out) > 14 {
			out = out[:14]
		}
	}
	return out
}

func mealEmpty(m MealSlot) bool {
	return strings.TrimSpace(m.Raw) == "" && len(m.Items) == 0
}

func mealMatchesQuery(m MealSlot, q string) bool {
	q = strings.ToLower(q)
	if strings.Contains(strings.ToLower(m.Raw), q) {
		return true
	}
	for _, it := range m.Items {
		if strings.Contains(strings.ToLower(it.Name), q) || strings.Contains(strings.ToLower(it.Note), q) {
			return true
		}
	}
	return false
}

func dayMatchesQuery(d *Day, q string) bool {
	q = strings.ToLower(q)
	for _, slot := range []string{"lunch", "dinner", "snack"} {
		if mealMatchesQuery(d.Meals[slot], q) {
			return true
		}
	}
	if strings.Contains(strings.ToLower(d.BP), q) || strings.Contains(strings.ToLower(d.Walking), q) {
		return true
	}
	return false
}

func (s *Store) ScaleItemFromFood(it *MealItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scaleItemFromFoodUnlocked(it, nil)
}

func (s *Store) scaleItemFromFoodUnlocked(it *MealItem, active map[string]bool) error {
	if len(it.Parts) > 0 {
		return s.scaleDishUnlocked(it, active)
	}
	if it.Grams <= 0 {
		FillMealItemGramsFromCount(it, s.listFoodsUnlocked())
	}
	if it.Grams <= 0 {
		return nil
	}
	if s.scaleItemFromStockUnlockedVis(it, active) {
		return nil
	}
	var f *Food
	var ok bool
	if it.FoodID != "" {
		f, ok = s.foodByIDUnlocked(it.FoodID)
	} else if byAlias, okAlias := s.foodByAliasUnlocked(it.Name); okAlias {
		f, ok = byAlias, true
	} else {
		f, ok = BestFoodMatch(s.listFoodsUnlocked(), it.Name)
	}
	if !ok || f == nil {
		return nil
	}
	if it.FoodID == "" {
		it.FoodID = f.ID
	}
	if strings.TrimSpace(it.Name) == "" {
		it.Name = f.Name
	}
	kcal100, sod100 := nutrientsForFood(f)
	if kcal100 != nil && it.Kcal == 0 {
		it.Kcal = *kcal100 * it.Grams / 100
	}
	if sod100 != nil && it.SodiumMg == 0 {
		it.SodiumMg = *sod100 * it.Grams / 100
	}
	return nil
}

func (s *Store) scaleDishUnlocked(it *MealItem, active map[string]bool) error {
	var batch, k, n float64
	for i := range it.Parts {
		p := &it.Parts[i]
		if strings.TrimSpace(p.Name) == "" {
			continue
		}
		_ = s.scaleItemFromFoodUnlocked(p, active)
		batch += p.Grams
		k += p.Kcal
		n += p.SodiumMg
	}
	// Grouping sets: total grams = sum of parts (full portion). Scale down only when eaten g
	// is explicitly below batch (leftovers / partial plate).
	factor := 1.0
	if batch <= 0 {
		it.Grams = 0
		it.Kcal = 0
		it.SodiumMg = 0
		return nil
	}
	eaten := it.Grams
	if eaten <= 0 || eaten >= batch-0.01 {
		it.Grams = batch
		factor = 1.0
	} else {
		factor = eaten / batch
	}
	it.Kcal = k * factor
	it.SodiumMg = n * factor
	return nil
}

func scaleItemFromFoodWithCatalog(it *MealItem, catalog []Food) error {
	if len(it.Parts) > 0 {
		return scaleDishWithCatalog(it, catalog)
	}
	if it.Grams <= 0 {
		return nil
	}
	f, ok := BestFoodMatch(catalog, it.Name)
	if !ok || f == nil {
		return nil
	}
	if it.FoodID == "" {
		it.FoodID = f.ID
	}
	if strings.TrimSpace(it.Name) == "" {
		it.Name = f.Name
	}
	kcal100, sod100 := nutrientsForFood(f)
	if kcal100 != nil && it.Kcal == 0 {
		it.Kcal = *kcal100 * it.Grams / 100
	}
	if sod100 != nil && it.SodiumMg == 0 {
		it.SodiumMg = *sod100 * it.Grams / 100
	}
	return nil
}

func scaleDishWithCatalog(it *MealItem, catalog []Food) error {
	var batch, k, n float64
	for i := range it.Parts {
		p := &it.Parts[i]
		if strings.TrimSpace(p.Name) == "" {
			continue
		}
		_ = scaleItemFromFoodWithCatalog(p, catalog)
		batch += p.Grams
		k += p.Kcal
		n += p.SodiumMg
	}
	factor := 1.0
	if batch <= 0 {
		it.Grams = 0
		it.Kcal = 0
		it.SodiumMg = 0
		return nil
	}
	eaten := it.Grams
	if eaten <= 0 || eaten >= batch-0.01 {
		it.Grams = batch
		factor = 1.0
	} else {
		factor = eaten / batch
	}
	it.Kcal = k * factor
	it.SodiumMg = n * factor
	return nil
}

type sqlExec interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func insertWeighInUser(exec sqlExec, userID string, w WeighIn) error {
	var rhr any
	if w.RHRBpm != nil {
		rhr = *w.RHRBpm
	}
	prot := 0
	if w.Protocol {
		prot = 1
	}
	_, err := exec.Exec(`INSERT INTO weigh_ins (user_id, id, date, kg, rhr_bpm, note, protocol, kg_raw)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, w.ID, w.Date, w.Kg, rhr, w.Note, prot, w.KgRaw)
	return err
}

func execFoodUpdate(exec sqlExec, userID string, f Food) error {
	_, err := exec.Exec(`UPDATE foods SET name=?, kcal_per_100g=?, sodium_mg_per_100g=?, notes=?,
		energy_kj_per_100g=?, fat_g_per_100g=?, saturates_g_per_100g=?, carbs_g_per_100g=?, sugars_g_per_100g=?, protein_g_per_100g=?, salt_g_per_100g=?
		WHERE user_id = ? AND id=?`,
		f.Name, nullFloat(f.KcalPer100g), nullFloat(f.SodiumMgPer100g), f.Notes,
		nullFloat(f.EnergyKjPer100g), nullFloat(f.FatGPer100g), nullFloat(f.SaturatesGPer100g),
		nullFloat(f.CarbsGPer100g), nullFloat(f.SugarsGPer100g), nullFloat(f.ProteinGPer100g), nullFloat(f.SaltGPer100g),
		userID, f.ID)
	return err
}

func insertFoodUser(exec sqlExec, userID string, f Food) error {
	NormalizeFoodLabel(&f)
	_, err := exec.Exec(`INSERT INTO foods (user_id, id, name, kcal_per_100g, sodium_mg_per_100g, notes,
		energy_kj_per_100g, fat_g_per_100g, saturates_g_per_100g, carbs_g_per_100g, sugars_g_per_100g, protein_g_per_100g, salt_g_per_100g)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, f.ID, f.Name, nullFloat(f.KcalPer100g), nullFloat(f.SodiumMgPer100g), f.Notes,
		nullFloat(f.EnergyKjPer100g), nullFloat(f.FatGPer100g), nullFloat(f.SaturatesGPer100g),
		nullFloat(f.CarbsGPer100g), nullFloat(f.SugarsGPer100g), nullFloat(f.ProteinGPer100g), nullFloat(f.SaltGPer100g))
	return err
}

func insertStockLotUser(exec sqlExec, userID string, lot StockLot) error {
	items, _ := json.Marshal(lot.Items)
	if items == nil {
		items = []byte("[]")
	}
	active := 0
	if stockLotIsActive(lot) {
		active = 1
	}
	_, err := exec.Exec(`INSERT INTO stock_lots (user_id, id, name, prepared_date, batch_grams, remaining_grams, active, items_json, notes, kcal_per_100g, sodium_mg_per_100g)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, lot.ID, lot.Name, nullStr(lot.PreparedDate), lot.BatchGrams, lot.RemainingGrams, active,
		string(items), lot.Notes, nullFloat(lot.KcalPer100g), nullFloat(lot.SodiumMgPer100g))
	return err
}
