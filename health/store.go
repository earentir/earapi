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
	mu       sync.Mutex
	db       *sql.DB
	userID   string
	userDir  string
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
func (s *Store) ReplaceImported(weights []WeighIn, foods []Food, days []Day) error {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	for _, d := range days {
		dd := d
		RecomputeDayTotals(&dd)
		if err := saveDayTxUser(tx, uid, &dd); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func nullFloat(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

func saveDayTxUser(tx *sql.Tx, userID string, d *Day) error {
	if d.Meals == nil {
		d.Meals = map[string]MealSlot{}
	}
	RecomputeDayTotals(d)
	_, err := tx.Exec(`INSERT INTO days (user_id, date, bp, walking, total_sodium_mg, total_kcal)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, date) DO UPDATE SET
		  bp=excluded.bp, walking=excluded.walking,
		  total_sodium_mg=excluded.total_sodium_mg, total_kcal=excluded.total_kcal`,
		userID, d.Date, d.BP, d.Walking, d.Totals.SodiumMg, d.Totals.Kcal)
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

// --- foods ---

func (s *Store) ListFoods() []Food {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT id, name, kcal_per_100g, sodium_mg_per_100g, notes FROM foods WHERE user_id = ? ORDER BY name`, s.userID)
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
	var kcal, sod sql.NullFloat64
	if err := rows.Scan(&f.ID, &f.Name, &kcal, &sod, &f.Notes); err != nil {
		return Food{}, false
	}
	if kcal.Valid {
		v := kcal.Float64
		f.KcalPer100g = &v
	}
	if sod.Valid {
		v := sod.Float64
		f.SodiumMgPer100g = &v
	}
	return f, true
}

func (s *Store) FoodByID(id string) (*Food, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.foodByIDUnlocked(id)
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
	if patch.Name != "" {
		cur.Name = strings.TrimSpace(patch.Name)
	}
	if patch.KcalPer100g != nil {
		cur.KcalPer100g = patch.KcalPer100g
	}
	if patch.SodiumMgPer100g != nil {
		cur.SodiumMgPer100g = patch.SodiumMgPer100g
	}
	if patch.Notes != "" {
		cur.Notes = patch.Notes
	}
	_, err := s.db.Exec(`UPDATE foods SET name=?, kcal_per_100g=?, sodium_mg_per_100g=?, notes=? WHERE user_id = ? AND id=?`,
		cur.Name, nullFloat(cur.KcalPer100g), nullFloat(cur.SodiumMgPer100g), cur.Notes, s.userID, id)
	if err != nil {
		return nil, err
	}
	return cur, nil
}

func (s *Store) foodByIDUnlocked(id string) (*Food, bool) {
	row := s.db.QueryRow(`SELECT id, name, kcal_per_100g, sodium_mg_per_100g, notes FROM foods WHERE user_id = ? AND id = ?`, s.userID, id)
	var f Food
	var kcal, sod sql.NullFloat64
	if err := row.Scan(&f.ID, &f.Name, &kcal, &sod, &f.Notes); err != nil {
		return nil, false
	}
	if kcal.Valid {
		v := kcal.Float64
		f.KcalPer100g = &v
	}
	if sod.Valid {
		v := sod.Float64
		f.SodiumMgPer100g = &v
	}
	return &f, true
}

// --- days ---

func (s *Store) readDayUnlocked(date string) (*Day, bool) {
	row := s.db.QueryRow(`SELECT date, bp, walking, total_sodium_mg, total_kcal FROM days WHERE user_id = ? AND date = ?`, s.userID, date)
	var d Day
	if err := row.Scan(&d.Date, &d.BP, &d.Walking, &d.Totals.SodiumMg, &d.Totals.Kcal); err != nil {
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
	return &d, true
}

func (s *Store) GetDay(date string) (*Day, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readDayUnlocked(date)
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
	return dates
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
	cur, exists := d.Meals[slot]
	if replace || !exists {
		d.Meals[slot] = meal
	} else {
		if meal.Raw != "" {
			if cur.Raw != "" {
				cur.Raw += " · " + meal.Raw
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
		if meal.KcalMargin > 0 {
			cur.KcalMargin = meal.KcalMargin
		}
		d.Meals[slot] = cur
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
	delete(d.Meals, slot)
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

// RecomputeDayTotals sums meal slots into day totals.
func RecomputeDayTotals(d *Day) {
	var sod, kc float64
	for _, slot := range []string{"lunch", "dinner", "snack"} {
		m, ok := d.Meals[slot]
		if !ok {
			continue
		}
		if len(m.Items) > 0 && (m.Kcal == 0 && m.SodiumMg == 0) {
			var is, ik float64
			for _, it := range m.Items {
				is += it.SodiumMg
				ik += it.Kcal
			}
			m.SodiumMg = is
			m.Kcal = ik
			if m.KcalMargin > 0 {
				m.Kcal = m.Kcal * (1 + m.KcalMargin)
			}
			d.Meals[slot] = m
		}
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
	var out []Day
	for rows.Next() {
		var dt string
		if rows.Scan(&dt) != nil {
			continue
		}
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
	if it.Kcal > 0 && it.SodiumMg > 0 {
		return nil
	}
	if it.Grams <= 0 {
		return nil
	}
	foods := s.ListFoods()
	f, ok := BestFoodMatch(foods, it.Name)
	if !ok || f == nil {
		return fmt.Errorf("%w: unknown food %q — add to catalog or pass kcal/sodium_mg", ErrInvalidInput, it.Name)
	}
	if f.KcalPer100g != nil && it.Kcal == 0 {
		it.Kcal = *f.KcalPer100g * it.Grams / 100
	}
	if f.SodiumMgPer100g != nil && it.SodiumMg == 0 {
		it.SodiumMg = *f.SodiumMgPer100g * it.Grams / 100
	}
	if it.Kcal == 0 && it.SodiumMg == 0 {
		return fmt.Errorf("%w: food %q has no per-100g data", ErrInvalidInput, it.Name)
	}
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

func insertFoodUser(exec sqlExec, userID string, f Food) error {
	_, err := exec.Exec(`INSERT INTO foods (user_id, id, name, kcal_per_100g, sodium_mg_per_100g, notes)
		VALUES (?, ?, ?, ?, ?, ?)`,
		userID, f.ID, f.Name, nullFloat(f.KcalPer100g), nullFloat(f.SodiumMgPer100g), f.Notes)
	return err
}
