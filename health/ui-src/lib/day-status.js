import { SLOTS } from "./constants.js";
import { esc } from "./strings.js";

function mealRawSkipped(raw) {
  const s = (raw || "").trim();
  if (!s || s === "-" || s === "\u2014") return true;
  if (/^[-\u2014\-]+$/.test(s)) return true;
  if (/^[-\u2014]\s*\(no /i.test(s)) return true;
  return false;
}

export function mealHasContent(m) {
  if (!m) return false;
  if ((m.items || []).some((it) => (it.name || "").trim() || (it.parts || []).length)) return true;
  const raw = (m.raw || "").trim();
  return !!raw && !mealRawSkipped(raw);
}

export function mealItemsNeedReparse(items, raw) {
  raw = (raw || "").trim();
  if (!raw || mealRawSkipped(raw)) return false;
  if (!items?.length) return true;
  if (items.length === 1) {
    const it = items[0];
    const parts = it.parts || [];
    if (!parts.length && ((it.name || "").includes(" + ") || (it.name || "").includes("\u2192"))) return true;
    if (!parts.length && !(Number(it.grams) > 0) && raw.includes("+")) return true;
  }
  return false;
}

function gapsForDate(gapItems, date) {
  return (gapItems || []).filter((g) => g.date === date);
}

function gapLineLabel(g) {
  const name = (g.name || "").trim() || "(unnamed)";
  const slot = (g.slot || "").trim() || "?";
  const grams = Number(g.grams);
  const gPart = grams > 0 ? ` ${grams}g` : "";
  const base = `${slot}: ${name}${gPart}`;
  const note = (g.note || "").trim();
  return note ? `${base}: ${note}` : base;
}

/**
 * @returns {{ level: "ok"|"warn"|"bad", title: string, lines: string[] }}
 */
export function dayCardStatus(day, { gapItems = [], weights = [] } = {}) {
  const reasons = [];
  const severe = [];

  const hasWeight = (weights || []).some((w) => w.date === day.date && Number(w.kg) > 0);
  const anyMeal = SLOTS.some((slot) => mealHasContent(day.meals?.[slot]));
  const optional = [];

  if (!hasWeight) optional.push("No weigh-in");
  const bp = (day.bp || "").trim();
  if (!bp) optional.push("No BP");
  if (!anyMeal) reasons.push("No meals logged");

  for (const slot of SLOTS) {
    const m = day.meals?.[slot];
    if (!mealHasContent(m)) continue;
    if (mealItemsNeedReparse(m.items, m.raw)) {
      severe.push(`${slot}: meal text not split into lines`);
    }
    const items = m.items || [];
    const blobLine = items.length === 1 && !(items[0].parts || []).length;
    if (blobLine && (m.raw || "").includes("+") && Number(m.kcal) <= 0) {
      severe.push(`${slot}: totals missing for multi-item meal`);
    }
  }

  const gaps = gapsForDate(gapItems, day.date);
  if (gaps.length) {
    reasons.push(`Missing catalog / per 100 g (${gaps.length}):`);
    for (const g of gaps) reasons.push(`  \u2022 ${gapLineLabel(g)}`);
  }

  const totalsK = Number(day.totals?.kcal) || 0;
  if (anyMeal && totalsK <= 0) reasons.push("Day kcal total is zero");

  const lines = severe.length > 0
    ? severe.concat(reasons)
    : reasons.length > 0
      ? reasons
      : optional.length > 0
        ? []
        : ["All set"];
  const title = [...optional, ...lines].join("\n");
  if (severe.length) {
    return { level: "bad", title, lines, optionalLines: optional };
  }
  if (reasons.length) {
    return { level: "warn", title, lines, optionalLines: optional };
  }
  return { level: "ok", title, lines, optionalLines: optional };
}

function statusTipMarkup(st) {
  const optional = st?.optionalLines || [];
  const lines = st?.lines?.length ? st.lines : (optional.length ? [] : ["All set"]);
  const tipText = [...optional, ...lines].join("\n");
  const html = [
    ...optional.map((t) => `<span class="day-status-tip-optional">${esc(t)}</span>`),
    ...lines.map((ln) => esc(ln)),
  ].join("<br>");
  const tip = html
    ? `<span class="day-card-status-tip" role="tooltip">${html}</span>`
    : "";
  return { tipText, tip };
}

/** Dot on Days list cards (same popup as Today when there is something to show). */
export function dayCardListDotMarkup(st) {
  const { tipText, tip } = statusTipMarkup(st);
  const level = st?.level || "ok";
  const dotClass = level === "warn" ? "day-card-status--warn" : level === "bad" ? "day-card-status--bad" : "day-card-status--ok";
  const label = level === "ok" ? "Day status" : level === "warn" ? "Needs attention" : "Needs fix";
  const hover = tip ? " day-card-status--hoverable" : "";
  return `<span class="day-card-status day-card-status--list ${dotClass}${hover}" title="${esc(tipText)}" aria-label="${esc(label)}: ${esc(tipText)}">${tip}</span>`;
}

/** Rich hover tooltip on Today header (not on list cards). */
export function dayStatusDotMarkup(st) {
  const { tipText, tip } = statusTipMarkup(st);
  if (!st || st.level === "ok") {
    return `<span class="day-title-status day-card-status day-card-status--ok day-card-status--hoverable" role="status" tabindex="0" title="${esc(tipText)}" aria-label="Day status: ${esc(tipText)}">${tip}</span>`;
  }
  const dotClass = st.level === "warn" ? "day-card-status--warn" : "day-card-status--bad";
  const dotLabel = st.level === "warn" ? "Needs attention" : "Needs fix";
  return `<span class="day-title-status day-card-status ${dotClass} day-card-status--hoverable" role="status" tabindex="0" title="${esc(tipText)}" aria-label="${esc(dotLabel)}: ${esc(st.title)}">${tip}</span>`;
}
