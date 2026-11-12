package health

import (
	"fmt"
	"strings"
	"time"
)

// FoodGap is a catalog row missing per-100g data (no kcal/sodium numbers in the response).
type FoodGap struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Missing []string `json:"missing"`
}

// ItemGap is a meal line that still needs catalog or per-100g data to compute nutrients.
type ItemGap struct {
	Ref   string  `json:"ref"`
	Date  string  `json:"date"`
	Slot  string  `json:"slot"`
	Index int     `json:"index"`
	Name  string  `json:"name"`
	Grams float64 `json:"grams,omitempty"`
	Note  string  `json:"note,omitempty"`
}

// NutritionGaps is returned from GET /health/v1/gaps.
type NutritionGaps struct {
	Foods     []FoodGap `json:"foods"`
	Items     []ItemGap `json:"items"`
	FoodCount int       `json:"food_count"`
	ItemCount int       `json:"item_count"`
}

// NutritionFill upserts catalog data and recomputes affected meals in one request.
type NutritionFill struct {
	Foods          []Food        `json:"foods"`
	RecomputeSlots []MealSlotRef `json:"recompute_slots,omitempty"`
	RecomputeDays  []string      `json:"recompute_days,omitempty"`
}

// MealSlotRef identifies one meal slot to rescale after catalog updates.
type MealSlotRef struct {
	Date string `json:"date"`
	Slot string `json:"slot"`
}

// NutritionFillResult summarizes POST /health/v1/fill.
type NutritionFillResult struct {
	FoodsEnsured []Food        `json:"foods_ensured"`
	FoodsCreated []Food        `json:"foods_created,omitempty"`
	Reply        string        `json:"reply,omitempty"`
	Recomputed   []MealSlotRef `json:"recomputed"`
	Gaps         NutritionGaps `json:"gaps_remaining"`
}

func foodGaps(foods []Food) []FoodGap {
	var out []FoodGap
	for _, f := range foods {
		var miss []string
		if f.KcalPer100g == nil {
			miss = append(miss, "kcal_per_100g")
		}
		if f.SodiumMgPer100g == nil {
			miss = append(miss, "sodium_mg_per_100g")
		}
		if len(miss) == 0 {
			continue
		}
		out = append(out, FoodGap{ID: f.ID, Name: f.Name, Missing: miss})
	}
	return out
}

func (s *Store) itemNeedsNutrition(it MealItem, foods []Food) bool {
	name := strings.TrimSpace(it.Name)
	if name == "" {
		return false
	}
	if len(it.Parts) > 0 {
		for _, p := range it.Parts {
			if itemNeedsNutritionLeaf(p, foods) {
				return true
			}
		}
		return it.Kcal == 0 && it.SodiumMg == 0 && it.Grams > 0
	}
	return itemNeedsNutritionLeaf(it, foods)
}

func itemNeedsNutritionLeaf(it MealItem, foods []Food) bool {
	if it.Kcal > 0 || it.SodiumMg > 0 {
		return false
	}
	if it.Grams <= 0 {
		return false
	}
	if it.FoodID != "" {
		for _, f := range foods {
			if f.ID == it.FoodID {
				return f.KcalPer100g == nil && f.SodiumMgPer100g == nil
			}
		}
	}
	f, ok := BestFoodMatch(foods, it.Name)
	if !ok || f == nil {
		return true
	}
	if f.KcalPer100g == nil && f.SodiumMgPer100g == nil {
		return true
	}
	return false
}

// ListNutritionGaps returns catalog and meal items that still need per-100g or item-level nutrients.
func (s *Store) ListNutritionGaps(from, to string, itemLimit int) NutritionGaps {
	foods := s.ListFoods()
	out := NutritionGaps{
		Foods:     foodGaps(foods),
		FoodCount: len(foodGaps(foods)),
	}
	if itemLimit <= 0 {
		itemLimit = 100
	}
	if from == "" && to == "" {
		to = todayAthens()
		if t, err := parseISODate(to); err == nil {
			from = t.AddDate(0, 0, -89).Format("2006-01-02")
		}
	}
	days := s.FilterDays(from, to, "", "", "", 0)
	for _, d := range days {
		for _, slot := range []string{"lunch", "dinner", "snack"} {
			m, ok := d.Meals[slot]
			if !ok {
				continue
			}
			if MealItemsNeedReparse(m.Items, m.Raw) {
				if parsed := ParseMealStructureFromRaw(m.Raw); len(parsed) > 0 {
					m.Items = parsed
				}
			}
			var walk func(items []MealItem, prefix string)
			walk = func(items []MealItem, prefix string) {
				for i, it := range items {
					ref := fmt.Sprintf("%s/%s/%d", d.Date, slot, i)
					if prefix != "" {
						ref = prefix + "/" + fmt.Sprint(i)
					}
					if len(it.Parts) > 0 {
						walk(it.Parts, ref)
						continue
					}
					if !s.itemNeedsNutrition(it, foods) {
						continue
					}
					out.Items = append(out.Items, ItemGap{
						Ref: ref, Date: d.Date, Slot: slot, Index: i,
						Name: it.Name, Grams: it.Grams, Note: strings.TrimSpace(it.Note),
					})
					if len(out.Items) >= itemLimit {
						return
					}
				}
			}
			walk(m.Items, "")
			if len(out.Items) >= itemLimit {
				out.ItemCount = len(out.Items)
				return out
			}
		}
	}
	out.ItemCount = len(out.Items)
	return out
}

func parseISODate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if len(s) > 10 {
		s = s[:10]
	}
	return time.Parse("2006-01-02", s)
}

func (s *Store) ApplyNutritionFill(req NutritionFill) (*NutritionFillResult, error) {
	ensured := make([]Food, 0, len(req.Foods))
	var created []Food
	names := map[string]bool{}
	for _, f := range req.Foods {
		got, isNew, err := s.EnsureFood(f)
		if err != nil {
			return nil, err
		}
		ensured = append(ensured, got)
		names[strings.ToLower(strings.TrimSpace(got.Name))] = true
		if isNew {
			created = append(created, got)
		}
	}

	slots := append([]MealSlotRef(nil), req.RecomputeSlots...)
	if len(slots) == 0 && len(req.RecomputeDays) > 0 {
		for _, date := range req.RecomputeDays {
			for _, slot := range []string{"lunch", "dinner", "snack"} {
				slots = append(slots, MealSlotRef{Date: date, Slot: slot})
			}
		}
	}
	if len(slots) == 0 && len(names) > 0 {
		slots = s.slotsReferencingFoodNames(names, "", "")
	}

	seen := map[string]bool{}
	recomputed := make([]MealSlotRef, 0, len(slots))
	for _, ref := range slots {
		key := ref.Date + "/" + ref.Slot
		if seen[key] {
			continue
		}
		seen[key] = true
		if err := s.recomputeMealSlot(ref.Date, ref.Slot); err != nil {
			if err == ErrNotFound {
				continue
			}
			return nil, err
		}
		recomputed = append(recomputed, ref)
	}

	gaps := s.ListNutritionGaps("", "", 100)
	out := &NutritionFillResult{
		FoodsEnsured: ensured,
		Recomputed:   recomputed,
		Gaps:         gaps,
	}
	if len(created) > 0 {
		out.FoodsCreated = dedupeFoodsByID(created)
		out.Reply = FormatFoodsCreatedReply(out.FoodsCreated)
	}
	return out, nil
}

func (s *Store) slotsReferencingFoodNames(names map[string]bool, from, to string) []MealSlotRef {
	if from == "" && to == "" {
		to = todayAthens()
		if t, err := parseISODate(to); err == nil {
			from = t.AddDate(0, 0, -89).Format("2006-01-02")
		}
	}
	var out []MealSlotRef
	seen := map[string]bool{}
	for _, d := range s.FilterDays(from, to, "", "", "", 0) {
		for _, slot := range []string{"lunch", "dinner", "snack"} {
			m, ok := d.Meals[slot]
			if !ok || mealEmpty(m) {
				continue
			}
			if mealReferencesNames(m, names) {
				key := d.Date + "/" + slot
				if !seen[key] {
					seen[key] = true
					out = append(out, MealSlotRef{Date: d.Date, Slot: slot})
				}
			}
		}
	}
	return out
}

func mealReferencesNames(m MealSlot, names map[string]bool) bool {
	var walk func([]MealItem) bool
	walk = func(items []MealItem) bool {
		for _, it := range items {
			if names[strings.ToLower(strings.TrimSpace(it.Name))] {
				return true
			}
			if len(it.Parts) > 0 && walk(it.Parts) {
				return true
			}
		}
		return false
	}
	return walk(m.Items)
}

func (s *Store) recomputeMealSlot(date, slot string) error {
	slot = strings.ToLower(strings.TrimSpace(slot))
	if !ValidMealSlots[slot] {
		return fmt.Errorf("%w: invalid slot", ErrInvalidInput)
	}
	d, ok := s.GetDay(date)
	if !ok {
		return ErrNotFound
	}
	m, ok := d.Meals[slot]
	if !ok {
		return ErrNotFound
	}
	if len(m.Items) == 0 && strings.TrimSpace(m.Raw) != "" {
		m.Items = ParseMealStructureFromRaw(m.Raw)
	}
	if _, err := s.PrepareMealSlot(&m, nil); err != nil {
		return err
	}
	d.Meals[slot] = m
	return s.PutDay(*d)
}
