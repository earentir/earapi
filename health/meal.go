package health

import (
	"fmt"
	"math"
	"strings"
)

// ApplyMealSlotMargin scales slot kcal/sodium by +marginPercent (e.g. 10 => multiply by 1.1).
// Deprecated for day totals: use ApplyDayMarginTotals on the summed catalog day instead.
func ApplyMealSlotMargin(meal *MealSlot, marginPercent float64) {
	if meal == nil || marginPercent <= 0 {
		return
	}
	f := 1 + marginPercent/100
	if meal.Kcal > 0 {
		meal.Kcal = math.Round(meal.Kcal * f)
	}
	if meal.SodiumMg > 0 {
		meal.SodiumMg = math.Round(meal.SodiumMg * f)
	}
}

// ApplyDayMarginTotals applies +margin% once to catalog day sums (after lunch+dinner+snack).
func ApplyDayMarginTotals(kcal, sodiumMg, marginPercent float64) (float64, float64) {
	if marginPercent <= 0 {
		return kcal, sodiumMg
	}
	f := 1 + marginPercent/100
	outK := kcal
	outS := sodiumMg
	if kcal > 0 {
		outK = math.Round(kcal * f)
	}
	if sodiumMg > 0 {
		outS = math.Round(sodiumMg * f)
	}
	return outK, outS
}

// FinalizeMealTotals fills slot sodium/kcal from items when not supplied.
func FinalizeMealTotals(meal *MealSlot) {
	if meal == nil || len(meal.Items) == 0 {
		return
	}
	var itemKcal, itemSod float64
	for _, it := range meal.Items {
		itemKcal += it.Kcal
		itemSod += it.SodiumMg
	}
	if meal.Kcal == 0 && itemKcal > 0 {
		meal.Kcal = itemKcal
	}
	if meal.SodiumMg == 0 && itemSod > 0 {
		meal.SodiumMg = itemSod
	}
	meal.KcalMargin = 0
}

// PrepareMealSlot ensures catalog entries, scales item nutrients, and finalizes slot totals.
func (s *Store) PrepareMealSlot(meal *MealSlot, ensureFoods []Food) ([]Food, error) {
	// Stock lots are opt-in per line (stock_lot_id set in UI); no auto-link from names.
	var created []Food
	for _, f := range ensureFoods {
		got, isNew, err := s.EnsureFood(f)
		if err != nil {
			return nil, err
		}
		if isNew {
			created = append(created, got)
		}
	}
	for i := range meal.Items {
		it := &meal.Items[i]
		if it.StockLotID != "" && len(it.Parts) > 0 {
			it.Parts = nil
		}
		added, err := s.ensureItemTree(it)
		if err != nil {
			return nil, err
		}
		created = append(created, added...)
		if err := s.ScaleItemFromFood(it); err != nil {
			return nil, err
		}
	}
	RecomputeMealSlotTotalsFromItems(meal)
	FinalizeMealTotals(meal)
	return created, nil
}

func (s *Store) ensureItemTree(it *MealItem) ([]Food, error) {
	added, err := s.maybeEnsureFoodFromItem(it)
	if err != nil {
		return nil, err
	}
	for i := range it.Parts {
		more, err := s.ensureItemTree(&it.Parts[i])
		if err != nil {
			return nil, err
		}
		added = append(added, more...)
	}
	return added, nil
}

func (s *Store) maybeEnsureFoodFromItem(it *MealItem) ([]Food, error) {
	if it.KcalPer100g == nil && it.SodiumMgPer100g == nil {
		return nil, nil
	}
	name := strings.TrimSpace(it.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name required to add food from item", ErrInvalidInput)
	}
	got, isNew, err := s.EnsureFood(Food{
		ID:              it.FoodID,
		Name:            name,
		KcalPer100g:     it.KcalPer100g,
		SodiumMgPer100g: it.SodiumMgPer100g,
	})
	if err != nil {
		return nil, err
	}
	if it.FoodID == "" {
		it.FoodID = got.ID
	}
	if isNew {
		return []Food{got}, nil
	}
	return nil, nil
}

// MealItemIsCatalogLinked is true when this line (or any part) has food_id or stock_lot_id.
func MealItemIsCatalogLinked(it *MealItem) bool {
	if it == nil {
		return false
	}
	if strings.TrimSpace(it.FoodID) != "" || strings.TrimSpace(it.StockLotID) != "" {
		return true
	}
	for i := range it.Parts {
		if MealItemIsCatalogLinked(&it.Parts[i]) {
			return true
		}
	}
	return false
}

// MealHasCatalogLinkedItems is true when any line is tied to Foods or Stock (structured Today UI logs).
func MealHasCatalogLinkedItems(m MealSlot) bool {
	var found bool
	var walk func(MealItem)
	walk = func(it MealItem) {
		if found {
			return
		}
		if len(it.Parts) > 0 {
			for i := range it.Parts {
				walk(it.Parts[i])
			}
			return
		}
		if strings.TrimSpace(it.FoodID) != "" || strings.TrimSpace(it.StockLotID) != "" {
			found = true
		}
	}
	for i := range m.Items {
		walk(m.Items[i])
	}
	return found
}

// DayHasCatalogLinkedItems reports whether any meal slot has catalog-linked structured lines.
func DayHasCatalogLinkedItems(d *Day) bool {
	if d == nil || d.Meals == nil {
		return false
	}
	for _, m := range d.Meals {
		if MealHasCatalogLinkedItems(m) {
			return true
		}
	}
	return false
}
