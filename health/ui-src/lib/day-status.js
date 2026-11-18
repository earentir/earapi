import { SLOTS } from "./constants.js";
import { esc } from "./strings.js";

const DOT_BADGE = {
  ok: "badge-success",
  warn: "badge-warning",
  bad: "badge-error",
};

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

function gapsForDate(gapItems, date, gapIndex) {
  if (gapIndex?.get) return gapIndex.get(date) || [];
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
export function dayCardStatus(day, { gapItems = [], gapIndex = null, weights = [] } = {}) {
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

  const gaps = gapsForDate(gapItems, day.date, gapIndex);
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

function statusTipText(st) {
  const optional = st?.optionalLines || [];
  const lines = st?.lines?.length ? st.lines : (optional.length ? [] : ["All set"]);
  return [...optional, ...lines].join("\n");
}

function dotMarkup(st, { list = false } = {}) {
  const level = st?.level || "ok";
  const tipText = statusTipText(st);
  const badge = DOT_BADGE[level] || DOT_BADGE.ok;
  const label = level === "ok" ? "Day status" : level === "warn" ? "Needs attention" : "Needs fix";
  const pos = list ? "tooltip-left" : "tooltip-bottom";
  const extra = list
    ? "absolute right-2 bottom-2"
    : "relative inline-flex shrink-0";
  return `<span class="tooltip ${pos} ${extra}" data-tip="${esc(tipText)}">
    <span class="badge badge-xs ${badge} border-2 border-base-100" role="status" tabindex="0" title="${esc(tipText)}" aria-label="${esc(label)}: ${esc(tipText)}"></span>
  </span>`;
}

/** Dot on Days list cards. */
export function dayCardListDotMarkup(st) {
  return dotMarkup(st, { list: true });
}

/** Dot on Today header. */
export function dayStatusDotMarkup(st) {
  return dotMarkup(st || { level: "ok", lines: ["All set"] }, { list: false });
}