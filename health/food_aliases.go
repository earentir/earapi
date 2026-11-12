package health

import (
	"database/sql"
	"fmt"
	"strings"
)

// FoodAlias maps a meal-line shorthand to a catalog food id.
type FoodAlias struct {
	Alias  string `json:"alias"`
	FoodID string `json:"food_id"`
}

func normalizeFoodAlias(alias string) string {
	return strings.ToLower(strings.TrimSpace(alias))
}

func (s *Store) ListFoodAliases() []FoodAlias {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT alias, food_id FROM food_aliases WHERE user_id = ? ORDER BY alias`, s.userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []FoodAlias
	for rows.Next() {
		var a FoodAlias
		if err := rows.Scan(&a.Alias, &a.FoodID); err != nil {
			continue
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil
	}
	return out
}

func (s *Store) foodByAliasUnlocked(alias string) (*Food, bool) {
	alias = normalizeFoodAlias(alias)
	if alias == "" {
		return nil, false
	}
	var foodID string
	err := s.db.QueryRow(`SELECT food_id FROM food_aliases WHERE user_id = ? AND alias = ? COLLATE NOCASE`,
		s.userID, alias).Scan(&foodID)
	if err != nil {
		return nil, false
	}
	return s.foodByIDUnlocked(foodID)
}

// AddFoodAlias links alias (e.g. meal shorthand) to an existing catalog food.
func (s *Store) AddFoodAlias(foodID, alias string) (FoodAlias, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return FoodAlias{}, fmt.Errorf("%w: alias required", ErrInvalidInput)
	}
	if _, ok := s.foodByIDUnlocked(foodID); !ok {
		return FoodAlias{}, ErrNotFound
	}
	norm := normalizeFoodAlias(alias)
	_, err := s.db.Exec(`INSERT INTO food_aliases (user_id, alias, food_id) VALUES (?, ?, ?)
		ON CONFLICT(user_id, alias) DO UPDATE SET food_id=excluded.food_id`,
		s.userID, norm, foodID)
	if err != nil {
		return FoodAlias{}, err
	}
	return FoodAlias{Alias: norm, FoodID: foodID}, nil
}

func ensureFoodAliasTable(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS food_aliases (
	user_id TEXT NOT NULL,
	alias TEXT NOT NULL,
	food_id TEXT NOT NULL,
	PRIMARY KEY (user_id, alias),
	FOREIGN KEY (user_id, food_id) REFERENCES foods(user_id, id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_food_aliases_food ON food_aliases(user_id, food_id);`)
	return err
}
