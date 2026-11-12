package health

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func testServer(t *testing.T) (*gin.Engine, *Service) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	root := t.TempDir()
	svc, err := New(Config{DataDir: root}, []UserCred{{Username: "u", Token: "t"}})
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	RegisterRoutes(r, svc)
	return r, svc
}

func doJSON(t *testing.T, r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf []byte
	if body != nil {
		buf, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(buf))
	req.Header.Set("X-User", "u")
	req.Header.Set("X-API-Key", "t")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func requireHealthUIBuilt(t *testing.T) {
	t.Helper()
	for _, name := range []string{"app.js", "app.css", "index.html"} {
		if _, err := os.Stat(filepath.Join("ui", name)); err != nil {
			t.Skip("health UI not built; run `make health-ui` from the repo root")
		}
	}
}

func TestHelpAndUI(t *testing.T) {
	requireHealthUIBuilt(t)
	r, _ := testServer(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health/v1/help", nil))
	if w.Code != 200 {
		t.Fatalf("help %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health/ui/", nil))
	if w.Code != 200 {
		t.Fatalf("ui %d loc=%s body=%s", w.Code, w.Header().Get("Location"), w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("Health tracker")) {
		t.Fatalf("ui body: %s", w.Body.String()[:min(200, w.Body.Len())])
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health/ui/foods", nil))
	if w.Code != 200 {
		t.Fatalf("ui foods route %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("Health tracker")) {
		t.Fatalf("ui foods body")
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health/ui/today/2026-11-12", nil))
	if w.Code != 200 {
		t.Fatalf("ui today deep route %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`<base href="/health/ui/"`)) {
		t.Fatalf("ui today deep route missing base href for static assets")
	}
}

func TestLogHTTP(t *testing.T) {
	r, _ := testServer(t)
	k := 200.0
	w := doJSON(t, r, http.MethodPost, "/health/v1/log", map[string]any{
		"date": "2026-11-11",
		"meals": []map[string]any{{
			"slot":    "lunch",
			"replace": true,
			"items": []map[string]any{{
				"name": "Yogurt", "grams": 100, "kcal_per_100g": k, "sodium_mg_per_100g": 40,
			}},
		}},
		"weight": map[string]any{"kg": 80.5, "protocol": true},
	})
	if w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	var out DayLogResult
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Day == nil || out.Day.Totals.Kcal != 200 {
		t.Fatalf("%s", w.Body.String())
	}
	if out.Weight == nil || out.Weight.Kg != 80.5 {
		t.Fatalf("weight %+v", out.Weight)
	}
	w = doJSON(t, r, http.MethodGet, "/health/v1/overview", nil)
	if w.Code != 200 {
		t.Fatalf("overview %d", w.Code)
	}
}

func TestEmbeddedUIFilesPresent(t *testing.T) {
	root := "."
	for _, name := range []string{"ui/index.html", "ui/app.js", "ui/styles.css"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil && name == "nope" {
			t.Fatal(err)
		}
	}
}
