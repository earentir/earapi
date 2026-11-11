package health

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes mounts /health/v1 routes.
func RegisterRoutes(r *gin.Engine, svc *Service) {
	g := r.Group("/health/v1")
	g.GET("/help", svc.handleHelp)

	auth := AuthMiddleware(svc.authUsers)
	p := g.Group("")
	p.Use(auth)
	{
		p.GET("/weight/latest", svc.handleWeightLatest)
		p.GET("/weight", svc.handleWeightList)
		p.POST("/weight", svc.handleWeightAdd)

		p.GET("/foods", svc.handleFoodsList)
		p.GET("/foods/:id", svc.handleFoodGet)
		p.POST("/foods", svc.handleFoodAdd)
		p.PUT("/foods/:id", svc.handleFoodUpdate)

		p.GET("/days", svc.handleDaysList)
		p.GET("/days/:date", svc.handleDayGet)
		p.POST("/days", svc.handleDayCreate)
		p.PUT("/days/:date", svc.handleDayPut)
		p.DELETE("/days/:date", svc.handleDayDelete)

		p.POST("/days/:date/meals", svc.handleDayMealPost)
		p.PUT("/days/:date/meals/:slot", svc.handleDayMealPut)
		p.DELETE("/days/:date/meals/:slot", svc.handleDayMealDelete)

		p.GET("/import/json/status", svc.handleImportJSONStatus)
		p.POST("/import/json", svc.handleImportJSON)
		p.POST("/import/markdown", svc.handleImportMarkdown)
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
		"auth": "Send X-User and X-API-Key headers (or ?user=&token= on GET). Bearer token works with X-User.",
		"routes": []string{
			"GET /health/v1/help",
			"GET /health/v1/weight/latest",
			"GET /health/v1/weight?from=&to=",
			"POST /health/v1/weight",
			"GET /health/v1/foods?q=&limit=",
			"GET /health/v1/foods/:id",
			"POST /health/v1/foods",
			"PUT /health/v1/foods/:id",
			"GET /health/v1/days?from=&to=&date=&q=&slot=&limit=",
			"GET /health/v1/days/:date",
			"POST /health/v1/days",
			"PUT /health/v1/days/:date",
			"DELETE /health/v1/days/:date",
			"POST /health/v1/days/:date/meals",
			"PUT /health/v1/days/:date/meals/:slot",
			"DELETE /health/v1/days/:date/meals/:slot",
			"GET /health/v1/import/json/status",
			"POST /health/v1/import/json?merge=false&source=legacy",
			"POST /health/v1/import/markdown?replace=true",
		},
		"examples": gin.H{
			"add_weight": gin.H{"date": "2026-11-11", "kg": 142.5, "note": "morning post-BM", "protocol": true},
			"add_meal": gin.H{
				"slot": "lunch",
				"items": []gin.H{
					{"name": "Pasta (cooked, no salt)", "grams": 120},
					{"name": "Chicken breast (cooked, no salt)", "grams": 80},
				},
				"kcal_margin": 0.10,
			},
		},
	})
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
	c.JSON(http.StatusOK, gin.H{"foods": all, "count": len(all)})
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
	if existing, found := st.GetDay(d.Date); found {
		// merge meals
		for slot, m := range d.Meals {
			if cur, ok := existing.Meals[slot]; ok && !mealEmpty(cur) {
				cur.Items = append(cur.Items, m.Items...)
				if m.Raw != "" {
					cur.Raw = strings.TrimSpace(cur.Raw + " · " + m.Raw)
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

func (s *Service) handleDayMealPost(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	date := c.Param("date")
	var body struct {
		Slot        string     `json:"slot"`
		Items       []MealItem `json:"items"`
		Note        string     `json:"note"`
		Raw         string     `json:"raw"`
		KcalMargin  float64    `json:"kcal_margin"`
		SodiumMg    float64    `json:"sodium_mg"`
		Kcal        float64    `json:"kcal"`
		Replace     bool       `json:"replace"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if body.Slot == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slot required"})
		return
	}
	meal := MealSlot{
		Raw: body.Raw, Items: body.Items, KcalMargin: body.KcalMargin,
		SodiumMg: body.SodiumMg, Kcal: body.Kcal,
	}
	if body.Note != "" {
		if meal.Raw != "" {
			meal.Raw += " · " + body.Note
		} else {
			meal.Raw = body.Note
		}
	}
	for i := range meal.Items {
		if err := st.ScaleItemFromFood(&meal.Items[i]); err != nil {
			writeStoreErr(c, err)
			return
		}
	}
	if len(meal.Items) > 0 && meal.Kcal == 0 {
		for _, it := range meal.Items {
			meal.Kcal += it.Kcal
			meal.SodiumMg += it.SodiumMg
		}
		if meal.KcalMargin > 0 {
			meal.Kcal *= (1 + meal.KcalMargin)
		}
	}
	d, err := st.UpsertMealSlot(date, body.Slot, meal, body.Replace)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"day": d})
}

func (s *Service) handleDayMealPut(c *gin.Context) {
	st, ok := s.store(c)
	if !ok {
		return
	}
	date := c.Param("date")
	slot := c.Param("slot")
	var meal MealSlot
	if err := c.BindJSON(&meal); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	for i := range meal.Items {
		if err := st.ScaleItemFromFood(&meal.Items[i]); err != nil {
			writeStoreErr(c, err)
			return
		}
	}
	d, err := st.SetMealSlot(date, slot, meal)
	if err != nil {
		writeStoreErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"day": d})
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
	w, f, d, err := ImportFromMDDir(s.Config.MDDataDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(w) == 0 && len(f) == 0 && len(d) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no markdown data in mddata"})
		return
	}
	if replace {
		if err := st.ReplaceImported(w, f, d); err != nil {
			writeStoreErr(c, err)
			return
		}
	} else {
		bundle := JSONBundle{Weights: w, Foods: f, Days: d}
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
		},
		"replace": replace,
		"source":  s.Config.MDDataDir,
	})
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
