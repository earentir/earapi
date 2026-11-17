/** Defaults from Energy baseline in Health Tracker - Misc.md (1,500-1,700 band; 2,200 gain). */
export const DEFAULT_KCAL_BUDGET = {
  soft_low_kcal: 1500,
  soft_high_kcal: 1700,
  gain_min_kcal: 2200,
  soft_low_sodium_mg: 1200,
  soft_high_sodium_mg: 2000,
  ceiling_sodium_mg: 2400,
  meal_margin_percent: 10,
  legacy_diff_green_lt: 100,
  legacy_diff_yellow_lt: 200,
  legacy_diff_orange_lt: 300,
  sex: "",
  age_years: 0,
  height_cm: 0,
  sedentary_discount_percent: 0,
};

export function mergedKcalBudget(raw) {
  if (!raw) return { ...DEFAULT_KCAL_BUDGET };
  const margin = Number(raw.meal_margin_percent);
  const g = Number(raw.legacy_diff_green_lt);
  const y = Number(raw.legacy_diff_yellow_lt);
  const o = Number(raw.legacy_diff_orange_lt);
  const age = Number(raw.age_years);
  const height = Number(raw.height_cm);
  const merged = {
    soft_low_kcal: Number(raw.soft_low_kcal) || DEFAULT_KCAL_BUDGET.soft_low_kcal,
    soft_high_kcal: Number(raw.soft_high_kcal) || DEFAULT_KCAL_BUDGET.soft_high_kcal,
    gain_min_kcal: Number(raw.gain_min_kcal) || DEFAULT_KCAL_BUDGET.gain_min_kcal,
    soft_low_sodium_mg: Number(raw.soft_low_sodium_mg) || DEFAULT_KCAL_BUDGET.soft_low_sodium_mg,
    soft_high_sodium_mg: Number(raw.soft_high_sodium_mg) || DEFAULT_KCAL_BUDGET.soft_high_sodium_mg,
    ceiling_sodium_mg: Number(raw.ceiling_sodium_mg) || DEFAULT_KCAL_BUDGET.ceiling_sodium_mg,
    meal_margin_percent: Number.isFinite(margin) && margin >= 0
      ? margin
      : DEFAULT_KCAL_BUDGET.meal_margin_percent,
    legacy_diff_green_lt: Number.isFinite(g) && g > 0 ? g : DEFAULT_KCAL_BUDGET.legacy_diff_green_lt,
    legacy_diff_yellow_lt: Number.isFinite(y) && y > 0 ? y : DEFAULT_KCAL_BUDGET.legacy_diff_yellow_lt,
    legacy_diff_orange_lt: Number.isFinite(o) && o > 0 ? o : DEFAULT_KCAL_BUDGET.legacy_diff_orange_lt,
    sex: String(raw.sex || "").trim().toLowerCase() === "female" ? "female"
      : String(raw.sex || "").trim().toLowerCase() === "male" ? "male" : "",
    age_years: Number.isFinite(age) && age > 0 ? Math.round(age) : 0,
    height_cm: Number.isFinite(height) && height > 0 ? height : 0,
    sedentary_discount_percent: (() => {
      const n = Number(raw.sedentary_discount_percent);
      if (!Number.isFinite(n) || n <= 0) return 0;
      return Math.min(50, Math.round(n));
    })(),
  };
  if (
    !(merged.legacy_diff_green_lt < merged.legacy_diff_yellow_lt &&
      merged.legacy_diff_yellow_lt < merged.legacy_diff_orange_lt)
  ) {
    merged.legacy_diff_green_lt = DEFAULT_KCAL_BUDGET.legacy_diff_green_lt;
    merged.legacy_diff_yellow_lt = DEFAULT_KCAL_BUDGET.legacy_diff_yellow_lt;
    merged.legacy_diff_orange_lt = DEFAULT_KCAL_BUDGET.legacy_diff_orange_lt;
  }
  if (
    !(merged.soft_low_sodium_mg < merged.soft_high_sodium_mg &&
      merged.soft_high_sodium_mg < merged.ceiling_sodium_mg)
  ) {
    merged.soft_low_sodium_mg = DEFAULT_KCAL_BUDGET.soft_low_sodium_mg;
    merged.soft_high_sodium_mg = DEFAULT_KCAL_BUDGET.soft_high_sodium_mg;
    merged.ceiling_sodium_mg = DEFAULT_KCAL_BUDGET.ceiling_sodium_mg;
  }
  return merged;
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

/** Intake bar band from stored day kcal + budget (same for list bar, chart, filter). */
export function dayIntakeBand(d, budgetRaw) {
  return kcalBudgetBand(d?.totals?.kcal, budgetRaw);
}

const INTAKE_BAND_IDS = ["green", "yellow", "orange", "red", "none"];

/** Count days per intake band in the current loaded set. */
export function dayBandCounts(days, budgetRaw) {
  const list = days || [];
  const counts = { all: list.length, green: 0, yellow: 0, orange: 0, red: 0, none: 0 };
  for (const d of list) {
    const b = dayIntakeBand(d, budgetRaw);
    if (INTAKE_BAND_IDS.includes(b)) counts[b]++;
    else counts.none++;
  }
  return counts;
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

export function dayKcalBarParts(band, title = "") {
  if (!band || band === "none") {
    return { hostClass: "", barClass: "", title: "" };
  }
  return {
    hostClass: " day-kcal-bar-host",
    barClass: band,
    title: title || kcalBudgetBandLabel(band),
  };
}
