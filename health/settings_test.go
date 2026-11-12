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
