package health

// ScaleDayMealsFromCatalog links food_id and fills line kcal/sodium from Foods + Stock during markdown import.
func ScaleDayMealsFromCatalog(d *Day, foods []Food, stock []StockLot, aliasFoods map[string]Food) {
	if d == nil {
		return
	}
	clearDayItemMacros(d)
	for slot, m := range d.Meals {
		ScaleMealSlotFromCatalog(&m, foods, stock, aliasFoods)
		RecomputeMealSlotTotalsFromItems(&m)
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
	if scaleImportItemFromStock(it, stock) {
		return
	}
	_ = scaleItemFromFoodWithCatalog(it, foods, aliasFoods)
}

func scaleImportItemFromStock(it *MealItem, lots []StockLot) bool {
	if it == nil || it.Grams <= 0 || len(lots) == 0 {
		return false
	}
	id := matchMealItemToStockLot(it.Name, lots)
	if id == "" {
		return false
	}
	var lot *StockLot
	for i := range lots {
		if lots[i].ID == id {
			lot = &lots[i]
			break
		}
	}
	if lot == nil {
		return false
	}
	it.StockLotID = id
	return scaleMealItemFromStockLotMacros(it, lot)
}
