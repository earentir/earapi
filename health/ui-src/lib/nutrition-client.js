import { api } from "./api.js";
import { state } from "./state.js";
import { SLOTS } from "./constants.js";
const BAND_TEXT = {
  muted: "text-base-content/50",
  green: "text-success",
  yellow: "text-warning",
  orange: "text-warning",
  red: "text-error",
  none: "",
};

let computeTimer = null;

/** Authoritative day for kcal, sodium, bands (server compute). */
export function nutritionDay() {
  return state.computedDay || state.day;
}

export function dayDisplay(d) {
  const day = d || nutritionDay();
  return day?.display || state.day?.display || {};
}

export function mealBandClass(band) {
  if (!band || band === "none") return "";
  return BAND_TEXT[band] || BAND_TEXT.muted;
}

export function computedMealItem(slot, idx, partIdx) {
  let it;
  if (slot === "stock") {
    it = state.computedStockMeal?.items?.[idx];
  } else {
    it = nutritionDay()?.meals?.[slot]?.items?.[idx];
  }
  if (!it) return null;
  if (partIdx != null) it = it.parts?.[partIdx];
  return it || null;
}

export function macroAtSlot(slot, field) {
  const m = nutritionDay()?.meals?.[slot];
  if (!m) return null;
  const v = Number(m[field === "sodium" ? "sodium_mg" : "kcal"]);
  return Number.isFinite(v) && v > 0 ? v : (field === "sodium" ? 0 : 0);
}

export function dayTotalsView() {
  const t = nutritionDay()?.totals;
  return { kcal: Number(t?.kcal) || 0, sodium_mg: Number(t?.sodium_mg) || 0 };
}

/** Stored markdown legacy + computed totals/display (for legacy diff dots). */
export function legacyDayForDiff() {
  const computed = nutritionDay();
  const stored = state.day;
  if (!stored) return computed || {};
  return {
    ...(computed || stored),
    legacy_totals: stored.legacy_totals,
    display: computed?.display || stored?.display,
    totals: computed?.totals ?? stored?.totals,
  };
}

export function mealSlotRaw(slot) {
  return state.day?.meals?.[slot]?.raw || nutritionDay()?.meals?.[slot]?.raw || "";
}

export function itemGapRef(slot, idx, partIdx) {
  const date = state.date || nutritionDay()?.date || "";
  if (partIdx == null) return `${date}/${slot}/${idx}`;
  return `${date}/${slot}/${idx}/${partIdx}`;
}

export function itemGapFor(slot, idx, partIdx) {
  const gaps = slot === "stock"
    ? (state.computedStockDisplay?.item_gaps || [])
    : (dayDisplay().item_gaps || []);
  const ref = itemGapRef(slot, idx, partIdx);
  return gaps.find((g) => g.ref === ref) || null;
}

export function buildDayComputeBody(draftItems, mealDraft) {
  const meals = {};
  for (const slot of SLOTS) {
    meals[slot] = {
      items: draftItems(slot),
      raw: state.day?.meals?.[slot]?.raw || "",
    };
  }
  return {
    date: state.date,
    meals,
    legacy_totals: state.day?.legacy_totals,
    bp: mealDraft?.bp,
    walking: mealDraft?.walking,
  };
}

export function scheduleDayNutritionCompute(draftItems, mealDraft, onDone) {
  if (state.tab !== "today" || !state.user || !state.token) return;
  clearTimeout(computeTimer);
  computeTimer = setTimeout(() => {
    refreshDayNutrition(draftItems, mealDraft).then(onDone).catch(() => {});
  }, 400);
}

export async function refreshDayNutrition(draftItems, mealDraft) {
  const body = buildDayComputeBody(draftItems, mealDraft);
  const out = await api("/nutrition/compute/day", { method: "POST", body: JSON.stringify(body) });
  state.computedDay = out.day || null;
  if (state.computedDay) {
    if (out.display) state.computedDay.display = out.display;
    const leg = state.day?.legacy_totals;
    if (leg && (Number(leg.kcal) > 0 || Number(leg.sodium_mg) > 0)) {
      state.computedDay.legacy_totals = leg;
    }
  }
  return state.computedDay;
}

let stockComputeSeq = 0;

/** Scale a stock batch recipe in the editor (does not persist). */
export async function refreshStockNutrition(meal) {
  const seq = ++stockComputeSeq;
  if (!meal?.items?.length) {
    if (seq === stockComputeSeq) {
      state.computedStockMeal = null;
      state.computedStockDisplay = null;
    }
    return null;
  }
  const out = await api("/nutrition/compute/meal", {
    method: "POST",
    body: JSON.stringify({ date: state.date, slot: "stock", meal }),
  });
  if (seq !== stockComputeSeq) return null;
  state.computedStockMeal = out.meal || null;
  state.computedStockDisplay = out.display || null;
  return out;
}

export function clearStockNutrition() {
  stockComputeSeq += 1;
  state.computedStockMeal = null;
  state.computedStockDisplay = null;
}
