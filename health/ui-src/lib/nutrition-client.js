import { api } from "./api.js";
import { state } from "./state.js";
import { SLOTS } from "./constants.js";

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
  if (band === "muted") return "meal-band--muted";
  return `meal-band--${band}`;
}

export function computedMealItem(slot, idx, partIdx) {
  const day = nutritionDay();
  let it = day?.meals?.[slot]?.items?.[idx];
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

export function itemGapRef(slot, idx, partIdx) {
  const date = state.date || nutritionDay()?.date || "";
  if (partIdx == null) return `${date}/${slot}/${idx}`;
  return `${date}/${slot}/${idx}/${partIdx}`;
}

export function itemGapFor(slot, idx, partIdx) {
  const gaps = dayDisplay().item_gaps || [];
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
  if (out.display && state.computedDay) {
    state.computedDay.display = out.display;
  }
  return state.computedDay;
}
