package health

import (
	"database/sql"
)

func (s *Store) GetKcalBudgetSettings() KcalBudgetSettings {
	def := DefaultKcalBudgetSettings()
	row := s.db.QueryRow(`SELECT soft_low_kcal, soft_high_kcal, gain_min_kcal FROM user_settings WHERE user_id = ?`, s.userID)
	var low, high, gain sql.NullFloat64
	err := row.Scan(&low, &high, &gain)
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
	return out.normalized()
}

func (s *Store) PutKcalBudgetSettings(in KcalBudgetSettings) (KcalBudgetSettings, error) {
	in = in.normalized()
	if err := in.Validate(); err != nil {
		return KcalBudgetSettings{}, err
	}
	_, err := s.db.Exec(`INSERT INTO user_settings (user_id, soft_low_kcal, soft_high_kcal, gain_min_kcal)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
		  soft_low_kcal=excluded.soft_low_kcal,
		  soft_high_kcal=excluded.soft_high_kcal,
		  gain_min_kcal=excluded.gain_min_kcal`,
		s.userID, in.SoftLowKcal, in.SoftHighKcal, in.GainMinKcal)
	if err != nil {
		return KcalBudgetSettings{}, err
	}
	return in, nil
}

func (s *Store) ensureKcalBudgetDefaults() error {
	def := DefaultKcalBudgetSettings()
	_, err := s.db.Exec(`INSERT OR IGNORE INTO user_settings (user_id, soft_low_kcal, soft_high_kcal, gain_min_kcal)
		VALUES (?, ?, ?, ?)`,
		s.userID, def.SoftLowKcal, def.SoftHighKcal, def.GainMinKcal)
	return err
}
