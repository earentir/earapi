package health

import (
	"fmt"
	"math"
)

// KcalBudgetSettings defines daily intake bands for the UI kcal bar (per user).
// Defaults match "Energy baseline (Oct 12 FINAL)" in mddata Health Tracker - Misc.md:
// intake band 1,500–1,700 kcal; ≥2,200 kcal = gain territory.
type KcalBudgetSettings struct {
	SoftLowKcal       float64 `json:"soft_low_kcal"`
	SoftHighKcal      float64 `json:"soft_high_kcal"`
	GainMinKcal       float64 `json:"gain_min_kcal"`
	MealMarginPercent float64 `json:"meal_margin_percent"` // +% on each meal slot total (lunch/dinner/snack), not per line
}

const DefaultMealMarginPercent = 10

// DefaultKcalBudgetSettings is used when seeding a new user row.
func DefaultKcalBudgetSettings() KcalBudgetSettings {
	return KcalBudgetSettings{
		SoftLowKcal:       1500,
		SoftHighKcal:      1700,
		GainMinKcal:       2200,
		MealMarginPercent: DefaultMealMarginPercent,
	}
}

func (s KcalBudgetSettings) normalized() KcalBudgetSettings {
	d := DefaultKcalBudgetSettings()
	if s.SoftLowKcal <= 0 {
		s.SoftLowKcal = d.SoftLowKcal
	}
	if s.SoftHighKcal <= 0 {
		s.SoftHighKcal = d.SoftHighKcal
	}
	if s.GainMinKcal <= 0 {
		s.GainMinKcal = d.GainMinKcal
	}
	if s.MealMarginPercent < 0 || math.IsNaN(s.MealMarginPercent) {
		s.MealMarginPercent = d.MealMarginPercent
	}
	return s
}

// Validate checks ordering; callers may coerce with normalized() first.
func (s KcalBudgetSettings) Validate() error {
	s = s.normalized()
	if !(s.SoftLowKcal < s.SoftHighKcal && s.SoftHighKcal < s.GainMinKcal) {
		return fmt.Errorf("%w: need soft_low < soft_high < gain_min", ErrInvalidInput)
	}
	if math.IsNaN(s.SoftLowKcal) || math.IsNaN(s.SoftHighKcal) || math.IsNaN(s.GainMinKcal) {
		return fmt.Errorf("%w: invalid kcal limits", ErrInvalidInput)
	}
	if s.MealMarginPercent > 100 {
		return fmt.Errorf("%w: meal_margin_percent must be at most 100", ErrInvalidInput)
	}
	return nil
}
