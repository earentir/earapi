export function emptyItem() {
  return {
    name: "",
    grams: "",
    count: "",
    food_id: "",
    stock_lot_id: "",
    parts: [],
    kcal_per_100g: "",
    sodium_mg_per_100g: "",
    note: "",
    is_set: false,
  };
}

export function emptySlot() {
  return { items: [emptyItem()], note: "" };
}

export function emptyDraft() {
  return { lunch: emptySlot(), dinner: emptySlot(), snack: emptySlot(), bp: "", walking: "" };
}

export function emptyStockSet() {
  const it = emptyItem();
  it.is_set = true;
  it.name = "";
  it.parts = [emptyItem()];
  return it;
}

export function emptyStockForm() {
  return {
    prepared_date: state.date || "",
    batch_grams: "",
    remaining_grams: "",
    active: true,
    notes: "",
    set: emptyStockSet(),
  };
}

export const state = {
  user: "",
  token: "",
  tab: "today",
  date: "",
  foods: [],
  foodAliases: [],
  foodPick: null,
  day: null,
  /** Server-scaled copy for Today UI (POST /nutrition/compute/day). */
  computedDay: null,
  days: [],
  daysQ: "",
  daysStatusFilter: "all",
  dayGapItems: [],
  foodsQ: "",
  weights: [],
  stock: [],
  stockForm: null,
  overview: null,
  kcalBudget: null,
  mealDraft: emptyDraft(),
  toast: "",
  busy: false,
  suggest: { slot: "", i: -1, q: "" },
};
