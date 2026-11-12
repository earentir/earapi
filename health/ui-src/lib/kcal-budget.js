/** Defaults from Energy baseline in Health Tracker - Misc.md (1,500-1,700 band; 2,200 gain). */
export const DEFAULT_KCAL_BUDGET = {
  soft_low_kcal: 1500,
  soft_high_kcal: 1700,
  gain_min_kcal: 2200,
  meal_margin_percent: 10,
};

export function mergedKcalBudget(raw) {
  if (!raw) return { ...DEFAULT_KCAL_BUDGET };
  const margin = Number(raw.meal_margin_percent);
  return {
    soft_low_kcal: Number(raw.soft_low_kcal) || DEFAULT_KCAL_BUDGET.soft_low_kcal,
    soft_high_kcal: Number(raw.soft_high_kcal) || DEFAULT_KCAL_BUDGET.soft_high_kcal,
    gain_min_kcal: Number(raw.gain_min_kcal) || DEFAULT_KCAL_BUDGET.gain_min_kcal,
    meal_margin_percent: Number.isFinite(margin) && margin >= 0
      ? margin
      : DEFAULT_KCAL_BUDGET.meal_margin_percent,
  };
}

/** @returns {"green"|"yellow"|"orange"|"red"|"none"} */
export function kcalBudgetBand(kcal, settings) {
  const s = mergedKcalBudget(settings);
  const n = Number(kcal);
  if (!Number.isFinite(n) || n <= 0) return "none";
  if (n < s.soft_low_kcal) return "green";
  if (n <= s.soft_high_kcal) return "yellow";
  if (n < s.gain_min_kcal) return "orange";
  return "red";
}

const BAND_LABEL = {
  green: "Below soft intake band",
  yellow: "In soft intake band",
  orange: "Above soft band (hard limit)",
  red: "Weight-gain territory",
};

export function kcalBudgetBandLabel(band) {
  return BAND_LABEL[band] || "";
}

export function dayKcalBarParts(band) {
  if (!band || band === "none") {
    return { hostClass: "", barClass: "", title: "" };
  }
  return {
    hostClass: " day-kcal-bar-host",
    barClass: band,
    title: kcalBudgetBandLabel(band),
  };
}
