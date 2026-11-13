package health

import "strings"

func containsAll(low string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(low, p) {
			return false
		}
	}
	return true
}

// canonicalMealFoodName maps frequent markdown meal labels to catalog names before fuzzy match.
func canonicalMealFoodName(name string) string {
	low := strings.ToLower(strings.TrimSpace(name))
	if low == "" {
		return name
	}
	switch {
	case containsAll(low, "charcoal", "skewer"):
		// Markdown uses this for weighed takeout / tray portions (not the Oct 7 homemade batch row).
		return "chicken souvlaki"
	case containsAll(low, "chicken", "skewer"):
		if strings.Contains(low, "homemade") {
			return "Homemade chicken skewers (Oct 7, no oil)"
		}
		if strings.Contains(low, "delivery") {
			return "Chicken skewer (delivery, verified Sep 30)"
		}
		// Meal logs say "chicken skewer" for the same tray portions as chicken souvlaki (not the 98 kcal delivery row).
		return "chicken souvlaki"
	case containsAll(low, "pork", "souvlaki"):
		return "pork skewer"
	case containsAll(low, "chicken", "souvlaki"):
		return "chicken souvlaki"
	case strings.Contains(low, "bonora") && strings.Contains(low, "chocolate"):
		return "Bonora milk chocolate bar"
	case strings.Contains(low, "fasolada") || (strings.Contains(low, "gigantes") && strings.Contains(low, "supermarket")):
		return "Sklavenitis gigantes (fasolada) ready meal"
	case strings.Contains(low, "rusk"):
		return "rusks no salt"
	case low == "grana" || strings.HasPrefix(low, "grana "):
		return "Grana Padano"
	case containsAll(low, "moussaka", "ready"):
		return "moussaka ready meal"
	case strings.Contains(low, "moussaka"):
		return "moussaka"
	case low == "tyrokateri" || strings.Contains(low, "tyrokateri"):
		return "tyrokafteri"
	case strings.Contains(low, "leaves and cabbage"):
		return "cabbage"
	case strings.Contains(low, "cabbage yellow"):
		return "cabbage"
	case strings.Contains(low, "2% yogurt") || low == "2% yogurt":
		return "2% yogurt"
	case strings.Contains(low, "marata"):
		return "Marata 1.7%"
	case strings.Contains(low, "vanilies") || low == "vanilia":
		return "Vanilia"
	case strings.Contains(low, "soup-tomato"):
		return "soup-tomato mix"
	case strings.Contains(low, "carcass soup"):
		return "carcass soup"
	case strings.Contains(low, "sole-tomato"):
		return "sole-tomato sauce"
	case strings.Contains(low, "boiled egg"):
		return "Egg (1 large, ~55 g)"
	case strings.Contains(low, "green salad"):
		return "iceberg"
	case low == "beef" || strings.HasPrefix(low, "beef "):
		return "Lean beef steak"
	case low == "sauce":
		return "sauce"
	case strings.Contains(low, "dried fried onion"):
		return "onion"
	case strings.Contains(low, "bean salad"):
		return "bean salad"
	case strings.Contains(low, "corn wafer"):
		return "corn wafers"
	case strings.Contains(low, "white bread"):
		return "white bread"
	case strings.Contains(low, "florina"):
		return "florina pepper"
	case strings.Contains(low, "crimson grapes"):
		return "crimson grapes"
	case strings.Contains(low, "roasted potato") || strings.Contains(low, "roasted potatoes"):
		return "Roasted potato"
	case strings.Contains(low, "roasted bell pepper"):
		return "Roasted bell pepper"
	case strings.Contains(low, "baby leaves"):
		return "baby leaves"
	case strings.Contains(low, "mayo-ketchup"):
		return "mayo-ketchup sauce"
	}
	return name
}
