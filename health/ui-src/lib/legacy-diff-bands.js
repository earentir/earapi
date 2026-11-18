/** Day header: color |catalog − markdown| using rising thresholds (defaults 100 / 200 / 300). */

import { DEFAULT_KCAL_BUDGET } from "./kcal-budget.js";

export const DEFAULT_LEGACY_DIFF_BANDS = {
  legacy_diff_green_lt: DEFAULT_KCAL_BUDGET.legacy_diff_green_lt,
  legacy_diff_yellow_lt: DEFAULT_KCAL_BUDGET.legacy_diff_yellow_lt,
  legacy_diff_orange_lt: DEFAULT_KCAL_BUDGET.legacy_diff_orange_lt,
};

export function mergedLegacyDiffBands(settings) {
  const s = settings || {};
  const g = Number(s.legacy_diff_green_lt);
  const y = Number(s.legacy_diff_yellow_lt);
  const o = Number(s.legacy_diff_orange_lt);
  let green = Number.isFinite(g) && g > 0 ? g : DEFAULT_LEGACY_DIFF_BANDS.legacy_diff_green_lt;
  let yellow = Number.isFinite(y) && y > 0 ? y : DEFAULT_LEGACY_DIFF_BANDS.legacy_diff_yellow_lt;
  let orange = Number.isFinite(o) && o > 0 ? o : DEFAULT_LEGACY_DIFF_BANDS.legacy_diff_orange_lt;
  if (!(green < yellow && yellow < orange)) {
    return { ...DEFAULT_LEGACY_DIFF_BANDS };
  }
  return {
    legacy_diff_green_lt: green,
    legacy_diff_yellow_lt: yellow,
    legacy_diff_orange_lt: orange,
  };
}

/** @returns {string} band key green|yellow|orange|red */
export function legacyDiffBandClass(delta, settings) {
  const d = Math.abs(Number(delta) || 0);
  if (d <= 0) return "green";
  const b = mergedLegacyDiffBands(settings);
  if (d < b.legacy_diff_green_lt) return "green";
  if (d < b.legacy_diff_yellow_lt) return "yellow";
  if (d < b.legacy_diff_orange_lt) return "orange";
  return "red";
}

export function catalogLegacyDeltaBand(computed, legacy, settings) {
  const leg = Number(legacy) || 0;
  if (leg <= 0) return "muted";
  const cur = Number(computed) || 0;
  return legacyDiffBandClass(cur - leg, settings);
}
