/** Default g per unit when markdown has ×N but no grams (matches health/itemCountGuessGrams). */
export function defaultGramsPerUnit(nameHint) {
  const low = (nameHint || "").toLowerCase();
  if (
    low.includes("wafer")
    || low.includes("rusk")
    || low.includes("bar")
    || low.includes("sachet")
    || low.includes("croissant")
    || low.includes("granola")
  ) {
    return 10;
  }
  return 0;
}

export function effectiveItemGrams(it) {
  const g = Number(it?.grams) || 0;
  if (g > 0) return g;
  const c = Number(it?.count) || 0;
  if (c <= 0) return 0;
  const per = defaultGramsPerUnit(it?.name || it?.note || "");
  return per > 0 ? c * per : 0;
}
