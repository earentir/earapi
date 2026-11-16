package health

import (
	"database/sql"
	"math"
)

func (s *Store) GetKcalBudgetSettings() KcalBudgetSettings {
	def := DefaultKcalBudgetSettings()
	row := s.db.QueryRow(`SELECT soft_low_kcal, soft_high_kcal, gain_min_kcal,
		soft_low_sodium_mg, soft_high_sodium_mg, ceiling_sodium_mg,
		meal_margin_percent,
		legacy_diff_green_lt, legacy_diff_yellow_lt, legacy_diff_orange_lt,
		sex, age_years, height_cm FROM user_settings WHERE user_id = ?`, s.userID)
	var low, high, gain, naLow, naHigh, naCeil, margin, diffG, diffY, diffO, height sql.NullFloat64
	var sex sql.NullString
	var age sql.NullInt64
	err := row.Scan(&low, &high, &gain, &naLow, &naHigh, &naCeil, &margin, &diffG, &diffY, &diffO, &sex, &age, &height)
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
	if naLow.Valid && naLow.Float64 > 0 {
		out.SoftLowSodiumMg = naLow.Float64
	}
	if naHigh.Valid && naHigh.Float64 > 0 {
		out.SoftHighSodiumMg = naHigh.Float64
	}
	if naCeil.Valid && naCeil.Float64 > 0 {
		out.CeilingSodiumMg = naCeil.Float64
	}
	if margin.Valid && !math.IsNaN(margin.Float64) && margin.Float64 >= 0 {
		out.MealMarginPercent = margin.Float64
	}
	if diffG.Valid && diffG.Float64 > 0 {
		out.LegacyDiffGreenLt = diffG.Float64
	}
	if diffY.Valid && diffY.Float64 > 0 {
		out.LegacyDiffYellowLt = diffY.Float64
	}
	if diffO.Valid && diffO.Float64 > 0 {
		out.LegacyDiffOrangeLt = diffO.Float64
	}
	if sex.Valid {
		out.Sex = sex.String
	}
	if age.Valid && age.Int64 > 0 {
		out.AgeYears = int(age.Int64)
	}
	if height.Valid && height.Float64 > 0 {
		out.HeightCm = height.Float64
	}
	return out.normalized()
}

func (s *Store) SettingsWithEnergy() KcalBudgetSettings {
	out := s.GetKcalBudgetSettings()
	if w, ok := s.LatestWeight(); ok {
		attachEnergyEstimate(&out, w.Kg)
	}
	return out
}

func (s *Store) PutKcalBudgetSettings(in KcalBudgetSettings) (KcalBudgetSettings, error) {
	in = in.normalized()
	if err := in.Validate(); err != nil {
		return KcalBudgetSettings{}, err
	}
	_, err := s.db.Exec(`INSERT INTO user_settings (user_id, soft_low_kcal, soft_high_kcal, gain_min_kcal,
			soft_low_sodium_mg, soft_high_sodium_mg, ceiling_sodium_mg,
			meal_margin_percent,
			legacy_diff_green_lt, legacy_diff_yellow_lt, legacy_diff_orange_lt,
			sex, age_years, height_cm)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
		  soft_low_kcal=excluded.soft_low_kcal,
		  soft_high_kcal=excluded.soft_high_kcal,
		  gain_min_kcal=excluded.gain_min_kcal,
		  soft_low_sodium_mg=excluded.soft_low_sodium_mg,
		  soft_high_sodium_mg=excluded.soft_high_sodium_mg,
		  ceiling_sodium_mg=excluded.ceiling_sodium_mg,
		  meal_margin_percent=excluded.meal_margin_percent,
		  legacy_diff_green_lt=excluded.legacy_diff_green_lt,
		  legacy_diff_yellow_lt=excluded.legacy_diff_yellow_lt,
		  legacy_diff_orange_lt=excluded.legacy_diff_orange_lt,
		  sex=excluded.sex,
		  age_years=excluded.age_years,
		  height_cm=excluded.height_cm`,
		s.userID, in.SoftLowKcal, in.SoftHighKcal, in.GainMinKcal,
		in.SoftLowSodiumMg, in.SoftHighSodiumMg, in.CeilingSodiumMg,
		in.MealMarginPercent,
		in.LegacyDiffGreenLt, in.LegacyDiffYellowLt, in.LegacyDiffOrangeLt,
		in.Sex, in.AgeYears, in.HeightCm)
	if err != nil {
		return KcalBudgetSettings{}, err
	}
	out := in
	if w, ok := s.LatestWeight(); ok {
		attachEnergyEstimate(&out, w.Kg)
	}
	return out, nil
}

func (s *Store) ensureKcalBudgetDefaults() error {
	def := DefaultKcalBudgetSettings()
	_, err := s.db.Exec(`INSERT OR IGNORE INTO user_settings (user_id, soft_low_kcal, soft_high_kcal, gain_min_kcal,
			soft_low_sodium_mg, soft_high_sodium_mg, ceiling_sodium_mg,
			meal_margin_percent,
			legacy_diff_green_lt, legacy_diff_yellow_lt, legacy_diff_orange_lt)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.userID, def.SoftLowKcal, def.SoftHighKcal, def.GainMinKcal,
		def.SoftLowSodiumMg, def.SoftHighSodiumMg, def.CeilingSodiumMg,
		def.MealMarginPercent,
		def.LegacyDiffGreenLt, def.LegacyDiffYellowLt, def.LegacyDiffOrangeLt)
	return err
}
