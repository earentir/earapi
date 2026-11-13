package health

import (
	"database/sql"
	"encoding/json"
)

func decodeMealItemsJSON(raw string) []MealItem {
	if raw == "" || raw == "[]" {
		return nil
	}
	var items []MealItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	normalizeMealItems(items)
	return items
}

func encodeMealItemsJSON(items []MealItem) []byte {
	normalizeMealItems(items)
	b, err := json.Marshal(items)
	if err != nil || b == nil {
		return []byte("[]")
	}
	return b
}

func repairStoredMealItemsJSON(db *sql.DB) error {
	if db == nil {
		return nil
	}
	if err := repairMealSlotsItemsJSON(db); err != nil {
		return err
	}
	return repairStockLotsItemsJSON(db)
}

func repairMealSlotsItemsJSON(db *sql.DB) error {
	rows, err := db.Query(`SELECT user_id, date, slot, items_json FROM meal_slots`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type row struct {
		user, date, slot, raw string
	}
	var pending []row
	for rows.Next() {
		var r row
		if rows.Scan(&r.user, &r.date, &r.slot, &r.raw) != nil {
			continue
		}
		items := decodeMealItemsJSON(r.raw)
		if string(encodeMealItemsJSON(items)) == r.raw {
			continue
		}
		pending = append(pending, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, r := range pending {
		items := decodeMealItemsJSON(r.raw)
		_, err := db.Exec(`UPDATE meal_slots SET items_json = ? WHERE user_id = ? AND date = ? AND slot = ?`,
			string(encodeMealItemsJSON(items)), r.user, r.date, r.slot)
		if err != nil {
			return err
		}
	}
	return nil
}

func repairStockLotsItemsJSON(db *sql.DB) error {
	rows, err := db.Query(`SELECT user_id, id, items_json FROM stock_lots`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type row struct {
		user, id, raw string
	}
	var pending []row
	for rows.Next() {
		var r row
		if rows.Scan(&r.user, &r.id, &r.raw) != nil {
			continue
		}
		items := decodeMealItemsJSON(r.raw)
		if string(encodeMealItemsJSON(items)) == r.raw {
			continue
		}
		pending = append(pending, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, r := range pending {
		items := decodeMealItemsJSON(r.raw)
		_, err := db.Exec(`UPDATE stock_lots SET items_json = ? WHERE user_id = ? AND id = ?`,
			string(encodeMealItemsJSON(items)), r.user, r.id)
		if err != nil {
			return err
		}
	}
	return nil
}
