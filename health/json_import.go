package health

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func loadLegacyJSONBundle(dir string) (JSONBundle, error) {
	if _, err := os.Stat(filepath.Join(dir, "health.bundle.json")); err == nil {
		return LoadJSONBundleFile(filepath.Join(dir, "health.bundle.json"))
	}
	return LoadJSONBundleFromDir(dir)
}

// JSONBundle is a single-file export/import format (v1).
type JSONBundle struct {
	Version     int                 `json:"version,omitempty"`
	ExportedAt  string              `json:"exported_at,omitempty"`
	Scope       string              `json:"scope,omitempty"`
	Settings    *KcalBudgetSettings `json:"settings,omitempty"`
	Weights     []WeighIn           `json:"weights"`
	Foods       []Food              `json:"foods"`
	FoodAliases []FoodAlias         `json:"food_aliases,omitempty"`
	Days        []Day               `json:"days"`
	Stock       []StockLot          `json:"stock,omitempty"`
}

// LoadJSONBundleFromDir reads weight.json, foods.json, and days/*.json from dir.
func LoadJSONBundleFromDir(dir string) (JSONBundle, error) {
	var bundle JSONBundle
	wPath := filepath.Join(dir, "weight.json")
	if b, err := os.ReadFile(wPath); err == nil {
		if err := json.Unmarshal(b, &bundle.Weights); err != nil {
			return bundle, fmt.Errorf("weight.json: %w", err)
		}
	}
	fPath := filepath.Join(dir, "foods.json")
	if b, err := os.ReadFile(fPath); err == nil {
		if err := json.Unmarshal(b, &bundle.Foods); err != nil {
			return bundle, fmt.Errorf("foods.json: %w", err)
		}
	}
	daysDir := filepath.Join(dir, "days")
	if ents, err := os.ReadDir(daysDir); err == nil {
		for _, e := range ents {
			if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
				continue
			}
			b, err := os.ReadFile(filepath.Join(daysDir, e.Name()))
			if err != nil {
				continue
			}
			var d Day
			if err := json.Unmarshal(b, &d); err != nil {
				return bundle, fmt.Errorf("days/%s: %w", e.Name(), err)
			}
			if d.Date == "" {
				d.Date = strings.TrimSuffix(e.Name(), ".json")
			}
			bundle.Days = append(bundle.Days, d)
		}
	}
	return bundle, nil
}

// LoadJSONBundleFile reads health.bundle.json (or any path) with weights, foods, days.
func LoadJSONBundleFile(path string) (JSONBundle, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return JSONBundle{}, err
	}
	var bundle JSONBundle
	if err := json.Unmarshal(b, &bundle); err != nil {
		return JSONBundle{}, err
	}
	return bundle, nil
}

// ImportBundle writes bundle into the store (replace or merge, scope all).
func (s *Store) ImportBundle(bundle JSONBundle, merge bool) (map[string]int, error) {
	return s.ImportBundleScope(bundle, "all", merge)
}

func (s *Store) mergeBundle(bundle JSONBundle) (map[string]int, error) {
	counts := map[string]int{}
	for _, w := range bundle.Weights {
		if w.ID == "" {
			_, err := s.AddWeight(w)
			if err != nil {
				return counts, err
			}
		} else {
			// skip duplicate ids
			list := s.ListWeights("", "")
			exists := false
			for _, x := range list {
				if x.ID == w.ID {
					exists = true
					break
				}
			}
			if !exists {
				s.mu.Lock()
				err := insertWeighInUser(s.db, s.userID, w)
				s.mu.Unlock()
				if err != nil {
					return counts, err
				}
			}
		}
		counts["weights"]++
	}
	for _, f := range bundle.Foods {
		if _, err := s.AddFood(f); err != nil {
			if err == ErrConflict {
				_, _ = s.UpdateFood(f.ID, f)
			} else {
				return counts, err
			}
		}
		counts["foods"]++
	}
	for _, d := range bundle.Days {
		if err := s.PutDay(dayForImport(d)); err != nil {
			return counts, err
		}
		counts["days"]++
	}
	for _, lot := range bundle.Stock {
		if _, err := s.UpsertStockLot(lot); err != nil {
			return counts, err
		}
		counts["stock"]++
	}
	for _, a := range bundle.FoodAliases {
		if _, err := s.AddFoodAlias(a.FoodID, a.Alias); err != nil {
			return counts, err
		}
		counts["food_aliases"]++
	}
	if bundle.Settings != nil {
		if _, err := s.PutKcalBudgetSettings(*bundle.Settings); err != nil {
			return counts, err
		}
		counts["settings"] = 1
	}
	return counts, nil
}

// LegacyJSONPresent reports whether legacy JSON files exist in dir.
func LegacyJSONPresent(dir string) bool {
	for _, name := range []string{"weight.json", "foods.json", "health.bundle.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	if ents, err := os.ReadDir(filepath.Join(dir, "days")); err == nil {
		for _, e := range ents {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
				return true
			}
		}
	}
	return false
}
