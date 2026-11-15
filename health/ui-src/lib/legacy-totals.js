/** Markdown day-column totals (hover only) vs catalog-computed day totals.
 *  Dot tooltips: `Legacy: {kcal}kcal` / `Legacy: {sodium_mg}mg`
 */

import { fmt } from "./util.js";
import { legacyDiffBandClass } from "./legacy-diff-bands.js";

const MEAL_MARGIN_RE =
  /\*\*~?([\d,.]+)\s*\/\s*~?([\d,.]+)\s*(?:→|->)\s*\+?\s*[\d.]+\s*%\s*margin\s*=\s*~?([\d,.]+)/i;

function parseFloatLoose(s) {
  return Number(String(s || "").replace(/,/g, "")) || 0;
}

/** Last **~mg / ~kcal → +margin = ~kcal** (or trailing pair) in a meal cell raw string. */
export function parseSlotLegacyFromRaw(raw) {
  const text = String(raw || "").trim();
  if (!text) return { sodium_mg: 0, kcal: 0 };
  let m;
  let lastMargin = null;
  const marginRe = new RegExp(MEAL_MARGIN_RE.source, "gi");
  while ((m = marginRe.exec(text)) !== null) lastMargin = m;
  if (lastMargin) {
    return {
      sodium_mg: parseFloatLoose(lastMargin[1]),
      kcal: parseFloatLoose(lastMargin[3]),
    };
  }
  const pairs = [...text.matchAll(/~?([\d,.]+)\s*\/\s*~?([\d,.]+)/g)];
  if (!pairs.length) return { sodium_mg: 0, kcal: 0 };
  const last = pairs[pairs.length - 1];
  return { sodium_mg: parseFloatLoose(last[1]), kcal: parseFloatLoose(last[2]) };
}

export function slotLegacyTooltip(field, legVal) {
  const v = Number(legVal) || 0;
  if (v <= 0) return "";
  return field === "sodium" ? `Legacy: ${fmt(v, 0)}mg` : `Legacy: ${fmt(v, 0)}kcal`;
}

/**
 * @param {{ legacy_totals?: { kcal?: number, sodium_mg?: number } } | null | undefined} d
 * @param {{ kcal?: number, sodium_mg?: number } | null | undefined} totals
 * @returns {{ kcal: boolean, sodium: boolean, hasLegacy: boolean }}
 */
export function legacyTotalMismatch(d, totals) {
  const leg = d?.legacy_totals;
  if (!leg) return { kcal: false, sodium: false, hasLegacy: false };
  const legK = Number(leg.kcal) || 0;
  const legNa = Number(leg.sodium_mg) || 0;
  if (legK <= 0 && legNa <= 0) return { kcal: false, sodium: false, hasLegacy: false };

  const kcal = Number(totals?.kcal) || 0;
  const sod = Number(totals?.sodium_mg) || 0;
  return {
    hasLegacy: true,
    kcal: legK > 0 && Math.round(kcal) !== Math.round(legK),
    sodium: legNa > 0 && Math.round(sod) !== Math.round(legNa),
  };
}
