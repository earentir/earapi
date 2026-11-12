/** Meal slot total color bands (kcal); sodium uses the same band for visual pairing. */

export function mealEnergyBand(kcal) {
  const k = Number(kcal) || 0;
  if (k <= 0) return "meal-band--muted";
  if (k < 100) return "meal-band--green";
  if (k < 200) return "meal-band--yellow";
  if (k < 300) return "meal-band--orange";
  return "meal-band--red";
}
