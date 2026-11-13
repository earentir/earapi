/** Markdown day-column totals (hover only) vs catalog-computed day totals.
 *  Dot tooltips: `Legacy: {kcal}kcal` / `Legacy: {sodium_mg}mg`
 */

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
