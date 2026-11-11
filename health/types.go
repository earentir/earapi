package health

// WeighIn is one scale reading (same calendar day may have several).
type WeighIn struct {
	ID       string   `json:"id"`
	Date     string   `json:"date"` // YYYY-MM-DD
	Kg       float64  `json:"kg"`
	RHRBpm   *int     `json:"rhr_bpm,omitempty"`
	Note     string   `json:"note,omitempty"`
	Protocol bool     `json:"protocol,omitempty"`
	KgRaw    string   `json:"kg_raw,omitempty"` // original text when kg was ambiguous
}

// Food is a catalog reference item.
type Food struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	KcalPer100g     *float64 `json:"kcal_per_100g,omitempty"`
	SodiumMgPer100g *float64 `json:"sodium_mg_per_100g,omitempty"`
	Notes           string   `json:"notes,omitempty"`
}

// MealItem is one line in a structured meal log.
type MealItem struct {
	Name      string   `json:"name"`
	Grams     float64  `json:"grams,omitempty"`
	Kcal      float64  `json:"kcal,omitempty"`
	SodiumMg  float64  `json:"sodium_mg,omitempty"`
	Note      string   `json:"note,omitempty"`
}

// MealSlot holds lunch, dinner, or snack for a day.
type MealSlot struct {
	Raw       string     `json:"raw,omitempty"`
	Items     []MealItem `json:"items,omitempty"`
	SodiumMg  float64    `json:"sodium_mg,omitempty"`
	Kcal      float64    `json:"kcal,omitempty"`
	KcalMargin float64   `json:"kcal_margin,omitempty"` // e.g. 0.10 when applied
}

// DayTotals are summed sodium/kcal for the day.
type DayTotals struct {
	SodiumMg float64 `json:"sodium_mg"`
	Kcal     float64 `json:"kcal"`
}

// Day is one calendar day's eating log.
type Day struct {
	Date   string             `json:"date"`
	Meals  map[string]MealSlot `json:"meals"`
	Totals DayTotals          `json:"totals"`
	BP     string             `json:"bp,omitempty"`
	Walking string            `json:"walking,omitempty"`
}

// ValidMealSlots are accepted slot/type values.
var ValidMealSlots = map[string]bool{
	"lunch": true, "dinner": true, "snack": true,
}
