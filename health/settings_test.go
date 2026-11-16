package health

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestKcalBudgetSettingsAPI(t *testing.T) {
	r, _ := testServer(t)
	w := doJSON(t, r, http.MethodGet, "/health/v1/settings", nil)
	if w.Code != 200 {
		t.Fatalf("get settings %d %s", w.Code, w.Body.String())
	}
	var got struct {
		Settings KcalBudgetSettings `json:"settings"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	def := DefaultKcalBudgetSettings()
	if got.Settings.SoftLowKcal != def.SoftLowKcal || got.Settings.GainMinKcal != def.GainMinKcal {
		t.Fatalf("defaults %+v", got.Settings)
	}
	w = doJSON(t, r, http.MethodPut, "/health/v1/settings", KcalBudgetSettings{
		SoftLowKcal:  1400,
		SoftHighKcal: 1650,
		GainMinKcal:  2100,
	})
	if w.Code != 200 {
		t.Fatalf("put settings %d %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodGet, "/health/v1/overview?days=1", nil)
	if w.Code != 200 {
		t.Fatalf("overview %d", w.Code)
	}
	var ov Overview
	if err := json.Unmarshal(w.Body.Bytes(), &ov); err != nil {
		t.Fatal(err)
	}
	if ov.Settings.SoftLowKcal != 1400 {
		t.Fatalf("overview settings %+v", ov.Settings)
	}
}

func TestEnergyProfileSettingsAPI(t *testing.T) {
	r, _ := testServer(t)
	w := doJSON(t, r, http.MethodPut, "/health/v1/settings", KcalBudgetSettings{
		SoftLowKcal:  1500,
		SoftHighKcal: 1700,
		GainMinKcal:  2200,
		Sex:          "male",
		AgeYears:     40,
		HeightCm:     180,
	})
	if w.Code != 200 {
		t.Fatalf("put energy %d %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodPost, "/health/v1/weight", map[string]any{
		"date": "2026-11-16",
		"kg":   90,
	})
	if w.Code != 201 {
		t.Fatalf("weight %d %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodGet, "/health/v1/settings", nil)
	if w.Code != 200 {
		t.Fatalf("get %d %s", w.Code, w.Body.String())
	}
	var got struct {
		Settings KcalBudgetSettings `json:"settings"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Settings.Sex != "male" || got.Settings.AgeYears != 40 || got.Settings.HeightCm != 180 {
		t.Fatalf("profile %+v", got.Settings)
	}
	if got.Settings.BmrKcal != 1830 || got.Settings.SedentaryKcal != 2196 {
		t.Fatalf("energy %+v", got.Settings)
	}
}
