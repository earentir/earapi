package health

// WeighIn is one scale reading (same calendar day may have several).
type WeighIn struct {
	ID       string  `json:"id"`
	Date     string  `json:"date"` // YYYY-MM-DD
	Kg       float64 `json:"kg"`
	RHRBpm   *int    `json:"rhr_bpm,omitempty"`
	Note     string  `json:"note,omitempty"`
	Protocol bool    `json:"protocol,omitempty"`
	KgRaw    string  `json:"kg_raw,omitempty"` // original text when kg was ambiguous
}

// Food is a catalog reference item.
type Food struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	KcalPer100g     *float64 `json:"kcal_per_100g,omitempty"`
	SodiumMgPer100g *float64 `json:"sodium_mg_per_100g,omitempty"`
	Notes           string   `json:"notes,omitempty"`
}

// MealItem is one line in a structured meal log: a catalog food + grams,
// or a dish (parts = recipe batch, grams = how much of the dish was eaten).
type MealItem struct {
	Name            string     `json:"name"`
	FoodID          string     `json:"food_id,omitempty"`
	StockLotID      string     `json:"stock_lot_id,omitempty"`
	Grams           float64    `json:"grams,omitempty"`
	Kcal            float64    `json:"kcal,omitempty"`
	SodiumMg        float64    `json:"sodium_mg,omitempty"`
	KcalPer100g     *float64   `json:"kcal_per_100g,omitempty"`      // create/update catalog on log
	SodiumMgPer100g *float64   `json:"sodium_mg_per_100g,omitempty"` // create/update catalog on log
	Note            string     `json:"note,omitempty"`
	Parts           []MealItem `json:"parts,omitempty"`
}

// MealSlot holds lunch, dinner, or snack for a day.
type MealSlot struct {
	Raw        string     `json:"raw,omitempty"`
	Items      []MealItem `json:"items,omitempty"`
	SodiumMg   float64    `json:"sodium_mg,omitempty"`
	Kcal       float64    `json:"kcal,omitempty"`
	KcalMargin float64    `json:"kcal_margin,omitempty"` // e.g. 0.10 when applied
}

// DayTotals are summed sodium/kcal for the day.
type DayTotals struct {
	SodiumMg float64 `json:"sodium_mg"`
	Kcal     float64 `json:"kcal"`
}

// Day is one calendar day's eating log.
type Day struct {
	Date    string              `json:"date"`
	Meals   map[string]MealSlot `json:"meals"`
	Totals  DayTotals           `json:"totals"`
	BP      string              `json:"bp,omitempty"`
	Walking string              `json:"walking,omitempty"`
}

// ValidMealSlots are accepted slot/type values.
var ValidMealSlots = map[string]bool{
	"lunch": true, "dinner": true, "snack": true,
}

// StockLot is a prepped batch (one meal-style set): Items are the full-batch ingredient lines (parts).
type StockLot struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	PreparedDate    string     `json:"prepared_date,omitempty"`
	BatchGrams      float64    `json:"batch_grams"`
	RemainingGrams  float64    `json:"remaining_grams"`
	Active          *bool      `json:"active,omitempty"`
	Items           []MealItem `json:"items,omitempty"`
	Notes           string     `json:"notes,omitempty"`
	KcalPer100g     *float64          `json:"kcal_per_100g,omitempty"`
	SodiumMgPer100g *float64          `json:"sodium_mg_per_100g,omitempty"`
	Usage           []StockMovement   `json:"usage,omitempty"`
}

// StockDraw logs grams from a lot onto a day slot.
type StockDraw struct {
	Date  string  `json:"date"`
	Slot  string  `json:"slot"`
	Grams float64 `json:"grams"`
}

// StockAdjust removes grams from a lot without logging a meal (household / discarded).
type StockAdjust struct {
	Grams float64 `json:"grams"` // 0 = all remaining
	Mode  string  `json:"mode"`  // household, discarded
}
