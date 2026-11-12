export const DEFAULT_MEAL_MARGIN_PERCENT = 10;

export function mealMarginPercent(settings) {
  if (!settings) return DEFAULT_MEAL_MARGIN_PERCENT;
  const n = Number(settings.meal_margin_percent);
  if (!Number.isFinite(n) || n < 0) return DEFAULT_MEAL_MARGIN_PERCENT;
  return n;
}

/** Apply +margin% to a slot total (kcal + sodium). */
export function applyMealMarginTotals(totals, marginPercent) {
  if (!totals || !marginPercent || marginPercent <= 0) return totals;
  const f = 1 + marginPercent / 100;
  return {
    kcal: totals.kcal != null ? Math.round(totals.kcal * f) : totals.kcal,
    sodium_mg: totals.sodium_mg != null ? Math.round(totals.sodium_mg * f) : totals.sodium_mg,
  };
}
