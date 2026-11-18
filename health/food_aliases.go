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

// bestFoodByAliasUnlocked resolves meal shorthand via exact or fuzzy alias rows.
func (s *Store) bestFoodByAliasUnlocked(mealLine string) (*Food, bool) {
	mealLine = strings.TrimSpace(mealLine)
	if mealLine == "" {
		return nil, false
	}
	if f, ok := s.foodByAliasUnlocked(mealLine); ok {
		return f, true
	}
	rows, err := s.db.Query(`SELECT alias, food_id FROM food_aliases WHERE user_id = ?`, s.userID)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	type scored struct {
		foodID string
		sc     int
	}
	var ranked []scored
	for rows.Next() {
		var alias, foodID string
		if rows.Scan(&alias, &foodID) != nil {
			continue
		}
		if !catalogMatchesMealLine(alias, mealLine) {
			continue
		}
		sc := scoreNameForMealLine(alias, mealLine)
		if sc < 80 {
			continue
		}
		ranked = append(ranked, scored{foodID: foodID, sc: sc})
	}
	if err := rows.Err(); err != nil || len(ranked) == 0 {
		return nil, false
	}
	for i := 0; i < len(ranked); i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].sc > ranked[i].sc {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}
	best := ranked[0].sc
	foodIDs := map[string]bool{}
	for _, r := range ranked {
		if r.sc < best-15 {
			break
		}
		foodIDs[r.foodID] = true
	}
	if len(foodIDs) > 1 {
		return nil, false
	}
	f, ok := s.foodByIDUnlocked(ranked[0].foodID)
	if !ok {
		return nil, false
	}
	return f, true
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

// foodByIDOrAliasUnlocked resolves a catalog row by id or alias row keyed on that id (food replacement).
func (s *Store) foodByIDOrAliasUnlocked(id string) (*Food, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, false
	}
	if f, ok := s.foodByIDUnlocked(id); ok {
		return f, true
	}
	if f, ok := s.foodByAliasUnlocked(id); ok {
		return f, true
	}
	return nil, false
}

func (s *Store) execAddFoodAliasUnlocked(foodID, alias string) error {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return fmt.Errorf("%w: alias required", ErrInvalidInput)
	}
	if _, ok := s.foodByIDUnlocked(foodID); !ok {
		return ErrNotFound
	}
	norm := normalizeFoodAlias(alias)
	_, err := s.db.Exec(`INSERT INTO food_aliases (user_id, alias, food_id) VALUES (?, ?, ?)
		ON CONFLICT(user_id, alias) DO UPDATE SET food_id=excluded.food_id`,
		s.userID, norm, foodID)
	return err
}

func (s *Store) listAliasStringsForFoodUnlocked(foodID string) []string {
	rows, err := s.db.Query(`SELECT alias FROM food_aliases WHERE user_id = ? AND food_id = ?`, s.userID, foodID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var al string
		if rows.Scan(&al) == nil && strings.TrimSpace(al) != "" {
			out = append(out, al)
		}
	}
	return out
}

// transferFoodAliasRedirectsBeforeDelete points aliases and the old id/name at replaceWith.
func (s *Store) transferFoodAliasRedirectsBeforeDelete(oldID, replaceWith string) error {
	old, ok := s.foodByIDUnlocked(oldID)
	if !ok {
		return ErrNotFound
	}
	replaceWith = strings.TrimSpace(replaceWith)
	if replaceWith == "" {
		return nil
	}
	if replaceWith == oldID {
		return fmt.Errorf("%w: cannot replace food with itself", ErrInvalidInput)
	}
	if _, ok := s.foodByIDUnlocked(replaceWith); !ok {
		return fmt.Errorf("%w: replacement food not found", ErrNotFound)
	}
	for _, al := range s.listAliasStringsForFoodUnlocked(oldID) {
		if err := s.execAddFoodAliasUnlocked(replaceWith, al); err != nil {
			return err
		}
	}
	if err := s.execAddFoodAliasUnlocked(replaceWith, oldID); err != nil {
		return err
	}
	return s.execAddFoodAliasUnlocked(replaceWith, old.Name)
}

// AddFoodAlias links alias (e.g. meal shorthand) to an existing catalog food.
func (s *Store) AddFoodAlias(foodID, alias string) (FoodAlias, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.execAddFoodAliasUnlocked(foodID, alias); err != nil {
		return FoodAlias{}, err
	}
	return FoodAlias{Alias: normalizeFoodAlias(alias), FoodID: foodID}, nil
}

// DeleteFoodAlias removes a shorthand link for this catalog food.
func (s *Store) DeleteFoodAlias(foodID, alias string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return fmt.Errorf("%w: alias required", ErrInvalidInput)
	}
	if _, ok := s.foodByIDUnlocked(foodID); !ok {
		return ErrNotFound
	}
	norm := normalizeFoodAlias(alias)
	res, err := s.db.Exec(`DELETE FROM food_aliases WHERE user_id = ? AND food_id = ? AND alias = ? COLLATE NOCASE`,
		s.userID, foodID, norm)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListFoodAliasesForFood(foodID string) []FoodAlias {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT alias, food_id FROM food_aliases WHERE user_id = ? AND food_id = ? ORDER BY alias`,
		s.userID, foodID)
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
	_ = rows.Err()
	return out
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
