package health

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// StockMovement is one line in a lot's usage ledger (grams out of stock; negative = put back).
type StockMovement struct {
	ID        string  `json:"id"`
	LotID     string  `json:"lot_id"`
	CreatedAt string  `json:"created_at"`
	MealDate  string  `json:"meal_date,omitempty"`
	MealSlot  string  `json:"meal_slot,omitempty"`
	Grams     float64 `json:"grams"`
	Kind      string  `json:"kind"` // meal, household, discarded, import, manual
	Note      string  `json:"note,omitempty"`
}

func stockMovementLabel(m StockMovement) string {
	switch m.Kind {
	case "meal":
		if m.MealDate != "" && m.MealSlot != "" {
			return fmt.Sprintf("%s %s", m.MealDate, m.MealSlot)
		}
		return "meal"
	case "household":
		return "household"
	case "discarded":
		return "discarded"
	case "import":
		return "before import"
	case "manual":
		return "manual"
	default:
		return m.Kind
	}
}

func (s *Store) listStockMovementsUnlocked(lotID string) ([]StockMovement, error) {
	rows, err := s.db.Query(`SELECT id, lot_id, created_at, meal_date, meal_slot, grams, kind, note
		FROM stock_movements WHERE user_id = ? AND lot_id = ? ORDER BY created_at ASC, id ASC`,
		s.userID, lotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StockMovement
	for rows.Next() {
		var m StockMovement
		var mealDate, mealSlot, note sql.NullString
		if err := rows.Scan(&m.ID, &m.LotID, &m.CreatedAt, &mealDate, &mealSlot, &m.Grams, &m.Kind, &note); err != nil {
			return nil, err
		}
		if mealDate.Valid {
			m.MealDate = mealDate.String
		}
		if mealSlot.Valid {
			m.MealSlot = mealSlot.String
		}
		if note.Valid {
			m.Note = note.String
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) listAllStockMovementsUnlocked() (map[string][]StockMovement, error) {
	rows, err := s.db.Query(`SELECT id, lot_id, created_at, meal_date, meal_slot, grams, kind, note
		FROM stock_movements WHERE user_id = ? ORDER BY lot_id, created_at ASC, id ASC`, s.userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]StockMovement{}
	for rows.Next() {
		var m StockMovement
		var mealDate, mealSlot, note sql.NullString
		if err := rows.Scan(&m.ID, &m.LotID, &m.CreatedAt, &mealDate, &mealSlot, &m.Grams, &m.Kind, &note); err != nil {
			return nil, err
		}
		if mealDate.Valid {
			m.MealDate = mealDate.String
		}
		if mealSlot.Valid {
			m.MealSlot = mealSlot.String
		}
		if note.Valid {
			m.Note = note.String
		}
		out[m.LotID] = append(out[m.LotID], m)
	}
	return out, rows.Err()
}

func (s *Store) recordStockMovementUnlocked(lotID string, grams float64, kind, mealDate, mealSlot, note string) error {
	if grams <= 0.01 && grams >= -0.01 {
		return nil
	}
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" {
		kind = "manual"
	}
	taken := map[string]bool{}
	rows, _ := s.db.Query(`SELECT id FROM stock_movements WHERE user_id = ? AND lot_id = ?`, s.userID, lotID)
	if rows != nil {
		for rows.Next() {
			var id string
			_ = rows.Scan(&id)
			taken[id] = true
		}
		rows.Close()
	}
	base := fmt.Sprintf("%s-%s-%d", lotID, kind, time.Now().UnixNano())
	id := UniqueSlug(base, taken)
	_, err := s.db.Exec(`INSERT INTO stock_movements (user_id, id, lot_id, created_at, meal_date, meal_slot, grams, kind, note)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.userID, id, lotID, time.Now().UTC().Format(time.RFC3339), nullStr(mealDate), nullStr(mealSlot), grams, kind, nullStr(note))
	return err
}

func (s *Store) recomputeStockRemainingFromLedgerUnlocked(lotID string) error {
	lot, ok := s.stockLotByIDUnlocked(lotID)
	if !ok {
		return ErrNotFound
	}
	var used float64
	err := s.db.QueryRow(`SELECT COALESCE(SUM(grams),0) FROM stock_movements WHERE user_id = ? AND lot_id = ?`,
		s.userID, lotID).Scan(&used)
	if err != nil {
		return err
	}
	rem := lot.BatchGrams - used
	if rem < 0 {
		rem = 0
	}
	active := stockLotIsActive(*lot)
	if rem <= 0.01 {
		active = false
	} else if lot.Active != nil && !*lot.Active {
		// keep user-marked finished unless grams came back
		active = false
	} else if rem > 0.01 {
		active = true
	}
	return s.applyStockRemainingUnlocked(lotID, rem, active)
}

func (s *Store) applyStockUsageUnlocked(lotID string, grams float64, kind, mealDate, mealSlot, note string) error {
	lot, ok := s.stockLotByIDUnlocked(lotID)
	if !ok {
		return ErrNotFound
	}
	var used float64
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(grams),0) FROM stock_movements WHERE user_id = ? AND lot_id = ?`,
		s.userID, lotID).Scan(&used); err != nil {
		return err
	}
	if used+grams > lot.BatchGrams+0.01 {
		left := lot.BatchGrams - used
		if left < 0 {
			left = 0
		}
		return fmt.Errorf("%w: only %.0f g left in stock", ErrInvalidInput, left)
	}
	if err := s.recordStockMovementUnlocked(lotID, grams, kind, mealDate, mealSlot, note); err != nil {
		return err
	}
	return s.recomputeStockRemainingFromLedgerUnlocked(lotID)
}

func (s *Store) seedStockImportUsageUnlocked(lot StockLot) error {
	used := lot.BatchGrams - lot.RemainingGrams
	if used <= 0.01 {
		return nil
	}
	note := "Used before ledger (Misc / import)"
	return s.recordStockMovementUnlocked(lot.ID, used, "import", "", "", note)
}

func insertStockMovementUser(exec sqlExec, userID string, m StockMovement) error {
	_, err := exec.Exec(`INSERT INTO stock_movements (user_id, id, lot_id, created_at, meal_date, meal_slot, grams, kind, note)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, m.ID, m.LotID, m.CreatedAt, nullStr(m.MealDate), nullStr(m.MealSlot), m.Grams, m.Kind, nullStr(m.Note))
	return err
}
