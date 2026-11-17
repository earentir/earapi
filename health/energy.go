package health

import (
	"fmt"
	"math"
	"strings"
)

// SedentaryPAL is Mifflin-St Jeor BMR × 1.2 (little or no exercise).
const SedentaryPAL = 1.2

func normalizeSex(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "m", "male", "man":
		return "male"
	case "f", "female", "woman":
		return "female"
	default:
		return ""
	}
}

// MifflinStJeorBMR is resting kcal/day (Mifflin-St Jeor, 1990).
func MifflinStJeorBMR(kg, heightCm float64, ageYears int, sex string) (float64, bool) {
	if kg <= 0 || heightCm <= 0 || ageYears <= 0 {
		return 0, false
	}
	var sexAdj float64
	switch normalizeSex(sex) {
	case "male":
		sexAdj = 5
	case "female":
		sexAdj = -161
	default:
		return 0, false
	}
	bmr := 10*kg + 6.25*heightCm - 5*float64(ageYears) + sexAdj
	if bmr <= 0 || math.IsNaN(bmr) {
		return 0, false
	}
	return math.Round(bmr), true
}

func sedentaryTDEE(bmr float64) float64 {
	return sedentaryTDEEWithDiscount(bmr, 0)
}

func sedentaryTDEEWithDiscount(bmr, discountPercent float64) float64 {
	if bmr <= 0 {
		return 0
	}
	disc := discountPercent
	if disc < 0 || math.IsNaN(disc) {
		disc = 0
	}
	if disc > 50 {
		disc = 50
	}
	base := bmr * SedentaryPAL
	return math.Round(base * (1 - disc/100))
}

func attachEnergyEstimate(s *KcalBudgetSettings, kg float64) {
	if s == nil {
		return
	}
	s.BmrKcal = 0
	s.SedentaryKcal = 0
	s.EnergyWeightKg = 0
	bmr, ok := MifflinStJeorBMR(kg, s.HeightCm, s.AgeYears, s.Sex)
	if !ok {
		return
	}
	s.BmrKcal = bmr
	s.SedentaryKcal = sedentaryTDEEWithDiscount(bmr, s.SedentaryDiscountPercent)
	s.EnergyWeightKg = kg
}

func (s KcalBudgetSettings) validateEnergyProfile() error {
	if strings.TrimSpace(s.Sex) != "" && normalizeSex(s.Sex) == "" {
		return fmt.Errorf("%w: sex must be male or female", ErrInvalidInput)
	}
	if s.AgeYears != 0 && (s.AgeYears < 12 || s.AgeYears > 120) {
		return fmt.Errorf("%w: age_years must be 12-120", ErrInvalidInput)
	}
	if s.HeightCm != 0 && (s.HeightCm < 80 || s.HeightCm > 250) {
		return fmt.Errorf("%w: height_cm must be 80-250", ErrInvalidInput)
	}
	return nil
}
