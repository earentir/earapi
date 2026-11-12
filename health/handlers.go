package health

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes mounts /health/v1 routes and the /health/ui tracker.
func RegisterRoutes(r *gin.Engine, svc *Service) {
	registerUI(r)

	g := r.Group("/health/v1")
	g.GET("/help", svc.handleHelp)

	auth := AuthMiddleware(svc.authUsers)
	p := g.Group("")
	p.Use(auth)
	{
		p.GET("/overview", svc.handleOverview)
		p.GET("/gaps", svc.handleNutritionGaps)
		p.POST("/fill", svc.handleNutritionFill)
		p.POST("/log", svc.handleLog)

		p.GET("/weight/latest", svc.handleWeightLatest)
		p.GET("/weight", svc.handleWeightList)
		p.POST("/weight", svc.handleWeightAdd)
		p.PUT("/weight/:id", svc.handleWeightUpdate)
		p.DELETE("/weight/:id", svc.handleWeightDelete)

		p.GET("/foods", svc.handleFoodsList)
		p.POST("/foods/batch", svc.handleFoodsBatch)
		p.GET("/foods/:id", svc.handleFoodGet)
		p.POST("/foods", svc.handleFoodAdd)
		p.PUT("/foods/:id", svc.handleFoodUpdate)
		p.DELETE("/foods/:id", svc.handleFoodDelete)
		p.POST("/foods/:id/aliases", svc.handleFoodAddAlias)

		p.GET("/days", svc.handleDaysList)
		p.GET("/days/:date", svc.handleDayGet)
		p.POST("/days", svc.handleDayCreate)
		p.PUT("/days/:date", svc.handleDayPut)
		p.PATCH("/days/:date", svc.handleDayPatch)
		p.DELETE("/days/:date", svc.handleDayDelete)

		p.POST("/days/:date/meals/log", svc.handleDayMealLog)
		p.POST("/days/:date/meals", svc.handleDayMealPost)
		p.PUT("/days/:date/meals/:slot", svc.handleDayMealPut)
		p.DELETE("/days/:date/meals/:slot", svc.handleDayMealDelete)

		p.GET("/import/json/status", svc.handleImportJSONStatus)
		p.POST("/import/json", svc.handleImportJSON)
		p.POST("/import/markdown", svc.handleImportMarkdown)

		p.GET("/stock", svc.handleStockList)
		p.GET("/stock/:id", svc.handleStockGet)
		p.POST("/stock", svc.handleStockCreate)
		p.PUT("/stock/:id", svc.handleStockUpdate)
		p.DELETE("/stock/:id", svc.handleStockDelete)
		p.POST("/stock/:id/draw", svc.handleStockDraw)
		p.POST("/stock/:id/adjust", svc.handleStockAdjust)
		p.PATCH("/stock/:id/active", svc.handleStockActive)
	}
}

func (s *Service) store(c *gin.Context) (*Store, bool) {
	user := UserFromContext(c)
	if user == "" {
		return nil, false
	}
	st, err := s.StoreForUser(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return nil, false
	}
	return st, true
}

func (s *Service) handleHelp(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"auth":         "Send X-User and X-API-Key headers (or ?user=&token= on GET). Bearer token works with X-User.",
		"ui":           "/health/ui/",
		"llm_workflow": "GET /gaps once for rows missing per-100g or unscorable items (no kcal numbers in response). POST /fill with foods[] (batch EnsureFood) to upsert catalog and auto-recompute affected meals. POST /log for new eating. Items are name+grams or dishes with parts[]. Import/markdown splits + into structured items. Server computes kcal/sodium. EnsureFood fuzzy-matches existing catalog names before insert; responses may include foods_created[] and reply when new rows were added (user should review Foods in the web UI).",
		"routes": []string{
			"GET /health/v1/help",
			"GET /health/v1/overview?days=14",
			"GET /health/v1/gaps?from=&to=&limit=",
			"POST /health/v1/fill",
			"POST /health/v1/log",
			"GET /health/v1/weight/latest",
			"GET /health/v1/weight?from=&to=",
			"POST /health/v1/weight",
			"PUT /health/v1/weight/:id",
			"DELETE /health/v1/weight/:id",
			"GET /health/v1/foods?q=&limit=",
			"POST /health/v1/foods/batch",
			"GET /health/v1/foods/:id",
			"POST /health/v1/foods",
			"PUT /health/v1/foods/:id",
			"DELETE /health/v1/foods/:id",
			"GET /health/v1/stock?active=1",
			"GET /health/v1/stock/:id",
			"POST /health/v1/stock",
			"PUT /health/v1/stock/:id",
			"DELETE /health/v1/stock/:id",
			"POST /health/v1/stock/:id/draw  (body: date, slot, grams)",
			"POST /health/v1/stock/:id/adjust  (body: grams, mode household|discarded — no meal log)",
			"PATCH /health/v1/stock/:id/active  (body: {active: true|false})",
			"GET /health/v1/days?from=&to=&date=&q=&slot=&limit=",
			"GET /health/v1/days/:date",
			"POST /health/v1/days",
			"PUT /health/v1/days/:date",
			"PATCH /health/v1/days/:date",
			"DELETE /health/v1/days/:date",
			"POST /health/v1/days/:date/meals/log",
			"POST /health/v1/days/:date/meals",
			"PUT /health/v1/days/:date/meals/:slot",
			"DELETE /health/v1/days/:date/meals/:slot",
			"GET /health/v1/import/json/status",
			"POST /health/v1/import/json?merge=false&source=legacy",
			"POST /health/v1/import/markdown?replace=true",
		},
		"examples": gin.H{
			"log_day": gin.H{
				"date": "2026-11-11",
				"bp":   "128/82",
				"walking":     "park 40min",
				"weight":      gin.H{"kg": 142.5, "protocol": true, "note": "morning post-BM"},
				"meals": []gin.H{
					{"slot": "lunch", "replace": true, "items": []gin.H{
						{"name": "Pasta (cooked, no salt)", "grams": 120},
						{"name": "Chicken breast (cooked, no salt)", "grams": 80},
					}},
					{"slot": "snack", "items": []gin.H{
						{"name": "New bar", "grams": 45, "kcal_per_100g": 400, "sodium_mg_per_100g": 120},
					}},
					{"slot": "dinner", "items": []gin.H{
						{"name": "Salad", "grams": 250, "parts": []gin.H{
							{"name": "Iceberg", "grams": 100},
							{"name": "Tomato", "grams": 120},
							{"name": "EVOO", "grams": 6},
						}},
					}},
				},
			},
			"log_fix": gin.H{
				"date":          "2026-11-11",
				"delete_meals":  []string{"snack"},
				"delete_foods":  []string{"new-bar"},
				"weight_update": gin.H{"id": "w_2026-11-11_1", "kg": 142.2, "protocol": true},
			},
			"fill_gaps": gin.H{
				"foods": []gin.H{
					{"name": "New spice", "kcal_per_100g": 250, "sodium_mg_per_100g": 800},
					{"id": "pasta", "kcal_per_100g": 155, "sodium_mg_per_100g": 1},
				},
				"recompute_days": []string{"2026-11-11"},
			},
		},
	})
}

func (s *Service) handleNutritionGaps(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	from := c.Query("from")
	to := c.Query("to")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	c.JSON(http.StatusOK, st.ListNutritionGaps(from, to, limit))
}

func (s *Service) handleNutritionFill(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	var body NutritionFill
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	out, err := st.ApplyNutritionFill(body)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (s *Service) handleOverview(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	n, _ := strconv.Atoi(c.DefaultQuery("days", "14"))
	c.JSON(http.StatusOK, st.Overview(n))
}

func (s *Service) handleLog(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	var body DayLog
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	out, err := st.ApplyLog(body)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (s *Service) handleWeightLatest(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	w, found := st.LatestWeight()
	if !found {
		c.JSON(http.StatusOK, gin.H{"weight": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"weight": w})
}

func (s *Service) handleWeightList(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	from := c.Query("from")
	to := c.Query("to")
	list := st.ListWeights(from, to)
	c.JSON(http.StatusOK, gin.H{"weights": list, "count": len(list)})
}

func (s *Service) handleWeightAdd(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	var body struct {
		Date     string  `json:"date"`
		Kg       float64 `json:"kg"`
		Note     string  `json:"note"`
		RHRBpm   *int    `json:"rhr_bpm"`
		Protocol bool    `json:"protocol"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	w := WeighIn{
		Date: body.Date, Kg: body.Kg, Note: body.Note,
		RHRBpm: body.RHRBpm, Protocol: body.Protocol,
	}
	out, err := st.AddWeight(w)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"weight": out})
}

func (s *Service) handleWeightUpdate(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	var body WeighIn
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	out, err := st.UpdateWeight(c.Param("id"), body)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"weight": out})
}

func (s *Service) handleWeightDelete(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	if err := st.DeleteWeight(c.Param("id")); err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": c.Param("id")})
}

func (s *Service) handleFoodsList(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	all := st.ListFoods()
	q := c.Query("q")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))
	if q != "" {
		all = RankFoods(all, q, limit)
	} else if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	c.JSON(http.StatusOK, gin.H{
		"foods":   all,
		"aliases": st.ListFoodAliases(),
		"count":   len(all),
	})
}

func (s *Service) handleFoodAddAlias(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	var body struct {
		Alias string `json:"alias"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	out, err := st.AddFoodAlias(c.Param("id"), body.Alias)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"alias": out})
}

func (s *Service) handleFoodGet(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	f, found := st.FoodByID(c.Param("id"))
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"food": f})
}

func (s *Service) handleFoodAdd(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	var f Food
	if err := c.BindJSON(&f); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	out, err := st.AddFood(f)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"food": out})
}

func (s *Service) handleFoodUpdate(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	var patch Food
	if err := c.BindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	out, err := st.UpdateFood(c.Param("id"), patch)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"food": out})
}

func (s *Service) handleFoodDelete(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	if err := st.DeleteFood(c.Param("id")); err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": c.Param("id")})
}

func (s *Service) handleFoodsBatch(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	var body struct {
		Upsert []Food   `json:"upsert"`
		Delete []string `json:"delete"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	upserted := make([]Food, 0, len(body.Upsert))
	for _, f := range body.Upsert {
		out, _, err := st.EnsureFood(f)
		if err != nil {
			writeStoreErr(c, err)
			return
		}
		upserted = append(upserted, out)
	}
	deleted := make([]string, 0, len(body.Delete))
	for _, id := range body.Delete {
		if err := st.DeleteFood(id); err != nil {
			writeStoreErr(c, err)
			return
		}
		deleted = append(deleted, id)
	}
	c.JSON(http.StatusOK, gin.H{"foods": upserted, "deleted": deleted})
}

func (s *Service) handleDaysList(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	from := c.Query("from")
	to := c.Query("to")
	date := c.Query("date")
	q := c.Query("q")
	slot := c.Query("slot")
	if slot == "" {
		slot = c.Query("type")
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))
	days := st.FilterDays(from, to, date, q, slot, limit)
	c.JSON(http.StatusOK, gin.H{"days": days, "count": len(days)})
}

func (s *Service) handleDayGet(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	d, found := st.GetDay(c.Param("date"))
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"day": d})
}

func (s *Service) handleDayCreate(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	var d Day
	if err := c.BindJSON(&d); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if d.Date == "" {
		d.Date = todayAthens()
	}
	if d.Meals == nil {
		d.Meals = map[string]MealSlot{}
	}
	if err := st.prepareDayMeals(&d); err != nil {
		writeStoreErr(c, err)
		return
	}
	if existing, found := st.GetDay(d.Date); found {
		// merge meals
		for slot, m := range d.Meals {
			if cur, ok := existing.Meals[slot]; ok && !mealEmpty(cur) {
				cur.Items = append(cur.Items, m.Items...)
				if m.Raw != "" {
					cur.Raw = strings.TrimSpace(cur.Raw + " + " + m.Raw)
				}
				cur.SodiumMg += m.SodiumMg
				cur.Kcal += m.Kcal
				existing.Meals[slot] = cur
			} else {
				existing.Meals[slot] = m
			}
		}
		if d.BP != "" {
			existing.BP = d.BP
		}
		if d.Walking != "" {
			existing.Walking = d.Walking
		}
		d = *existing
	}
	if err := st.PutDay(d); err != nil {
		writeStoreErr(c, err)
		return
	}
	out, _ := st.GetDay(d.Date)
	c.JSON(http.StatusCreated, gin.H{"day": out})
}

func (s *Service) handleDayPut(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	var d Day
	if err := c.BindJSON(&d); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	d.Date = c.Param("date")
	if d.Meals == nil {
		d.Meals = map[string]MealSlot{}
	}
	if err := st.prepareDayMeals(&d); err != nil {
		writeStoreErr(c, err)
		return
	}
	if err := st.PutDay(d); err != nil {
		writeStoreErr(c, err)
		return
	}
	out, _ := st.GetDay(d.Date)
	c.JSON(http.StatusOK, gin.H{"day": out})
}

func (s *Service) handleDayDelete(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	if err := st.DeleteDay(c.Param("date")); err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": c.Param("date")})
}

func (s *Service) handleDayPatch(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	var body struct {
		BP      *string `json:"bp"`
		Walking *string `json:"walking"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	d, err := st.PatchDayMeta(c.Param("date"), body.BP, body.Walking)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"day": d})
}

func (s *Store) prepareDayMeals(d *Day) error {
	for slot, m := range d.Meals {
		meal := m
		if _, err := s.PrepareMealSlot(&meal, nil); err != nil {
			return err
		}
		d.Meals[slot] = meal
	}
	return nil
}

type mealWriteBody struct {
	Slot        string     `json:"slot"`
	Items       []MealItem `json:"items"`
	EnsureFoods []Food     `json:"ensure_foods,omitempty"`
	Note        string     `json:"note"`
	Raw         string     `json:"raw"`
	KcalMargin  float64    `json:"kcal_margin"`
	SodiumMg    float64    `json:"sodium_mg"`
	Kcal        float64    `json:"kcal"`
	Replace     bool       `json:"replace"`
}

func mealFromWriteBody(body mealWriteBody) (string, MealSlot, error) {
	if body.Slot == "" {
		return "", MealSlot{}, fmt.Errorf("%w: slot required", ErrInvalidInput)
	}
	meal := MealSlot{
		Raw: body.Raw, Items: body.Items,
		SodiumMg: body.SodiumMg, Kcal: body.Kcal,
	}
	if body.Note != "" {
		if meal.Raw != "" {
			meal.Raw += " + " + body.Note
		} else {
			meal.Raw = body.Note
		}
	}
	return body.Slot, meal, nil
}

func (s *Service) handleDayMealLog(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	date := c.Param("date")
	var body mealWriteBody
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	slot, meal, err := mealFromWriteBody(body)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	ensured, created, err := st.ensureMealFoods(&meal, body.EnsureFoods)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	d, err := st.UpsertMealSlot(date, slot, meal, body.Replace)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	resp := gin.H{"day": d}
	if len(ensured) > 0 {
		resp["foods_ensured"] = ensured
	}
	if len(created) > 0 {
		resp["foods_created"] = created
		resp["reply"] = FormatFoodsCreatedReply(created)
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Service) handleDayMealPost(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	date := c.Param("date")
	var body mealWriteBody
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	slot, meal, err := mealFromWriteBody(body)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	created, err := st.PrepareMealSlot(&meal, body.EnsureFoods)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	d, err := st.UpsertMealSlot(date, slot, meal, body.Replace)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	resp := gin.H{"day": d}
	if len(created) > 0 {
		resp["foods_created"] = created
		resp["reply"] = FormatFoodsCreatedReply(created)
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Service) handleDayMealPut(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	date := c.Param("date")
	slot := c.Param("slot")
	var body struct {
		MealSlot
		EnsureFoods []Food `json:"ensure_foods,omitempty"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	meal := body.MealSlot
	created, err := st.PrepareMealSlot(&meal, body.EnsureFoods)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	d, err := st.SetMealSlot(date, slot, meal)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	resp := gin.H{"day": d}
	if len(created) > 0 {
		resp["foods_created"] = created
		resp["reply"] = FormatFoodsCreatedReply(created)
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Store) ensureMealFoods(meal *MealSlot, ensureFoods []Food) (ensured, created []Food, err error) {
	for _, f := range ensureFoods {
		got, isNew, e := s.EnsureFood(f)
		if e != nil {
			return nil, nil, e
		}
		ensured = append(ensured, got)
		if isNew {
			created = append(created, got)
		}
	}
	added, e := s.PrepareMealSlot(meal, nil)
	if e != nil {
		return nil, nil, e
	}
	created = dedupeFoodsByID(append(created, added...))
	return ensured, created, nil
}

func (s *Service) handleDayMealDelete(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	d, err := st.DeleteMealSlot(c.Param("date"), c.Param("slot"))
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"day": d})
}

func (s *Service) handleImportJSONStatus(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	dir := st.UserDir()
	c.JSON(http.StatusOK, gin.H{
		"legacy_present": LegacyJSONPresent(dir),
		"db_empty":       st.IsEmpty(),
	})
}

func (s *Service) handleImportJSON(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	merge := c.Query("merge") == "true" || c.Query("merge") == "1"
	source := strings.ToLower(c.DefaultQuery("source", "auto"))

	var bundle JSONBundle
	if c.Request.ContentLength > 0 {
		if err := c.BindJSON(&bundle); err != nil && source == "body" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
			return
		}
	}
	if !bundleHasData(bundle) {
		switch source {
		case "auto", "", "legacy":
			var err error
			bundle, err = loadLegacyJSONBundle(st.UserDir())
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		case "body":
			c.JSON(http.StatusBadRequest, gin.H{"error": "empty body", "hint": "POST JSON with weights, foods, days"})
			return
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown source", "hint": "source=auto|legacy|body"})
			return
		}
	}
	if !bundleHasData(bundle) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "no data",
			"hint":  "place legacy JSON in your import folder, POST a bundle with source=body, or use source=legacy",
		})
		return
	}
	counts, err := st.ImportBundle(bundle, merge)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"imported": counts,
		"merge":    merge,
	})
}

func bundleHasData(b JSONBundle) bool {
	return len(b.Weights) > 0 || len(b.Foods) > 0 || len(b.Days) > 0
}

func (s *Service) handleImportMarkdown(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	replace := !(c.Query("merge") == "true" || c.Query("merge") == "1")
	w, f, d, stock, err := ImportFromMDDir(s.Config.MDDataDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(w) == 0 && len(f) == 0 && len(d) == 0 && len(stock) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no markdown data in mddata"})
		return
	}
	if replace {
		if err := st.ReplaceImported(w, f, d, stock); err != nil {
			writeStoreErr(c, err)
			return
		}
	} else {
		bundle := JSONBundle{Weights: w, Foods: f, Days: d, Stock: stock}
		if _, err := st.ImportBundle(bundle, true); err != nil {
			writeStoreErr(c, err)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"imported": map[string]int{
			"weights": len(w),
			"foods":   len(f),
			"days":    len(d),
			"stock":   len(stock),
		},
		"replace": replace,
		"source":  s.Config.MDDataDir,
	})
}

func (s *Service) handleStockList(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	activeOnly := c.Query("active") == "1" || c.Query("active") == "true"
	lots, err := st.ListStockLots(activeOnly)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"lots": lots, "count": len(lots)})
}

func (s *Service) handleStockGet(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	lot, found := st.StockLotByID(c.Param("id"))
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"lot": lot})
}

func (s *Service) handleStockCreate(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	var lot StockLot
	if err := c.BindJSON(&lot); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	out, err := st.UpsertStockLot(lot)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"lot": out})
}

func (s *Service) handleStockUpdate(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	var lot StockLot
	if err := c.BindJSON(&lot); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	lot.ID = c.Param("id")
	out, err := st.UpsertStockLot(lot)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"lot": out})
}

func (s *Service) handleStockDelete(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	if err := st.DeleteStockLot(c.Param("id")); err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": c.Param("id")})
}

func (s *Service) handleStockDraw(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	var body StockDraw
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if body.Date == "" {
		body.Date = todayAthens()
	}
	day, lot, err := st.DrawFromStock(c.Param("id"), body)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"day": day, "lot": lot})
}

func (s *Service) handleStockAdjust(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	var body StockAdjust
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	lot, err := st.AdjustFromStock(c.Param("id"), body)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"lot": lot})
}

func (s *Service) handleStockActive(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	var body struct {
		Active bool `json:"active"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	lot, err := st.SetStockLotActive(c.Param("id"), body.Active)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"lot": lot})
}

func writeStoreErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"error": "conflict", "hint": "use PUT to update existing food"})
	case errors.Is(err, ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		if strings.Contains(err.Error(), string(ErrInvalidInput.Error())) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
