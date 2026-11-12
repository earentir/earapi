package health

import (
	"database/sql"
	"math"
)

func (s *Store) GetKcalBudgetSettings() KcalBudgetSettings {
	def := DefaultKcalBudgetSettings()
	row := s.db.QueryRow(`SELECT soft_low_kcal, soft_high_kcal, gain_min_kcal, meal_margin_percent FROM user_settings WHERE user_id = ?`, s.userID)
	var low, high, gain, margin sql.NullFloat64
	err := row.Scan(&low, &high, &gain, &margin)
	if err == sql.ErrNoRows {
		_ = s.ensureKcalBudgetDefaults()
		return def
	}
	if err != nil {
		return def
	}
	out := def
	if low.Valid && low.Float64 > 0 {
		out.SoftLowKcal = low.Float64
	}
	if high.Valid && high.Float64 > 0 {
		out.SoftHighKcal = high.Float64
	}
	if gain.Valid && gain.Float64 > 0 {
		out.GainMinKcal = gain.Float64
	}
	if margin.Valid && !math.IsNaN(margin.Float64) && margin.Float64 >= 0 {
		out.MealMarginPercent = margin.Float64
	}
	return out.normalized()
}

func (s *Store) PutKcalBudgetSettings(in KcalBudgetSettings) (KcalBudgetSettings, error) {
	in = in.normalized()
	if err := in.Validate(); err != nil {
		return KcalBudgetSettings{}, err
	}
	_, err := s.db.Exec(`INSERT INTO user_settings (user_id, soft_low_kcal, soft_high_kcal, gain_min_kcal, meal_margin_percent)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
		  soft_low_kcal=excluded.soft_low_kcal,
		  soft_high_kcal=excluded.soft_high_kcal,
		  gain_min_kcal=excluded.gain_min_kcal,
		  meal_margin_percent=excluded.meal_margin_percent`,
		s.userID, in.SoftLowKcal, in.SoftHighKcal, in.GainMinKcal, in.MealMarginPercent)
	if err != nil {
		return KcalBudgetSettings{}, err
	}
	return in, nil
}

func (s *Store) ensureKcalBudgetDefaults() error {
	def := DefaultKcalBudgetSettings()
	_, err := s.db.Exec(`INSERT OR IGNORE INTO user_settings (user_id, soft_low_kcal, soft_high_kcal, gain_min_kcal, meal_margin_percent)
		VALUES (?, ?, ?, ?, ?)`,
		s.userID, def.SoftLowKcal, def.SoftHighKcal, def.GainMinKcal, def.MealMarginPercent)
	return err
}
