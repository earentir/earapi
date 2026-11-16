package health

import (
	"fmt"
	"strings"
	"time"
)

const healthBundleVersion = 1

func bundleHasData(b JSONBundle) bool {
	return bundleHasDataScope(b, "all")
}

func bundleHasDataScope(b JSONBundle, scope string) bool {
	switch normalizeExportScope(scope) {
	case "foods":
		return len(b.Foods) > 0 || len(b.FoodAliases) > 0
	case "days":
		return len(b.Days) > 0
	case "stock":
		return len(b.Stock) > 0
	case "weight":
		return len(b.Weights) > 0
	default:
		return len(b.Weights) > 0 || len(b.Foods) > 0 || len(b.Days) > 0 || len(b.Stock) > 0 ||
			len(b.FoodAliases) > 0 || b.Settings != nil
	}
}

func normalizeExportScope(scope string) string {
	scope = strings.ToLower(strings.TrimSpace(scope))
	switch scope {
	case "food", "foods", "catalog":
		return "foods"
	case "day", "days":
		return "days"
	case "stock", "lots":
		return "stock"
	case "weight", "weights", "weigh":
		return "weight"
	case "all", "full", "bundle", "":
		return "all"
	default:
		return scope
	}
}

func dayForImport(d Day) Day {
	d.Display = nil
	return d
}

func dayForExport(d Day) Day {
	d.Display = nil
	return d
}

// ExportBundle returns JSON for scope: all, foods, days, stock, weight.
func (s *Store) ExportBundle(scope string) (JSONBundle, error) {
	scope = normalizeExportScope(scope)
	out := JSONBundle{
		Version:    healthBundleVersion,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Scope:      scope,
	}
	switch scope {
	case "foods":
		out.Foods = s.ListFoods()
		out.FoodAliases = s.ListFoodAliases()
	case "days":
		for _, d := range s.FilterDaysComputed("", "", "", "", "", 0) {
			out.Days = append(out.Days, dayForExport(d))
		}
	case "stock":
		lots, err := s.ListStockLots(false)
		if err != nil {
			return out, err
		}
		for _, lot := range lots {
			if full, ok := s.StockLotByID(lot.ID); ok {
				out.Stock = append(out.Stock, *full)
			} else {
				out.Stock = append(out.Stock, lot)
			}
		}
	case "weight":
		out.Weights = s.ListWeights("", "")
	case "all":
		out.Foods = s.ListFoods()
		out.FoodAliases = s.ListFoodAliases()
		for _, d := range s.FilterDaysComputed("", "", "", "", "", 0) {
			out.Days = append(out.Days, dayForExport(d))
		}
		out.Weights = s.ListWeights("", "")
		lots, err := s.ListStockLots(false)
		if err != nil {
			return out, err
		}
		for _, lot := range lots {
			if full, ok := s.StockLotByID(lot.ID); ok {
				out.Stock = append(out.Stock, *full)
			} else {
				out.Stock = append(out.Stock, lot)
			}
		}
		settings := s.GetKcalBudgetSettings()
		out.Settings = &settings
	default:
		return out, fmt.Errorf("%w: unknown export scope %q", ErrInvalidInput, scope)
	}
	return out, nil
}

// ImportBundleScope applies only the section matching scope (merge=true upserts; merge=false replace only for scope all).
func (s *Store) ImportBundleScope(bundle JSONBundle, scope string, merge bool) (map[string]int, error) {
	scope = normalizeExportScope(scope)
	if !bundleHasDataScope(bundle, scope) {
		return nil, fmt.Errorf("%w: no data for scope %s", ErrInvalidInput, scope)
	}
	if scope == "all" && !merge {
		if err := s.ReplaceImported("", bundle.Weights, bundle.Foods, bundle.Days, bundle.Stock); err != nil {
			return nil, err
		}
		counts := map[string]int{
			"weights": len(bundle.Weights),
			"foods":   len(bundle.Foods),
			"days":    len(bundle.Days),
			"stock":   len(bundle.Stock),
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
	if scope == "all" && merge {
		return s.mergeBundle(bundle)
	}
	if !merge {
		return nil, fmt.Errorf("%w: scoped import requires merge=true (scope=%s)", ErrInvalidInput, scope)
	}
	return s.mergeBundleScope(bundle, scope)
}

func (s *Store) mergeBundleScope(bundle JSONBundle, scope string) (map[string]int, error) {
	trim := JSONBundle{}
	switch scope {
	case "foods":
		trim.Foods = bundle.Foods
		trim.FoodAliases = bundle.FoodAliases
	case "days":
		trim.Days = bundle.Days
	case "stock":
		trim.Stock = bundle.Stock
	case "weight":
		trim.Weights = bundle.Weights
	default:
		return s.mergeBundle(bundle)
	}
	return s.mergeBundle(trim)
}