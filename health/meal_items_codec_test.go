package health

import (
	"database/sql"
	"testing"
)

func TestRepairMealSlotsItemsJSON_clearsSpuriousCount(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, _ = db.Exec(schemaDDL)
	bad := `[{"name":"soup-tomato mix","grams":360,"count":360,"note":"soup-tomato mix 360 g"}]`
	_, err = db.Exec(`INSERT INTO days (user_id, date) VALUES ('u','2026-11-12')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO meal_slots (user_id, date, slot, items_json) VALUES ('u','2026-11-12','lunch',?)`, bad)
	if err != nil {
		t.Fatal(err)
	}
	if err := repairMealSlotsItemsJSON(db); err != nil {
		t.Fatal(err)
	}
	var fixed string
	if err := db.QueryRow(`SELECT items_json FROM meal_slots WHERE user_id='u'`).Scan(&fixed); err != nil {
		t.Fatal(err)
	}
	items := decodeMealItemsJSON(fixed)
	if len(items) != 1 || items[0].Count != 0 || items[0].Grams != 360 {
		t.Fatalf("after repair %+v json=%s", items, fixed)
	}
}
