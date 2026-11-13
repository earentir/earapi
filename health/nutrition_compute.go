package health

import (
	"encoding/json"
	"fmt"
)

// ComputeDayResult is returned from POST /health/v1/nutrition/compute/day.
type ComputeDayResult struct {
	Day     *Day       `json:"day"`
	Display DayDisplay `json:"display"`
}

// ComputeMealResult is returned from POST /health/v1/nutrition/compute/meal.
type ComputeMealResult struct {
	Slot    string     `json:"slot"`
	Meal    MealSlot   `json:"meal"`
	Display DayDisplay `json:"display,omitempty"` // optional day-level preview when date+legacy present
}

func cloneDayJSON(d *Day) (*Day, error) {
	if d == nil {
		return &Day{Meals: map[string]MealSlot{}}, nil
	}
	b, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	var out Day
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	if out.Meals == nil {
		out.Meals = map[string]MealSlot{}
	}
	return &out, nil
}

// ComputeDayNutrition runs the same pipeline as GET /days/:date (scale, calibrate, margin, day totals).
func (s *Store) ComputeDayNutrition(d *Day) (*Day, error) {
	cp, err := cloneDayJSON(d)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.enrichDayTotalsUnlocked(cp)
	s.mu.Unlock()
	return cp, nil
}

func (s *Store) computeDayWithDisplay(d *Day) (ComputeDayResult, error) {
	out := ComputeDayResult{}
	if d == nil {
		return out, ErrInvalidInput
	}
	cp, err := s.ComputeDayNutrition(d)
	if err != nil {
		return out, err
	}
	gaps := s.itemGapsForDay(cp, 200)
	settings := s.GetKcalBudgetSettings()
	out.Day = cp
	out.Display = dayDisplayFor(cp, settings, gaps)
	return out, nil
}

// ComputeMealSlotNutrition scales one meal slot (PrepareMealSlot + slot margin rules on a temp day).
func (s *Store) ComputeMealSlotNutrition(date, slot string, meal MealSlot, legacy DayTotals) (ComputeMealResult, error) {
	if slot == "" {
		return ComputeMealResult{}, fmt.Errorf("%w: slot required", ErrInvalidInput)
	}
	mealCopy := meal
	created, err := s.PrepareMealSlot(&mealCopy, nil)
	if err != nil {
		return ComputeMealResult{}, err
	}
	_ = created
	d := &Day{
		Date: date,
		Meals: map[string]MealSlot{
			slot: mealCopy,
		},
		LegacyTotals: legacy,
	}
	margin := s.GetKcalBudgetSettings().MealMarginPercent
	RecomputeDayTotalsWithMargin(d, margin)
	mealCopy = d.Meals[slot]
	gaps := s.itemGapsForMeal(date, slot, mealCopy, 50)
	settings := s.GetKcalBudgetSettings()
	return ComputeMealResult{
		Slot: slot,
		Meal: mealCopy,
		Display: dayDisplayFor(&Day{
			Date:         date,
			Meals:        d.Meals,
			Totals:       d.Totals,
			LegacyTotals: legacy,
		}, settings, gaps),
	}, nil
}

// attachDayDisplay must be called with s.mu held (same as enrichDayTotalsUnlocked).
func (s *Store) attachDayDisplay(d *Day) {
	if d == nil {
		return
	}
	gaps := collectItemGaps(d, s.listFoodsUnlocked(), s, 200)
	settings := s.GetKcalBudgetSettings()
	disp := dayDisplayFor(d, settings, gaps)
	d.Display = &disp
}
