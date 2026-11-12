package health

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
)

// FoodAliasFromMD maps a meal-line name to a catalog food display name (resolved on import).
type FoodAliasFromMD struct {
	Alias      string
	CatalogFood string
}

func parseFoodAliasesMarkdown(content string) []FoodAliasFromMD {
	var out []FoodAliasFromMD
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") || strings.Contains(line, "---") {
			continue
		}
		cells := splitTableRow(line)
		if len(cells) < 2 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(cells[0]), "meal shorthand") {
			continue
		}
		alias := strings.TrimSpace(cells[0])
		target := strings.TrimSpace(cells[1])
		if alias == "" || target == "" {
			continue
		}
		out = append(out, FoodAliasFromMD{Alias: alias, CatalogFood: target})
	}
	return out
}

func LoadFoodAliasesFromMDDir(dir string) ([]FoodAliasFromMD, error) {
	path := filepath.Join(dir, "Health Tracker - Food aliases.md")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return parseFoodAliasesFromMD(string(b)), nil
}

func parseFoodAliasesFromMD(content string) []FoodAliasFromMD {
	return parseFoodAliasesMarkdown(content)
}

func foodIDByDisplayName(foods []Food, displayName string) (string, bool) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return "", false
	}
	q := strings.ToLower(displayName)
	for _, f := range foods {
		if strings.ToLower(strings.TrimSpace(f.Name)) == q {
			return f.ID, true
		}
	}
	if f, ok := BestFoodMatch(foods, displayName); ok {
		return f.ID, true
	}
	return "", false
}

// ApplyFoodAliasesFromMD seeds food_aliases from mddata for this user.
func (s *Store) ApplyFoodAliasesFromMD(dir string, foods []Food) (int, error) {
	specs, err := LoadFoodAliasesFromMDDir(dir)
	if err != nil {
		return 0, err
	}
	if len(specs) == 0 {
		return 0, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	applied := 0
	for _, sp := range specs {
		fid, ok := foodIDByDisplayName(foods, sp.CatalogFood)
		if !ok {
			continue
		}
		if _, ok := s.foodByIDUnlocked(fid); !ok {
			continue
		}
		norm := normalizeFoodAlias(sp.Alias)
		_, err := s.db.Exec(`INSERT INTO food_aliases (user_id, alias, food_id) VALUES (?, ?, ?)
			ON CONFLICT(user_id, alias) DO UPDATE SET food_id=excluded.food_id`,
			s.userID, norm, fid)
		if err != nil {
			return applied, err
		}
		applied++
	}
	return applied, nil
}

func (s *Store) applyFoodAliasesTx(tx *sql.Tx, uid string, foods []Food, specs []FoodAliasFromMD) error {
	for _, sp := range specs {
		fid, ok := foodIDByDisplayName(foods, sp.CatalogFood)
		if !ok {
			continue
		}
		norm := normalizeFoodAlias(sp.Alias)
		if _, err := tx.Exec(`INSERT INTO food_aliases (user_id, alias, food_id) VALUES (?, ?, ?)
			ON CONFLICT(user_id, alias) DO UPDATE SET food_id=excluded.food_id`,
			uid, norm, fid); err != nil {
			return err
		}
	}
	return nil
}
