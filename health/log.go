package health

import (
	"strings"
	"time"
)

// DayLog is a single-request write: meals, foods, weight, and day meta.
type DayLog struct {
	Date          string          `json:"date"`
	Slot          string          `json:"slot,omitempty"`
	Items         []MealItem      `json:"items,omitempty"`
	Note          string          `json:"note,omitempty"`
	Raw           string          `json:"raw,omitempty"`
	KcalMargin    float64         `json:"kcal_margin,omitempty"`
	Replace       bool            `json:"replace,omitempty"`
	Meals         []mealWriteBody `json:"meals,omitempty"`
	EnsureFoods   []Food          `json:"ensure_foods,omitempty"`
	DeleteMeals   []string        `json:"delete_meals,omitempty"`
	DeleteFoods   []string        `json:"delete_foods,omitempty"`
	DeleteWeights []string        `json:"delete_weights,omitempty"`
	DeleteDay     bool            `json:"delete_day,omitempty"`
	BP            *string         `json:"bp,omitempty"`
	Walking       *string         `json:"walking,omitempty"`
	Weight        *WeighIn        `json:"weight,omitempty"`
	WeightUpdate  *WeighIn        `json:"weight_update,omitempty"`
}

// DayLogResult is returned from POST /health/v1/log.
type DayLogResult struct {
	Day            *Day     `json:"day,omitempty"`
	Weight         *WeighIn `json:"weight,omitempty"`
	FoodsEnsured   []Food   `json:"foods_ensured,omitempty"`
	DeletedFoods   []string `json:"deleted_foods,omitempty"`
	DeletedWeights []string `json:"deleted_weights,omitempty"`
	DeletedMeals   []string `json:"deleted_meals,omitempty"`
	DeletedDay     string   `json:"deleted_day,omitempty"`
}

// ApplyLog applies all writes in one go so clients (and LLMs) need a single request.
func (s *Store) ApplyLog(req DayLog) (*DayLogResult, error) {
	out := &DayLogResult{}
	date := stringsTrimDate(req.Date)
	if date == "" {
		date = todayAthens()
	}

	for _, id := range req.DeleteFoods {
		if err := s.DeleteFood(id); err != nil {
			return nil, err
		}
		out.DeletedFoods = append(out.DeletedFoods, id)
	}
	for _, id := range req.DeleteWeights {
		if err := s.DeleteWeight(id); err != nil {
			return nil, err
		}
		out.DeletedWeights = append(out.DeletedWeights, id)
	}

	if req.DeleteDay {
		if err := s.DeleteDay(date); err != nil && err != ErrNotFound {
			return nil, err
		}
		out.DeletedDay = date
		return out, nil
	}

	ensured := make([]Food, 0, len(req.EnsureFoods))
	for _, f := range req.EnsureFoods {
		got, err := s.EnsureFood(f)
		if err != nil {
			return nil, err
		}
		ensured = append(ensured, got)
	}

	meals := append([]mealWriteBody(nil), req.Meals...)
	if req.Slot != "" || len(req.Items) > 0 || req.Raw != "" || req.Note != "" {
		meals = append([]mealWriteBody{{
			Slot:       req.Slot,
			Items:      req.Items,
			Note:       req.Note,
			Raw:        req.Raw,
			KcalMargin: req.KcalMargin,
			Replace:    req.Replace,
		}}, meals...)
	}

	var lastDay *Day
	for _, body := range meals {
		slot, meal, err := mealFromWriteBody(body)
		if err != nil {
			return nil, err
		}
		if err := s.PrepareMealSlot(&meal, body.EnsureFoods); err != nil {
			return nil, err
		}
		d, err := s.UpsertMealSlot(date, slot, meal, body.Replace)
		if err != nil {
			return nil, err
		}
		lastDay = d
	}

	for _, slot := range req.DeleteMeals {
		d, err := s.DeleteMealSlot(date, slot)
		if err != nil {
			return nil, err
		}
		out.DeletedMeals = append(out.DeletedMeals, slot)
		lastDay = d
	}

	if req.BP != nil || req.Walking != nil {
		d, err := s.PatchDayMeta(date, req.BP, req.Walking)
		if err != nil {
			return nil, err
		}
		lastDay = d
	}

	if req.Weight != nil && req.Weight.Kg > 0 {
		w := *req.Weight
		if w.Date == "" {
			w.Date = date
		}
		got, err := s.AddWeight(w)
		if err != nil {
			return nil, err
		}
		out.Weight = &got
	}
	if req.WeightUpdate != nil && req.WeightUpdate.ID != "" {
		got, err := s.UpdateWeight(req.WeightUpdate.ID, *req.WeightUpdate)
		if err != nil {
			return nil, err
		}
		out.Weight = got
	}

	if lastDay == nil {
		if d, ok := s.GetDay(date); ok {
			lastDay = d
		}
	}
	out.Day = lastDay
	if len(ensured) > 0 {
		out.FoodsEnsured = ensured
	}
	return out, nil
}

func stringsTrimDate(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 10 {
		if _, err := time.Parse("2006-01-02", s[:10]); err == nil {
			return s[:10]
		}
	}
	return ""
}

// Overview is a compact snapshot for the UI and for one LLM GET.
type Overview struct {
	Today        string   `json:"today"`
	LatestWeight *WeighIn `json:"latest_weight"`
	Day          *Day     `json:"day,omitempty"`
	RecentDays   []Day    `json:"recent_days"`
	FoodCount    int      `json:"food_count"`
}

func (s *Store) Overview(recentLimit int) Overview {
	if recentLimit <= 0 {
		recentLimit = 14
	}
	today := todayAthens()
	from := today
	if t, err := time.Parse("2006-01-02", today); err == nil {
		from = t.AddDate(0, 0, -(recentLimit - 1)).Format("2006-01-02")
	}
	ov := Overview{
		Today:      today,
		RecentDays: s.FilterDays(from, today, "", "", "", recentLimit),
		FoodCount:  s.FoodCount(),
	}
	if w, ok := s.LatestWeight(); ok {
		ov.LatestWeight = w
	}
	if d, ok := s.GetDay(today); ok {
		ov.Day = d
	}
	return ov
}
