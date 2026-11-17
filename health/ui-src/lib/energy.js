export const SEDENTARY_PAL = 1.2;

export function normalizeSex(s) {
  const v = String(s || "").trim().toLowerCase();
  if (v === "m" || v === "male" || v === "man") return "male";
  if (v === "f" || v === "female" || v === "woman") return "female";
  return "";
}

/** Mifflin-St Jeor BMR (kcal/day). */
export function mifflinStJeorBmr({ kg, heightCm, ageYears, sex }) {
  const w = Number(kg);
  const h = Number(heightCm);
  const age = Number(ageYears);
  const sx = normalizeSex(sex);
  if (!(w > 0) || !(h > 0) || !(age > 0) || !sx) return null;
  const adj = sx === "male" ? 5 : -161;
  const bmr = 10 * w + 6.25 * h - 5 * age + adj;
  if (!Number.isFinite(bmr) || bmr <= 0) return null;
  return Math.round(bmr);
}

export function clampSedentaryDiscount(p) {
  const n = Number(p);
  if (!Number.isFinite(n) || n <= 0) return 0;
  return Math.min(50, Math.round(n));
}

export function sedentaryTdee(bmr, discountPercent = 0) {
  const n = Number(bmr);
  if (!(n > 0)) return null;
  const disc = clampSedentaryDiscount(discountPercent);
  return Math.round(n * SEDENTARY_PAL * (1 - disc / 100));
}

export function energyFromInputs({ kg, heightCm, ageYears, sex, sedentary_discount_percent: discount }) {
  const bmr = mifflinStJeorBmr({ kg, heightCm, ageYears, sex });
  if (bmr == null) return null;
  const disc = clampSedentaryDiscount(discount);
  const sedentary = sedentaryTdee(bmr, disc);
  const sedentaryBase = sedentaryTdee(bmr, 0);
  return { bmr, sedentary, sedentaryBase, sedentaryDiscountPercent: disc, kg: Number(kg) };
}

/** Wishnofsky: 7700 kcal ≈ 1 kg body mass. */
export const KCAL_PER_KG = 7700;

/**
 * Estimated mass change in grams (intake vs sedentary TDEE).
 * Positive = gain, negative = loss.
 */
export function estimatedMassDeltaGrams(intakeKcal, sedentaryKcal) {
  const eat = Number(intakeKcal);
  const burn = Number(sedentaryKcal);
  if (!(eat > 0) || !(burn > 0)) return null;
  return Math.round(((eat - burn) * 1000) / KCAL_PER_KG);
}

export function kgOnOrBefore(date, weights) {
  const day = String(date || "").trim();
  const rows = (weights || [])
    .filter((w) => w?.date && Number(w.kg) > 0)
    .slice()
    .sort((a, b) => String(a.date).localeCompare(String(b.date)));
  if (!rows.length || !day) return 0;
  let kg = 0;
  for (const w of rows) {
    if (String(w.date) <= day) kg = Number(w.kg);
    else break;
  }
  return kg > 0 ? kg : Number(rows[0].kg) || 0;
}