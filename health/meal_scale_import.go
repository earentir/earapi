package health

// ScaleDayMealsFromCatalog links food_id and fills line kcal/sodium from Foods + Stock during markdown import.
func ScaleDayMealsFromCatalog(d *Day, foods []Food, stock []StockLot, aliasFoods map[string]Food) {
	if d == nil {
		return
	}
	clearDayItemMacros(d)
	clearMealStockLotLinks(d)
	for slot, m := range d.Meals {
		ScaleMealSlotFromCatalog(&m, foods, stock, aliasFoods)
		RecomputeMealSlotTotalsFromItems(&m)
		d.Meals[slot] = m
	}
}

func clearMealStockLotLinks(d *Day) {
	if d == nil || d.Meals == nil {
		return
	}
	var walk func(*MealItem)
	walk = func(it *MealItem) {
		if it == nil {
			return
		}
		it.StockLotID = ""
		for i := range it.Parts {
			walk(&it.Parts[i])
		}
	}
	for slot := range d.Meals {
		m := d.Meals[slot]
		for i := range m.Items {
			walk(&m.Items[i])
		}
		d.Meals[slot] = m
	}
}

func clearDayItemMacros(d *Day) {
	var walk func(*MealItem)
	walk = func(it *MealItem) {
		if it == nil {
			return
		}
		it.Kcal = 0
		it.SodiumMg = 0
		for i := range it.Parts {
			walk(&it.Parts[i])
		}
	}
	for slot := range d.Meals {
		m := d.Meals[slot]
		for i := range m.Items {
			walk(&m.Items[i])
		}
		d.Meals[slot] = m
	}
}

// RecomputeMealSlotTotalsFromItems sets slot kcal/sodium from item lines (after catalog scale).
func RecomputeMealSlotTotalsFromItems(meal *MealSlot) {
	if meal == nil {
		return
	}
	var kcal, sod float64
	for _, it := range meal.Items {
		kcal += it.Kcal
		sod += it.SodiumMg
	}
	meal.Kcal = kcal
	meal.SodiumMg = sod
	meal.KcalMargin = 0
}

func ScaleMealSlotFromCatalog(meal *MealSlot, foods []Food, stock []StockLot, aliasFoods map[string]Food) {
	if meal == nil {
		return
	}
	normalizeMealItems(meal.Items)
	for i := range meal.Items {
		scaleItemImport(&meal.Items[i], foods, stock, aliasFoods)
	}
	FinalizeMealTotals(meal)
}

func scaleItemImport(it *MealItem, foods []Food, stock []StockLot, aliasFoods map[string]Food) {
	if it == nil {
		return
	}
	if len(it.Parts) > 0 {
		for i := range it.Parts {
			scaleItemImport(&it.Parts[i], foods, stock, aliasFoods)
		}
		_ = scaleDishWithCatalog(it, foods, aliasFoods)
		return
	}
	if it.Grams <= 0 {
		FillMealItemGramsFromCount(it, foods)
	}
	if it.Grams <= 0 {
		return
	}
	// Meals use Food Data + aliases only. Stock lots are for prep batches you track in Stock, not auto-linked on import.
	_ = stock
	_ = scaleItemFromFoodWithCatalog(it, foods, aliasFoods)
}
