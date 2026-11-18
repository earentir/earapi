import { pad2, shiftISODate } from "./util.js";

export const DAYS_RANGE_OPTIONS = [
  { id: "7", label: "Last 7 days" },
  { id: "14", label: "Last 14 days" },
  { id: "28", label: "Last 28 days" },
  { id: "month", label: "This month" },
  { id: "quarter", label: "This quarter" },
  { id: "all", label: "All days" },
];

/** Inclusive { from, to } ISO dates for GET /days (empty = no bound). */
export function daysRangeBounds(rangeId, todayISO) {
  const to = String(todayISO || "").trim();
  if (!rangeId || rangeId === "all" || !to) return { from: "", to: "" };
  if (rangeId === "7") return { from: shiftISODate(to, -6), to };
  if (rangeId === "14") return { from: shiftISODate(to, -13), to };
  if (rangeId === "28") return { from: shiftISODate(to, -27), to };
  if (rangeId === "month") {
    const [y, m] = to.split("-");
    return { from: `${y}-${m}-01`, to };
  }
  if (rangeId === "quarter") {
    const parts = to.split("-").map(Number);
    const y = parts[0];
    const m = parts[1];
    const qStart = Math.floor((m - 1) / 3) * 3 + 1;
    return { from: `${y}-${pad2(qStart)}-01`, to };
  }
  return { from: "", to: "" };
}

export function daysRangeQueryString(rangeId, todayISO) {
  const { from, to } = daysRangeBounds(rangeId, todayISO);
  if (!from && !to) return "";
  const p = new URLSearchParams();
  if (from) p.set("from", from);
  if (to) p.set("to", to);
  const s = p.toString();
  return s ? `&${s}` : "";
}

/** Keep rows whose ISO `date` falls in the range (inclusive). */
export function recordsInRange(records, rangeId, todayISO) {
  const { from, to } = daysRangeBounds(rangeId, todayISO);
  if (!from && !to) return records || [];
  return (records || []).filter((row) => {
    const d = String(row?.date || "").trim();
    if (!d) return false;
    if (from && d < from) return false;
    if (to && d > to) return false;
    return true;
  });
}

export function periodRangeSelectHtml(selectId, curValue, esc, { label = "Period" } = {}) {
  const cur = curValue || "28";
  const opts = DAYS_RANGE_OPTIONS.map(
    ({ id, label: optLabel }) =>
      `<option value="${esc(id)}"${id === cur ? " selected" : ""}>${esc(optLabel)}</option>`,
  ).join("");
  const cap = label
    ? `<span class="label-text text-xs uppercase opacity-60">${esc(label)}</span>`
    : "";
  const aria = label ? "" : ` aria-label="${esc("Period")}"`;
  return `<label class="form-control min-w-0 flex-1">${cap}<select id="${esc(selectId)}" class="select select-bordered select-sm w-full"${aria}>${opts}</select></label>`;
}