package health

import "math"

// DayDisplay holds UI-only bands and gap hints; all math is server-side.
type DayDisplay struct {
	IntakeBarBand      string    `json:"intake_bar_band"`       // green|yellow|orange|red|none (kcal day total)
	SodiumIntakeBand   string    `json:"sodium_intake_band"`    // green|yellow|orange|red|none (Na day total)
	KcalDiffBand       string    `json:"kcal_diff_band"`        // green|yellow|orange|red|muted
	SodiumDiffBand     string    `json:"sodium_diff_band"`      // green|yellow|orange|red|muted
	ItemGaps           []ItemGap `json:"item_gaps,omitempty"`
}

func kcalIntakeBarBand(kcal float64, settings KcalBudgetSettings) string {
	s := settings.normalized()
	if kcal <= 0 || math.IsNaN(kcal) {
		return "none"
	}
	if kcal < s.SoftLowKcal {
		return "green"
	}
	if kcal <= s.SoftHighKcal {
		return "yellow"
	}
	if kcal < s.GainMinKcal {
		return "orange"
	}
	return "red"
}

func sodiumIntakeBarBand(sodiumMg float64, settings KcalBudgetSettings) string {
	s := settings.normalized()
	if sodiumMg <= 0 || math.IsNaN(sodiumMg) {
		return "none"
	}
	if sodiumMg < s.SoftLowSodiumMg {
		return "green"
	}
	if sodiumMg <= s.SoftHighSodiumMg {
		return "yellow"
	}
	if sodiumMg < s.CeilingSodiumMg {
		return "orange"
	}
	return "red"
}

func legacyDiffBand(computed, legacy float64, settings KcalBudgetSettings) string {
	if legacy <= 0 || math.IsNaN(legacy) {
		return "muted"
	}
	d := math.Abs(computed - legacy)
	if d <= 0 || math.IsNaN(d) {
		return "green"
	}
	s := settings.normalized()
	if d < s.LegacyDiffGreenLt {
		return "green"
	}
	if d < s.LegacyDiffYellowLt {
		return "yellow"
	}
	if d < s.LegacyDiffOrangeLt {
		return "orange"
	}
	return "red"
}

func dayDisplayFor(d *Day, settings KcalBudgetSettings, itemGaps []ItemGap) DayDisplay {
	if d == nil {
		return DayDisplay{
			IntakeBarBand:    "none",
			SodiumIntakeBand: "none",
			KcalDiffBand:     "muted",
			SodiumDiffBand:   "muted",
		}
	}
	k := d.Totals.Kcal
	na := d.Totals.SodiumMg
	return DayDisplay{
		IntakeBarBand:      kcalIntakeBarBand(k, settings),
		SodiumIntakeBand:   sodiumIntakeBarBand(na, settings),
		KcalDiffBand:       legacyDiffBand(k, d.LegacyTotals.Kcal, settings),
		SodiumDiffBand:     legacyDiffBand(na, d.LegacyTotals.SodiumMg, settings),
		ItemGaps:           itemGaps,
	}
}
