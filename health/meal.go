package health

import (
	"fmt"
	"strings"
)

// FinalizeMealTotals fills slot sodium/kcal from items when not supplied, then applies kcal_margin.
func FinalizeMealTotals(meal *MealSlot) {
	if meal == nil || len(meal.Items) == 0 {
		return
	}
	var itemKcal, itemSod float64
	for _, it := range meal.Items {
		itemKcal += it.Kcal
		itemSod += it.SodiumMg
	}
	computedKcal := false
	if meal.Kcal == 0 && itemKcal > 0 {
		meal.Kcal = itemKcal
		computedKcal = true
	}
	if meal.SodiumMg == 0 && itemSod > 0 {
		meal.SodiumMg = itemSod
	}
	if computedKcal && meal.KcalMargin > 0 {
		meal.Kcal = meal.Kcal * (1 + meal.KcalMargin)
	}
}

// PrepareMealSlot ensures catalog entries, scales item nutrients, and finalizes slot totals.
func (s *Store) PrepareMealSlot(meal *MealSlot, ensureFoods []Food) error {
	for _, f := range ensureFoods {
		if _, err := s.EnsureFood(f); err != nil {
			return err
		}
	}
	for i := range meal.Items {
		it := &meal.Items[i]
		if err := s.ensureItemTree(it); err != nil {
			return err
		}
		if err := s.ScaleItemFromFood(it); err != nil {
			return err
		}
	}
	FinalizeMealTotals(meal)
	return nil
}

func (s *Store) ensureItemTree(it *MealItem) error {
	if err := s.maybeEnsureFoodFromItem(it); err != nil {
		return err
	}
	for i := range it.Parts {
		if err := s.ensureItemTree(&it.Parts[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) maybeEnsureFoodFromItem(it *MealItem) error {
	if it.KcalPer100g == nil && it.SodiumMgPer100g == nil {
		return nil
	}
	name := strings.TrimSpace(it.Name)
	if name == "" {
		return fmt.Errorf("%w: name required to add food from item", ErrInvalidInput)
	}
	_, err := s.EnsureFood(Food{
		ID:              it.FoodID,
		Name:            name,
		KcalPer100g:     it.KcalPer100g,
		SodiumMgPer100g: it.SodiumMgPer100g,
	})
	return err
}
