import { AUTH_KEY, EMPTY_CELL, SLOTS } from "./constants.js";
import { NAV_TABS, applyRouteToState, pathForTab, pushAppRoute, resolveTodayDate } from "./route.js";
import { dayCardStatus, dayCardListDotMarkup, dayStatusDotMarkup } from "./day-status.js";
import {
  nutritionDay,
  dayDisplay,
  mealBandClass,
  computedMealItem,
  dayTotalsView,
  itemGapFor,
  scheduleDayNutritionCompute,
  refreshDayNutrition,
  refreshStockNutrition,
  clearStockNutrition,
  legacyDayForDiff,
  mealSlotRaw,
} from "./nutrition-client.js";
import { parseSlotLegacyFromRaw, slotLegacyTooltip } from "./legacy-totals.js";
import { legacyDiffBandClass } from "./legacy-diff-bands.js";
import { themeSelectHtml, saveTheme } from "./theme.js";
import { state, emptyItem, emptySlot, emptyDraft, emptyStockSet, emptyStockForm } from "./state.js";
import {
  fmt, el, pad2, isoDate, parseISODate, shiftISODate, parseTime24, dateField,
  bindIsoInputs, confirmModal, formModal, customModal, weekdayNameForISODate, daisyToggle, isoDateInputHtml,
} from "./util.js";
import { sodiumMgFromSaltG, saltGFromSodiumMg } from "./eu-salt.js";
import { api, loadAuth, saveAuth, toast } from "./api.js";
import { esc } from "./strings.js";
import {
  itemHasParts,
  setsInSlot,
  setNameSuggestions,
  pruneBlankSetParts,
  setPartStats,
  recomputeSetGrams,
  absorbIntoSet,
  ejectPartFromSet,
  cloneDraftItem,
} from "./meal-items.js";
import {
  bestFoodMatch, bestFoodByAlias, foodByAlias, foodMatchCandidates,
  mealLineFoodSuggestions, foodPickCandidates, cleanFoodDisplayName,
} from "./fuzzy.js";
import { matchStockLotByName } from "./stock-match.js";
import {
  DEFAULT_KCAL_BUDGET, mergedKcalBudget, dayKcalBarParts, dayIntakeBand, dayBandCounts,
} from "./kcal-budget.js";
import { dataBackupBarHtml, dataBackupButtonsHtml, bindDataBackupHandlers } from "./data-backup.js";
import { weightChartSectionHtml, mountWeightChart } from "./weight-chart.js";
import { daysKcalChartSectionHtml, mountDaysKcalChart } from "./days-kcal-chart.js";
import { daysRangeQueryString, periodRangeSelectHtml, recordsInRange } from "./days-range.js";
import { energyFromInputs, estimatedMassDeltaGrams, kgOnOrBefore } from "./energy.js";
const MAIN_WRAP = "max-w-[980px] mx-auto px-4 pt-[10px] pb-12";
const NAV_SHELL = "sticky top-0 z-20 border-b border-base-300 bg-base-200";
const NAVBAR = "navbar max-w-[980px] mx-auto w-full min-h-12 bg-base-200 px-2";
const TABS = "tabs tabs-boxed tabs-sm w-full overflow-x-auto flex-nowrap";
const ITEM_ROW =
  "grid grid-cols-[2.25rem_minmax(0,1fr)_2.5rem_4rem_3.75rem_4rem_2.25rem] gap-1.5 mb-1.5 items-center max-sm:grid-cols-[2.25rem_minmax(0,1fr)_2.25rem_3rem_2.75rem_3rem_2.25rem] max-sm:gap-1";
const ITEM_ROW_HEAD = `${ITEM_ROW} text-xs uppercase opacity-60 mb-1`;
const MEAL_PARTS = "parts min-w-0";
const MEAL_INP_H = "h-[2.1rem] min-h-[2.1rem]";
const MEAL_ICON_SPACER = "meal-row-icon-spacer";
const MEAL_ROW_ICON = "btn btn-square btn-sm btn-outline meal-row-icon";
/** ⊞ renders tiny in most fonts at 1.5rem — fixed-size grid SVG */
const MEAL_ICON_MAKE_SET = `<svg class="meal-row-icon-graphic" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M3 3h8v8H3V3zm10 0h8v8h-8V3zM3 13h8v8H3v-8zm10 0h8v8h-8v-8z"/></svg>`;
const CARD_SECTION = "card card-bordered bg-base-100 shadow-sm border-base-300";
const GRID_4 = "grid grid-cols-2 sm:grid-cols-4 gap-2 items-end";
const SUGGEST_BTN = "btn btn-ghost btn-sm justify-start w-full";
const MEAL_SUGGEST_LIST =
  "meal-suggest-list menu menu-sm absolute z-30 left-0 right-0 mt-1 rounded-box border border-base-300 bg-base-200 shadow-lg p-1";
const FF_INP = "input input-bordered input-sm w-full";
const FF_TEXTAREA = "textarea textarea-bordered textarea-sm w-full min-h-16";
const FF_LABEL = "label-text text-xs uppercase opacity-60";
const FF_FOOD_FORM = "flex flex-col gap-2 w-full min-w-0";
/** ~60% of prior fixed widths (7.5rem grams, ~13rem variant column, 7rem EU label fields). */
const FF_FOOD_G = "form-control min-w-0 w-full sm:w-[4.5rem] sm:shrink-0";
const FF_FOOD_ED = "form-control min-w-0 w-full sm:w-[5.25rem] sm:shrink-0";
const FF_VARIANT_MACRO = "form-control min-w-0 w-full";
/** Four 7.75rem macro columns + gaps; matches food edit modal width. */
const FF_MODAL_BOX = "modal-box w-11/12 max-w-[36rem]";
const GRID_FF_VARIANT =
  "grid grid-cols-2 gap-2 items-end w-fit max-w-full sm:grid-cols-[repeat(4,7.75rem)]";
const MEAL_INP = "input input-bordered input-xs w-full";
const MEAL_INP_NUM = `${MEAL_INP} text-right font-mono tabular-nums`;
const INTAKE_SWATCH = {
  green: "badge-success",
  yellow: "badge-warning",
  orange: "badge-warning",
  red: "badge-error",
  none: "badge-ghost",
};

function intakeSwatchHtml(band) {
  const bg = INTAKE_SWATCH[band] || INTAKE_SWATCH.none;
  return `<span class="badge badge-xs ${bg} border-0 h-2 w-2 min-h-0 p-0 shrink-0" aria-hidden="true"></span>`;
}

const SETTINGS_CARD = "card card-bordered bg-base-100 shadow-sm mt-4";
const SETTINGS_INP = "input input-bordered input-sm w-full";
const SETTINGS_SEL = "select select-bordered select-sm w-full";

function defaultSetName(slot) {
  const s = (slot || "").trim().toLowerCase();
  if (!SLOTS.includes(s)) return "";
  return s.charAt(0).toUpperCase() + s.slice(1);
}

function loginView() {
  const box = el(`<div class="card bg-base-200 shadow-xl border border-base-300 max-w-sm mx-auto mt-24">
    <div class="card-body gap-3">
    <h1 class="card-title">Health tracker</h1>
    <p class="text-base-content/60 text-sm">Sign in with the same username and token as the API (<span class="font-mono">X-User</span> / <span class="font-mono">X-API-Key</span>).</p>
    <label class="form-control w-full"><span class="label-text text-xs uppercase opacity-60">Username</span><input id="login-user" class="input input-bordered input-sm w-full" autocomplete="username"></label>
    <label class="form-control w-full"><span class="label-text text-xs uppercase opacity-60">Token</span><input id="login-token" class="input input-bordered input-sm w-full" type="password" autocomplete="current-password"></label>
    <button class="btn btn-primary btn-sm w-full" id="login-go">Open tracker</button>
    <p id="login-err" class="text-error text-sm"></p>
    </div>
  </div>`);
  const userEl = box.querySelector("#login-user");
  const tokenEl = box.querySelector("#login-token");
  const errEl = box.querySelector("#login-err");
  const submitLogin = async () => {
    state.user = userEl.value.trim();
    state.token = tokenEl.value.trim();
    errEl.textContent = "";
    try {
      saveAuth();
      await boot();
    } catch (e) {
      state.user = "";
      state.token = "";
      errEl.textContent = e.message || "login failed";
    }
  };
  box.querySelector("#login-go").addEventListener("click", () => { submitLogin(); });
  userEl.addEventListener("keydown", (e) => {
    if (e.key !== "Enter") return;
    e.preventDefault();
    tokenEl.focus();
    const n = tokenEl.value.length;
    tokenEl.setSelectionRange(n, n);
  });
  tokenEl.addEventListener("keydown", (e) => {
    if (e.key !== "Enter") return;
    e.preventDefault();
    submitLogin();
  });
  return box;
}

export function render() {
  const root = document.getElementById("app");
  if (!state.user || !state.token) {
    root.replaceChildren(loginView());
    return;
  }
  const page = document.createElement("div");
  page.append(topbar(), content(), state.toast
    ? el(`<div class="toast toast-end toast-bottom z-50"><div class="alert alert-info shadow-lg"><span>${esc(state.toast)}</span></div></div>`)
    : "");
  root.replaceChildren(page);
  bind();
}

function topbar() {
  const tabs = NAV_TABS.map((t) => {
    const href = t.id === "today" ? pathForTab("today") : pathForTab(t.id);
    const active = state.tab === t.id ? " tab-active" : "";
    return `<a role="tab" class="tab${active}" href="${esc(href)}" data-nav-tab="${t.id}">${esc(t.label)}</a>`;
  }).join("");
  return el(`<header class="${NAV_SHELL}">
    <div class="${NAVBAR}" aria-label="Site">
      <div class="navbar-start gap-2 min-w-0">
        <a class="link link-primary text-lg font-bold no-underline" href="${esc(pathForTab("today"))}" data-nav-brand>health</a>
        <span class="text-sm opacity-70 hidden sm:inline truncate">${esc(state.user)}</span>
      </div>
      <div class="navbar-end gap-2 shrink-0">
        ${themeSelectHtml(document.documentElement.dataset.theme || "dark")}
        <button type="button" class="btn btn-ghost btn-sm" id="logout">Log out</button>
      </div>
    </div>
    <div class="max-w-[980px] mx-auto w-full px-2 pb-2">
      <div role="tablist" class="${TABS}">${tabs}</div>
    </div>
  </header>`);
}

function content() {
  const wrap = el(`<main class="${MAIN_WRAP}"></main>`);
  if (state.tab === "today") wrap.append(todayView());
  if (state.tab === "days") wrap.append(daysView());
  if (state.tab === "foods") wrap.append(foodsView());
  if (state.tab === "stock") wrap.append(stockView());
  if (state.tab === "weight") wrap.append(weightView());
  if (state.tab === "settings") wrap.append(settingsView());
  return wrap;
}

function slotMealTotals(slot) {
  const m = nutritionDay()?.meals?.[slot];
  if (!m) return null;
  const kcal = Number(m.kcal) || 0;
  const sodium_mg = Number(m.sodium_mg) || 0;
  if (kcal <= 0 && sodium_mg <= 0) return null;
  return { kcal, sodium_mg };
}

function computedDayTotalsFromDraft() {
  return dayTotalsView();
}

function dayLegacyTotalsForDots() {
  const d = legacyDayForDiff();
  const leg = d?.legacy_totals;
  const lk = Number(leg?.kcal) || 0;
  const ln = Number(leg?.sodium_mg) || 0;
  if (lk <= 0 && ln <= 0) return { kcal: 0, sodium_mg: 0 };
  const tot = d?.totals || dayTotalsView();
  const tk = Number(tot?.kcal) || 0;
  const tn = Number(tot?.sodium_mg) || 0;
  // Catalog saves used to copy totals into legacy; that is not a markdown day column.
  if (Math.round(lk) === Math.round(tk) && Math.round(ln) === Math.round(tn)) {
    return { kcal: 0, sodium_mg: 0 };
  }
  return { kcal: lk, sodium_mg: ln };
}

function dayLegacyComparePair(field) {
  const isNa = field === "sodium";
  const leg = dayLegacyTotalsForDots();
  const totals = dayTotalsView();
  const legVal = isNa ? leg.sodium_mg : leg.kcal;
  const curVal = isNa ? Number(totals.sodium_mg) : Number(totals.kcal);
  return { partial: false, leg: legVal, cur: curVal };
}

function legacyDotTooltip(_d, field) {
  const { leg } = dayLegacyComparePair(field);
  const v = Number(leg) || 0;
  if (v <= 0) return "";
  return field === "sodium" ? `Legacy: ${fmt(v, 0)}mg` : `Legacy: ${fmt(v, 0)}kcal`;
}

function legacyDiffDotMarkup(tip, band) {
  if (!tip) return "";
  const b = band && band !== "muted" ? band : "green";
  const dotBg = { green: "badge-success", yellow: "badge-warning", orange: "badge-warning", red: "badge-error" }[b] || "badge-success";
  return `<span class="tooltip tooltip-bottom inline-flex align-middle" data-tip="${esc(tip)}" data-day-legacy-hover>
    <span class="badge badge-xs ${dotBg} border-2 border-base-100 h-2 w-2 min-h-0 p-0" tabindex="0" aria-label="${esc(tip)}"></span>
  </span>`;
}

function legacyDiffBandForField(_d, field) {
  const { leg, cur, partial } = dayLegacyComparePair(field);
  const legVal = Number(leg) || 0;
  if (legVal <= 0) return "muted";
  if (!partial) {
    const disp = dayDisplay(_d);
    const isNa = field === "sodium";
    const serverBand = isNa ? disp.sodium_diff_band : disp.kcal_diff_band;
    if (serverBand && serverBand !== "muted") return serverBand;
  }
  return legacyDiffBandClass(Number(cur) - legVal, state.kcalBudget);
}

/** Dot after kcal or mg — color and hover for that nutrient only (day legacy column). */
function legacyDiffDot(d, field) {
  const { leg } = dayLegacyComparePair(field);
  const legVal = Number(leg) || 0;
  if (legVal <= 0) return "";
  const tip = legacyDotTooltip(d, field);
  if (!tip) return "";
  const band = legacyDiffBandForField(d, field);
  return legacyDiffDotMarkup(tip, band);
}

/** Dot for lunch/dinner/snack row vs **~mg / ~kcal** in that slot's raw markdown. */
function slotLegacyDiffDot(slot, field) {
  const leg = parseSlotLegacyFromRaw(mealSlotRaw(slot));
  const isNa = field === "sodium";
  const legVal = isNa ? leg.sodium_mg : leg.kcal;
  if (legVal <= 0) return "";
  const m = nutritionDay()?.meals?.[slot];
  const cur = isNa ? Number(m?.sodium_mg) || 0 : Number(m?.kcal) || 0;
  const tip = slotLegacyTooltip(field, legVal);
  const band = legacyDiffBandClass(cur - legVal, state.kcalBudget);
  return legacyDiffDotMarkup(tip, band);
}

function dayMassDeltaMarkup(d, kcal) {
  const b = mergedKcalBudget(state.kcalBudget);
  const kg = kgOnOrBefore(d?.date || state.date, state.weights)
    || Number(state.overview?.latest_weight?.kg)
    || 0;
  const est = energyFromInputs({
    kg,
    heightCm: b.height_cm,
    ageYears: b.age_years,
    sex: b.sex,
    sedentary_discount_percent: b.sedentary_discount_percent,
  });
  if (!est) return "";
  const grams = estimatedMassDeltaGrams(kcal, est.sedentary);
  if (grams == null) return "";
  const signed = grams > 0 ? `+${grams}` : String(grams);
  const kind = grams > 0 ? "gain" : grams < 0 ? "loss" : "flat";
  const label = grams > 0 ? "estimated gain" : grams < 0 ? "estimated loss" : "estimated even";
  const tone = kind === "gain" ? "text-warning" : kind === "loss" ? "text-success" : "text-base-content/60";
  return ` <span class="font-mono tabular-nums text-sm ${tone}" title="${esc(label)} vs sedentary ${fmt(est.sedentary, 0)} kcal">(${esc(signed)} g)</span>`;
}

function dayHeaderTotalMarkup(d, totals) {
  const kcal = Number(totals?.kcal) || 0;
  const sod = Number(totals?.sodium_mg) || 0;
  const intakeBand = mealBandClass(dayIntakeBand(d, state.kcalBudget));
  const naIntakeBand = mealBandClass(dayDisplay(d).sodium_intake_band);
  const kDot = legacyDiffDot(d, "kcal");
  const naDot = legacyDiffDot(d, "sodium");
  const mass = dayMassDeltaMarkup(d, kcal);
  return `<span class="font-mono tabular-nums"><span class="${intakeBand}">${fmt(kcal, 0)}</span><span class="text-base-content/60"> kcal</span>${mass}${kDot}, <span class="${naIntakeBand}">${fmt(sod, 0)}</span><span class="text-base-content/60"> mg Na</span>${naDot}</span>`;
}

function dayPanelEnergyMarkup(totals, d) {
  const diffDay = legacyDayForDiff();
  const disp = dayDisplay(diffDay);
  const kcal = Number(totals?.kcal) || 0;
  const sod = Number(totals?.sodium_mg) || 0;
  const intakeBand = mealBandClass(disp.intake_bar_band);
  const naIntakeBand = mealBandClass(disp.sodium_intake_band);
  const kDot = legacyDiffDot(diffDay, "kcal");
  const naDot = legacyDiffDot(diffDay, "sodium");
  const mass = dayMassDeltaMarkup(d || { date: state.date }, kcal);
  return {
    kcal: `<div class="stat-value font-mono tabular-nums text-xl"><span class="${intakeBand}">${fmt(kcal, 0)}</span> <span class="text-base-content/60 text-sm">kcal</span>${mass}${kDot}</div>`,
    sodium: `<div class="stat-value font-mono tabular-nums text-xl"><span class="${naIntakeBand}">${fmt(sod, 0)}</span> <span class="text-base-content/60 text-sm">mg</span>${naDot}</div>`,
  };
}

function renderDayKcalBar(_kcal, d) {
  const band = dayIntakeBand(d, state.kcalBudget);
  const parts = dayKcalBarParts(band);
  const tip = parts.title ? ` title="${esc(parts.title)}"` : "";
  const borderClass = parts.borderClass ? ` ${parts.borderClass}` : "";
  return { cardClass: borderClass, tip };
}

function daySummaryPanel() {
  const totals = computedDayTotalsFromDraft();
  const d = state.mealDraft;
  const daySt = todayDayCardStatus();
  const weight = (state.weights || []).find((w) => w.date === state.date);
  const slotRows = SLOTS.map((slot) => {
    const label = slot.charAt(0).toUpperCase() + slot.slice(1);
    const t = slotMealTotals(slot);
    const metricsHtml = t
      ? `<div class="text-sm font-mono tabular-nums">
          <span class="font-semibold">${fmt(t.kcal, 0)}</span><span class="text-base-content/60"> kcal</span>${slotLegacyDiffDot(slot, "kcal")}
          <span class="mx-1 text-base-content/30">·</span>
          <span class="font-semibold">${fmt(t.sodium_mg, 0)}</span><span class="text-base-content/60"> mg</span>${slotLegacyDiffDot(slot, "sodium")}
        </div>`
      : `<span class="font-mono tabular-nums text-base-content/60">${EMPTY_CELL}</span>`;
    return `<div class="stat place-items-start p-2">
      <div class="stat-title text-xs uppercase opacity-60">${esc(label)}</div>
      <div class="stat-value text-base p-0">${metricsHtml}</div>
    </div>`;
  }).join("");
  const status = [];
  if ((d.bp || "").trim()) status.push(`BP ${(d.bp || "").trim()}`);
  if ((d.walking || "").trim()) status.push((d.walking || "").trim());
  const statusRow = status.length
    ? `<div class="alert alert-info py-2 text-sm"><span>${esc(status.join(", "))}</span></div>`
    : "";
  const weightLine = weight
    ? `<p class="text-sm text-base-content/70 m-0">Weight <span class="font-mono tabular-nums font-semibold">${fmt(weight.kg, 1)}</span> kg</p>`
    : "";
  const panelTotals = dayPanelEnergyMarkup(totals, nutritionDay());
  const headEnergySodium = `<div class="stats stats-vertical sm:stats-horizontal shadow bg-base-200/80 w-full max-w-full sm:w-auto shrink-0" aria-label="Day totals">
      <div class="stat p-3">
        <div class="stat-title">Energy</div>
        ${panelTotals.kcal}
      </div>
      <div class="stat p-3">
        <div class="stat-title">Sodium</div>
        ${panelTotals.sodium}
      </div>
    </div>`;
  const kbar = renderDayKcalBar(totals.kcal, nutritionDay());
  return `<section class="${CARD_SECTION}${kbar.cardClass}" data-day-summary ${kbar.tip ? `title="${esc(kbar.tip)}" ` : ""}>
    <div class="card-body gap-3 p-4">
      <div class="flex flex-col gap-2 lg:flex-row lg:items-start lg:justify-between">
        <div class="min-w-0">
          <h2 class="card-title text-lg m-0 flex-wrap gap-2">${dayStatusDotMarkup(daySt)}<span class="font-mono tabular-nums">${esc(state.date)}</span> ${esc(weekdayNameForISODate(state.date))}</h2>
          ${weightLine}
        </div>
        ${headEnergySodium}
      </div>
      ${statusRow}
      <div class="stats stats-vertical sm:stats-horizontal w-full max-w-full bg-base-200/40 rounded-box">${slotRows}</div>
      <div class="card-actions justify-start flex-wrap gap-2 items-center">
        ${dateField("day-date", state.date, { label: false })}
        ${dataBackupButtonsHtml("days")}
        <button type="button" class="btn btn-error btn-sm" id="delete-day">Delete day</button>
      </div>
    </div>
  </section>`;
}

function todayView() {
  const d = state.mealDraft;
  const box = el(`<div class="flex flex-col gap-4">
    ${daySummaryPanel()}
    <div class="flex flex-col gap-4">
      ${SLOTS.map((slot) => slotCard(slot, d[slot])).join("")}
      <section class="card card-bordered bg-base-100 shadow-sm">
        <div class="card-body gap-2 p-4">
        <h2 class="card-title text-base">Notes</h2>
        <div class="flex flex-wrap gap-2 items-end">
          <label class="form-control min-w-0 flex-1"><span class="${FF_LABEL}">Blood pressure</span><input class="${FF_INP}" id="day-bp" value="${esc(d.bp)}" placeholder="128/82"></label>
          <label class="form-control min-w-0 flex-1"><span class="${FF_LABEL}">Walking</span><input class="${FF_INP}" id="day-walking" value="${esc(d.walking)}" placeholder="park 40min"></label>
        </div>
        </div>
      </section>
    </div>
  </div>`);
  return box;
}

function itemSkipRaw(raw) {
  const s = (raw || "").trim();
  return !s || s === "-" || s === "\u2014" || /^[-\u2014]\s*\(no /i.test(s);
}

function mealItemsLookLikeBlob(items) {
  if (!items || !items.length) return true;
  if (items.length > 1) return false;
  const it = items[0];
  if (it.parts && it.parts.length) return false;
  const name = (it.name || "").trim();
  return name.includes(" + ") || name.includes("→") || name.length > 72;
}

function foodForItem(it) {
  if (it?.food_id) {
    const byID = state.foods.find((f) => f.id === it.food_id);
    if (byID) return byID;
    const byIdAlias = foodByAlias(state.foodAliases, state.foods, it.food_id);
    if (byIdAlias) return byIdAlias;
  }
  const byAlias = bestFoodByAlias(state.foodAliases, state.foods, it?.name);
  if (byAlias) return byAlias;
  const fuzzy = bestFoodMatch(state.foods, it?.name);
  if (fuzzy) return fuzzy;
  const q = (it?.name || "").trim().toLowerCase();
  if (!q) return null;
  const base = q.replace(/\s*\(.*$/, "").trim();
  return (
    state.foods.find((f) => f.name.toLowerCase() === q)
    || state.foods.find((f) => f.name.toLowerCase() === base)
    || null
  );
}

function knownFood(name) {
  return foodForItem({ name });
}

function serializeItemFoodID(it) {
  const id = (it?.food_id || "").trim();
  if (id && (state.foods.some((f) => f.id === id) || foodByAlias(state.foodAliases, state.foods, id))) {
    return id;
  }
  const linked = foodForItem(it);
  return linked?.id || id || undefined;
}

function stockLotForItem(it) {
  if (!it) return null;
  const editingId = state.stockForm?.id;
  const lots = (state.stock || []).filter((l) => !editingId || l.id !== editingId);
  if (it.stock_lot_id && it.stock_lot_id !== editingId) {
    const byID = lots.find((l) => l.id === it.stock_lot_id);
    if (byID) return byID;
  }
  return matchStockLotByName(it.name, lots);
}

function linkDraftItemFood(it) {
  if (!it || itemHasParts(it)) return;
  if (it.food_id && state.foods.some((f) => f.id === it.food_id)) return;
  const aliasFood = bestFoodByAlias(state.foodAliases, state.foods, it.name);
  if (aliasFood) {
    it.food_id = aliasFood.id;
    return;
  }
  const strict = foodMatchCandidates(state.foods, it.name);
  if (strict.length >= 2) return;
  if (strict.length === 1) {
    it.food_id = strict[0].id;
    return;
  }
  if (foodPickCandidates(state.foods, it.name).length >= 2) return;
  const m = bestFoodMatch(state.foods, it.name);
  if (m) it.food_id = m.id;
}

function foodItemLinked(it) {
  const fid = (it?.food_id || "").trim();
  if (fid && state.foods.some((f) => f.id === fid)) return true;
  if (fid && foodByAlias(state.foodAliases, state.foods, fid)) return true;
  return !!bestFoodByAlias(state.foodAliases, state.foods, it?.name);
}

function itemMissingCatalogFood(it) {
  const fid = (it?.food_id || "").trim();
  if (!fid) return false;
  if (state.foods.some((f) => f.id === fid)) return false;
  if (foodByAlias(state.foodAliases, state.foods, fid)) return false;
  return true;
}

function showFoodPickUi(it) {
  if (!it || itemHasParts(it)) return false;
  if (foodItemLinked(it)) return false;
  if (stockLotForItem(it)) return false;
  const name = (it?.name || "").trim();
  if (!name) return false;
  return foodPickOptions(it).length > 1;
}

function foodPickOptions(it) {
  const name = (it?.name || "").trim();
  if (!name || itemHasParts(it)) return [];
  if (foodItemLinked(it)) return [];
  const strict = foodPickCandidates(state.foods, name);
  if (strict.length) {
    return strict.map((f) => ({ f, label: (f.name || "").trim() }));
  }
  return mealLineFoodSuggestions(state.foods, state.foodAliases, name, 15);
}

function foodPickOpen(slot, i, partIdx) {
  const p = state.foodPick;
  if (!p) return false;
  return p.slot === slot && p.i === i && (p.p ?? -1) === (partIdx == null ? -1 : partIdx);
}

function foodPickList(slot, i, partIdx, it, options) {
  const open = foodPickOpen(slot, i, partIdx);
  if (!open) return "";
  let opts = options;
  if (!opts.length) {
    const strict = foodPickCandidates(state.foods, (it?.name || "").trim());
    opts = strict.length
      ? strict.map((f) => ({ f, label: (f.name || "").trim() }))
      : mealLineFoodSuggestions(state.foods, state.foodAliases, (it?.name || "").trim(), 15);
  }
  if (!opts.length) {
    return `<div class="text-sm text-base-content/60 p-2" data-food-pick-list role="status">No catalog matches. Add the food under Foods, or set per 100 g in the kcal / Na columns.</div>`;
  }
  return `<ul class="${MEAL_SUGGEST_LIST}" data-food-pick-list role="listbox">${opts.flatMap(({ f, label }) => {
    const rows = foodVariantPickChoices(f);
    if (rows.length <= 1 && !rows[0]?.value) {
      return [`<li><button type="button" class="${SUGGEST_BTN}" data-resolve-food="${esc(f.id)}:${esc(slot)}:${i}${partIdx == null ? "" : `:${partIdx}`}" title="Save alias for this line name">
        ${esc(label)} <span class="text-base-content/60">${esc(rows[0]?.hint || foodCatalogPickHint(f) || "—")}</span>
      </button></li>`];
    }
    return rows.map((ch) => `<li><button type="button" class="${SUGGEST_BTN}" data-resolve-food="${esc(f.id)}:${esc(slot)}:${i}${partIdx == null ? "" : `:${partIdx}`}" data-pick-variant="${esc(ch.value)}" title="Save alias for this line name">
        ${esc(ch.label)} <span class="text-base-content/60">${esc(ch.hint || "—")}</span>
      </button></li>`);
  }).join("")}</ul>`;
}

function refreshStockModalIfOpen() {
  const overlay = document.querySelector("[data-stock-batch-modal]");
  if (overlay) scheduleStockModalFieldRefresh(overlay);
}

async function resolveFoodPick(foodID, slot, idx, part, { cut, prep } = {}) {
  const target = draftTarget(slot, idx, part);
  if (!target) return;
  const alias = (target.name || "").trim();
  if (!alias) {
    toast("Name required for alias");
    return;
  }
  const food = state.foods.find((x) => x.id === foodID);
  applyCatalogFoodToMealItem(target, food, alias, { cut, prep });
  try {
    await api(`/foods/${foodID}/aliases`, { method: "POST", body: JSON.stringify({ alias }) });
    await loadFoods();
    state.foodPick = null;
    toast("Linked to catalog");
    if (slot === "stock") {
      refreshStockModalIfOpen();
      return;
    }
    scheduleSaveDay();
    render();
  } catch (e) {
    toast(e.message || "alias failed");
  }
}

function linkDraftItemStock(it) {
  if (!it || itemHasParts(it)) return;
  const lot = stockLotForItem(it);
  if (lot) it.stock_lot_id = lot.id;
}

function slotHeaderTotals(slot) {
  const t = slotMealTotals(slot);
  if (t && (t.kcal || t.sodium_mg)) {
    return `${fmt(t.kcal, 0)} kcal, ${fmt(t.sodium_mg, 0)} mg`;
  }
  const meal = state.day?.meals?.[slot];
  if (meal && (meal.kcal || meal.sodium_mg)) {
    return `${fmt(meal.kcal, 0)} kcal, ${fmt(meal.sodium_mg, 0)} mg`;
  }
  return "empty";
}

function slotCard(slot, draft) {
  const summary = slotHeaderTotals(slot);
  const rows = (draft.items || []).map((it, i) => itemBlock(slot, i, it)).join("");
  const hasItems = (draft.items || []).some((it) => (it.name || "").trim() || itemHasParts(it));
  return `<section class="card card-bordered bg-base-100 shadow-sm" data-slot="${slot}">
    <div class="card-body gap-2 p-4">
    <div class="flex flex-wrap items-baseline justify-between gap-2">
      <h2 class="card-title capitalize m-0 text-base">${slot}</h2>
      <span class="text-base-content/60 font-mono tabular-nums text-sm">${summary}</span>
    </div>
    ${hasItems ? mealItemsGridHead() : ""}
    <div class="w-full min-w-0">${rows}</div>
    <div class="flex flex-wrap gap-2 items-end">
      <button class="btn btn-outline btn-sm" data-add="${slot}">Add food</button>
    </div>
    </div>
  </section>`;
}

function clearNameSuggest() {
  state.foodPick = null;
  state.suggest = { slot: "", i: -1, p: -1, q: "" };
}

function captureFieldFocus(root) {
  const el = document.activeElement;
  if (!el?.matches?.("[data-field], #day-bp, #day-walking")) return null;
  if (root && !root.contains(el)) return null;
  return {
    slot: el.dataset.slot,
    idx: el.dataset.idx,
    part: el.dataset.part ?? null,
    field: el.dataset.field,
    id: el.id || "",
    start: el.selectionStart,
    end: el.selectionEnd,
  };
}

function restoreFieldFocus(cap, root = document) {
  if (!cap) return;
  let el;
  if (cap.id) {
    el = document.getElementById(cap.id) || root.querySelector(`#${cap.id}`);
  }
  if (!el && cap.field) {
    const partSel = cap.part == null ? ":not([data-part])" : `[data-part="${cap.part}"]`;
    const sel = `[data-field="${cap.field}"][data-slot="${cap.slot}"][data-idx="${cap.idx}"]${partSel}`;
    el = root.querySelector(sel);
  }
  if (!el) return;
  el.focus();
  if (typeof cap.start === "number" && typeof cap.end === "number" && el.setSelectionRange) {
    const len = (el.value || "").length;
    const s = Math.min(cap.start, len);
    const e = Math.min(cap.end, len);
    try {
      el.setSelectionRange(s, e);
    } catch {
      el.setSelectionRange(len, len);
    }
  }
}

/** Full rerender without kicking the user out of meal/draft inputs. */
export function renderPreservingFocus() {
  const overlay = document.querySelector("[data-stock-batch-modal]");
  if (overlay) {
    refreshStockModalIfOpen();
    return;
  }
  const cap = captureFieldFocus();
  render();
  restoreFieldFocus(cap);
}

let mealFieldRenderTimer = null;
function scheduleMealFieldRender() {
  clearTimeout(mealFieldRenderTimer);
  mealFieldRenderTimer = setTimeout(() => renderPreservingFocus(), 80);
}

let stockModalInputTimer = null;
function scheduleStockModalFieldRefresh(overlay) {
  clearTimeout(stockModalInputTimer);
  const capNow = captureFieldFocus(overlay);
  refreshStockModalBody(overlay);
  restoreFieldFocus(capNow, overlay);
  stockModalInputTimer = setTimeout(() => {
    refreshStockNutritionThen(overlay, () => {
      const cap = captureFieldFocus(overlay);
      refreshStockModalBody(overlay);
      restoreFieldFocus(cap, overlay);
    });
  }, 80);
}

let nameSuggestDismissBound = false;
function refreshAfterNameSuggestDismiss() {
  if (document.querySelector("[data-stock-batch-modal]")) refreshStockModalIfOpen();
  else renderPreservingFocus();
}

function ensureNameSuggestDismiss() {
  if (nameSuggestDismissBound) return;
  nameSuggestDismissBound = true;
  document.addEventListener("mousedown", (e) => {
    if (e.target.closest?.("[data-meal-suggest]") || e.target.closest?.("[data-food-pick-list]")) return;
    const hasSuggest = Boolean(state.suggest?.slot);
    if (!state.foodPick && !hasSuggest) return;
    clearNameSuggest();
    refreshAfterNameSuggestDismiss();
  });
  document.addEventListener(
    "keydown",
    (e) => {
      if (e.key !== "Escape") return;
      const hasSuggest = Boolean(state.suggest?.slot);
      if (!state.foodPick && !hasSuggest) return;
      e.preventDefault();
      e.stopPropagation();
      clearNameSuggest();
      refreshAfterNameSuggestDismiss();
    },
    true,
  );
}

function suggestList(slot, i, partIdx, q, { isSetHeader = false } = {}) {
  const open = state.suggest.slot === slot && state.suggest.i === i
    && (state.suggest.p ?? -1) === (partIdx == null ? -1 : partIdx);
  const query = (q || "").trim();
  if (!open || query.length < 1) return "";
  if (isSetHeader) {
    const names = setNameSuggestions(slot, i, query, 10);
    if (!names.length) return "";
    return `<ul class="${MEAL_SUGGEST_LIST}" role="listbox">${names.map((name) =>
      `<li><button type="button" class="${SUGGEST_BTN}" data-pick-set-name="${esc(name)}" data-slot="${esc(slot)}" data-idx="${i}">${esc(name)}</button></li>`
    ).join("")}</ul>`;
  }
  const hits = mealLineFoodSuggestions(state.foods, state.foodAliases, query, 10);
  if (!hits.length) return "";
  const partAttr = partIdx == null ? "" : ` data-part="${partIdx}"`;
  return `<ul class="${MEAL_SUGGEST_LIST}" role="listbox">${hits.flatMap(({ f, label }) => expandFoodPickButtons(f, label, slot, i, partAttr)).join("")}</ul>`;
}

function draftStockLine(idx, partIdx) {
  const set = state.mealDraft?.stock?.items?.[idx];
  if (!set) return null;
  return partIdx == null ? set : set.parts?.[partIdx] || null;
}

function stockLinePreviewTotals(idx, partIdx) {
  if (partIdx != null) {
    return partNutrientPreview(draftStockLine(idx, partIdx));
  }
  const set = draftStockLine(idx, null);
  let kcal = 0;
  let na = 0;
  let any = false;
  for (const p of set?.parts || []) {
    const t = partNutrientPreview(p);
    if (t.kcal != null) {
      kcal += t.kcal;
      any = true;
    }
    if (t.na != null) {
      na += t.na;
      any = true;
    }
  }
  return any ? { kcal, na } : { kcal: null, na: null };
}

function itemKcalCell(slot, idx, partIdx) {
  const it = computedMealItem(slot, idx, partIdx);
  const k = Number(it?.kcal);
  if (Number.isFinite(k) && k > 0) return fmt(k, 0);
  if (slot === "stock") {
    const prev = stockLinePreviewTotals(idx, partIdx);
    if (prev.kcal != null && prev.kcal > 0) return fmt(prev.kcal, 0);
  }
  return EMPTY_CELL;
}

function itemNaCell(slot, idx, partIdx) {
  const it = computedMealItem(slot, idx, partIdx);
  const n = Number(it?.sodium_mg);
  if (Number.isFinite(n) && n > 0) return fmt(n, 0);
  if (slot === "stock") {
    const prev = stockLinePreviewTotals(idx, partIdx);
    if (prev.na != null && prev.na > 0) return fmt(prev.na, 0);
  }
  return EMPTY_CELL;
}

function foodHasPer100Nutrition(f) {
  if (!f) return false;
  const k = f.kcal_per_100g;
  const n = f.sodium_mg_per_100g;
  return k != null && k !== "" && !Number.isNaN(Number(k))
    && n != null && n !== "" && !Number.isNaN(Number(n));
}

function needsPer100gRow(it) {
  if (!(it?.name || "").trim() || itemHasParts(it)) return false;
  const f = foodForItem(it);
  if (f && foodHasPer100Nutrition(f)) return false;
  if (foodPickOptions(it).length >= 1) return false;
  const lot = stockLotForItem(it);
  if (lot && lot.kcal_per_100g != null && lot.sodium_mg_per_100g != null) return false;
  const k = Number(it.kcal);
  const n = Number(it.sodium_mg);
  if ((!Number.isNaN(k) && k > 0) || (!Number.isNaN(n) && n > 0)) return false;
  if (foodItemLinked(it) || it?.food_id) return false;
  return true;
}

/** Highlight from server gap list (POST /nutrition/compute/day). */
function ingredientMissingGrams(it) {
  if (!(it?.name || "").trim()) return false;
  if (itemHasParts(it)) return false;
  const g = numOrUndef(it.grams);
  const c = numOrUndef(it.count);
  return !(g != null && g > 0) && !(c != null && c > 0);
}

function itemNutritionIssue(slot, idx, partIdx) {
  const it = partIdx == null
    ? state.mealDraft?.[slot]?.items?.[idx]
    : state.mealDraft?.[slot]?.items?.[idx]?.parts?.[partIdx];
  const gap = itemGapFor(slot, idx, partIdx);
  if (gap?.note === "missing_food" || itemMissingCatalogFood(it)) {
    return { msg: "Catalog food missing — choose a replacement when deleting foods", missingFood: true };
  }
  if (gap) {
    const msg = (gap.note || "").trim() || "Missing catalog data to compute this line";
    return { msg };
  }
  if (partIdx != null && ingredientMissingGrams(it)) {
    return { msg: "Enter grams (or × count) for this ingredient" };
  }
  return null;
}

function draftNutritionGaps() {
  return (dayDisplay().item_gaps || []).map((g) => ({
    date: g.date || state.date,
    slot: g.slot,
    name: g.name,
    grams: g.grams,
    note: g.note,
  }));
}

function draftDayForStatus() {
  const md = state.mealDraft || emptyDraft();
  const saved = state.day || { meals: {}, totals: {} };
  const meals = {};
  for (const slot of SLOTS) {
    const m = saved.meals?.[slot];
    meals[slot] = {
      items: md[slot]?.items || [],
      raw: m?.raw || "",
      kcal: m?.kcal,
      sodium_mg: m?.sodium_mg,
    };
  }
  return {
    date: state.date,
    meals,
    totals: saved.totals || { kcal: 0, sodium_mg: 0 },
    bp: md.bp,
    walking: md.walking,
  };
}

function todayDayCardStatus() {
  const day = draftDayForStatus();
  for (const slot of SLOTS) {
    const t = slotMealTotals(slot);
    if (t && day.meals?.[slot]) {
      day.meals[slot].kcal = t.kcal;
      day.meals[slot].sodium_mg = t.sodium_mg;
    }
  }
  day.totals = computedDayTotalsFromDraft();
  return dayCardStatus(day, {
    gapItems: draftNutritionGaps(),
    weights: state.weights,
  });
}

function rowIconBtn(attrs, title, glyph, extraClass = "") {
  const warn = extraClass.includes("warn") ? " btn-warning" : "";
  const part = attrs.includes("data-eject-part") ? " meal-part-eject" : "";
  const inner = glyph.includes("<svg") ? glyph : esc(glyph);
  return `<button type="button" class="${MEAL_ROW_ICON}${warn}${part}" title="${esc(title)}" aria-label="${esc(title)}" ${attrs}>${inner}</button>`;
}

function mealItemsGridHead() {
  return `<div class="${ITEM_ROW_HEAD}" aria-hidden="true">
    <span class="${MEAL_ICON_SPACER}" aria-hidden="true"></span>
    <span>Name</span>
    <span class="text-right">×</span>
    <span class="text-right">g</span>
    <span class="text-right">kcal</span>
    <span class="text-right">Na mg</span>
    <span class="${MEAL_ICON_SPACER}" aria-hidden="true"></span>
  </div>`;
}

function nameGramsRow(slot, i, it, partIdx) {
  const isPart = partIdx != null;
  const needPer100g = needsPer100gRow(it);
  const nutritionIssue = itemNutritionIssue(slot, i, partIdx);
  const partAttr = isPart ? ` data-part="${partIdx}"` : "";
  const isSetHeader = !isPart && itemHasParts(it);
  const computedSet = isSetHeader ? computedMealItem(slot, i, null) : null;
  const setPartComplete = (p) => (p?.name || "").trim() && (Number(p.grams) > 0 || Number(p.count) > 0);
  const setPartCount = (it.parts || []).filter(setPartComplete).length;
  const gramsPh = isPart ? "g" : "g";
  const namePh = isPart ? "Ingredient" : (isSetHeader ? (slot === "stock" ? "Batch name" : "Set name") : "Food");
  const countCell = isSetHeader
    ? `<span class="text-right text-base-content/60 text-sm">${EMPTY_CELL}</span>`
    : `<input class="${MEAL_INP_NUM} ${MEAL_INP_H}" data-field="count" data-slot="${esc(slot)}" data-idx="${i}"${partAttr} value="${esc(it.count)}" placeholder="×" inputmode="decimal" title="How many units (e.g. 4 wafers)">`;
  const setGrams = Number(computedSet?.grams) || Number(it.grams) || 0;
  const gramsCell = isSetHeader
    ? `<span class="text-right font-mono tabular-nums text-sm" title="${setPartCount} ingredients, ${fmt(setGrams, 0)} g total">${fmt(setGrams, 0)} <span class="text-base-content/60">g, ${setPartCount} items</span></span>`
    : `<input class="${MEAL_INP_NUM} ${MEAL_INP_H}" data-field="grams" data-slot="${esc(slot)}" data-idx="${i}"${partAttr} value="${esc(it.grams)}" placeholder="${gramsPh}" inputmode="decimal" title="grams total (or leave blank with × count)">`;
  const issueBtn = nutritionIssue
    ? "warn"
    : "";
  const ejectPart = isPart
    ? rowIconBtn(
        `data-eject-part="${esc(slot)}:${i}:${partIdx}"`,
        nutritionIssue ? nutritionIssue.msg : "Remove from set",
        "↩",
        issueBtn,
      )
    : "";
  let lead;
  if (isPart) {
    lead = `<span class="${MEAL_ICON_SPACER}" aria-hidden="true"></span>`;
  } else if (!itemHasParts(it)) {
    const leadTitle = nutritionIssue ? nutritionIssue.msg : "Convert to set";
    lead = rowIconBtn(`data-make-dish="${esc(slot)}:${i}"`, leadTitle, MEAL_ICON_MAKE_SET, issueBtn);
  } else if (isSetHeader) {
    lead = `<span class="${MEAL_ICON_SPACER} flex items-center justify-center" title="Set"><span class="badge badge-sm badge-primary badge-outline">Set</span></span>`;
  } else {
    lead = `<span class="${MEAL_ICON_SPACER}" aria-hidden="true"></span>`;
  }
  const addPartBtn = !isPart && itemHasParts(it)
    ? rowIconBtn(`data-add-part="${esc(slot)}:${i}"`, "New ingredient", "+")
    : "";
  const addToSet = !isPart && !itemHasParts(it) ? moveIntoSetControl(slot, i) : "";
  const pickOpts = foodPickOptions(it);
  const canPick = showFoodPickUi(it);
  const pickBtn = canPick
    ? `<button type="button" class="btn btn-outline btn-xs shrink-0  " data-food-pick="${esc(slot)}:${i}${partIdx == null ? "" : `:${partIdx}`}" title="${esc(pickOpts.length ? `${pickOpts.length} catalog match${pickOpts.length === 1 ? "" : "es"}: pick one` : "Pick food from catalog")}" aria-label="Pick food from catalog">pick</button>`
    : "";
  const stockLot = stockLotForItem(it);
  const stockHint = stockLot && !needPer100g
    ? `<span class="btn btn-outline btn-xs shrink-0  " title="${esc(stockLot.name)}">stock</span>`
    : "";
  const nameTitle = needPer100g ? "Not in Foods or Stock: set per 100 g in the kcal and Na columns" : "";
  const nameInput = `<div class="relative min-w-0 flex-1 w-full" data-meal-suggest>
        <span class="flex items-center gap-1.5 min-w-0 w-full">${stockHint}${pickBtn}<input class="${MEAL_INP} ${MEAL_INP_H} min-w-0 flex-1 w-full" data-field="name" data-slot="${esc(slot)}" data-idx="${i}"${partAttr} value="${esc(it.name)}" placeholder="${namePh}" autocomplete="off"${nameTitle ? ` title="${esc(nameTitle)}"` : ""}>${addToSet}</span>
        ${foodPickList(slot, i, partIdx, it, pickOpts)}
        ${suggestList(slot, i, partIdx, it.name, { isSetHeader })}
      </div>`;
  const kcalCell = needPer100g
    ? `<input class="${MEAL_INP_NUM}" data-field="kcal_per_100g" data-slot="${esc(slot)}" data-idx="${i}"${partAttr} value="${esc(it.kcal_per_100g)}" placeholder="kcal/100g" inputmode="decimal" title="kcal per 100 g (not in catalog)">`
    : "";
  const kcalDisplay = itemKcalCell(slot, i, partIdx);
  const naDisplay = itemNaCell(slot, i, partIdx);
  const kcalCellOut = needPer100g
    ? kcalCell
    : `<span class="text-right font-mono tabular-nums text-sm">${kcalDisplay}</span>`;
  const naCell = needPer100g
    ? `<input class="${MEAL_INP_NUM}" data-field="sodium_mg_per_100g" data-slot="${esc(slot)}" data-idx="${i}"${partAttr} value="${esc(it.sodium_mg_per_100g)}" placeholder="Na/100g" inputmode="decimal" title="sodium mg per 100 g (not in catalog)">`
    : `<span class="text-right font-mono tabular-nums text-sm">${naDisplay}</span>`;
  const nameCell = nameInput;
  const rowCls = [
    isSetHeader ? "bg-base-200/30 rounded-box" : "",
    nutritionIssue
      ? (nutritionIssue.missingFood ? "border border-error rounded-box" : "border border-warning rounded-box")
      : "",
  ].filter(Boolean).join(" ");
  const delBtn = isPart
    ? `<button type="button" class="${MEAL_ROW_ICON} btn-error justify-self-end" title="Delete ingredient" data-del-part="${esc(slot)}:${i}:${partIdx}">×</button>`
    : `<button type="button" class="${MEAL_ROW_ICON} btn-error justify-self-end" title="Delete" data-del="${esc(slot)}:${i}">×</button>`;
  const rowClsAll = [
    ...(isPart ? [] : [rowCls]),
    nutritionIssue
      ? (nutritionIssue.missingFood ? "border border-error rounded-box" : "border border-warning rounded-box")
      : "",
  ].filter(Boolean).join(" ");
  const nameColCls = isPart ? "meal-item-name meal-item-name--part" : "meal-item-name";
  return `<div class="${ITEM_ROW} ${rowClsAll}" data-meal-row${isPart ? " data-meal-part" : ""}${nutritionIssue ? ` title="${esc(nutritionIssue.msg)}"` : ""}>
    ${lead}
    <div class="min-w-0 w-full flex items-center gap-1 ${nameColCls}">${ejectPart}${nameCell}${addPartBtn}</div>
    ${countCell}
    ${gramsCell}
    ${kcalCellOut}
    ${naCell}
    ${delBtn}
  </div>`;
}

function moveIntoSetControl(slot, i) {
  const sets = setsInSlot(slot, i);
  if (!sets.length) return "";
  const opts = sets.map(({ it, j }) =>
    `<option value="${j}">${esc((it.name || "Set").trim() || "Set")}</option>`
  ).join("");
  return `<select class="select select-bordered select-xs shrink-0 w-[6.5rem] max-w-[100px] ${MEAL_INP_H} min-h-0 py-0 leading-tight" data-into-set="${esc(slot)}:${i}" title="Add to set" aria-label="Add to set">
    <option value="">→ set</option>${opts}
  </select>`;
}

function isStockMealLine(it) {
  return !!(it?.stock_lot_id || stockLotForItem(it));
}

function itemBlock(slot, i, it) {
  const dish = itemHasParts(it) && (slot === "stock" || !isStockMealLine(it));
  const parts = dish
    ? (it.parts || []).map((p, j) => nameGramsRow(slot, i, p, j)).join("")
    : "";
  return `<div class="item-block w-full min-w-0"${dish ? " data-meal-set" : ""}>
    ${nameGramsRow(slot, i, it, null)}
    ${dish ? `<div class="${MEAL_PARTS}">${parts}</div>` : ""}
  </div>`;
}

function itemNutrientTags(it) {
  const k = Number(it.kcal);
  const n = Number(it.sodium_mg);
  const bits = [];
  if (!Number.isNaN(k) && k > 0) bits.push(`${fmt(k, 0)} kcal`);
  if (!Number.isNaN(n) && n > 0) bits.push(`${fmt(n, 0)} mg Na`);
  return bits.join(", ");
}

function formatItemLine(it) {
  if (!it?.name) return "";
  if (it.parts && it.parts.length) {
    const inner = it.parts.filter((p) => p.name).map((p) => `${p.name} ${p.grams || 0}g`).join(" + ");
    return `${it.name} ${it.grams || 0}g (${inner})`;
  }
  return `${it.name} ${it.grams || 0}g`;
}

function mealText(m) {
  if (!m) return "";
  const items = (m.items || []).filter((it) => it.name).map(formatItemLine);
  if (items.length) return items.join(", ");
  const raw = (m.raw || "").replace(/\s+/g, " ").trim();
  if (!raw || raw === "-" || raw === "\u2014" || /^[-\u2014\-]+$/.test(raw) || /^[-\u2014]\s*\(no /i.test(raw)) return "";
  return raw;
}

function mealLine(slot, d) {
  const m = d.meals?.[slot];
  const text = mealText(m);
  if (!text) return "";
  const totRow = m && (m.kcal || m.sodium_mg)
    ? `<div class="text-xs font-mono tabular-nums text-base-content/60 mt-0.5">${fmt(m.kcal, 0)} kcal, ${fmt(m.sodium_mg, 0)} mg</div>`
    : "";
  return `<div class="flex gap-2 text-sm py-0.5"><span class="badge badge-outline badge-sm capitalize shrink-0">${esc(slot)}</span><div class="min-w-0 flex-1"><div class="break-words">${esc(text)}</div>${totRow}</div></div>`;
}

function dayStatusForList(d) {
  return dayCardStatus(d, {
    gapItems: state.dayGapItems,
    gapIndex: state.dayGapsByDate,
    weights: state.weights,
  });
}

function dayMatchesStatusFilter(d) {
  const f = state.daysStatusFilter || "all";
  if (f === "all") return true;
  // Legacy ids from old gap/status filter
  const legacy = { ok: "green", warn: "orange", bad: "red", clear: "green" };
  const band = legacy[f] || f;
  return dayIntakeBand(d, state.kcalBudget) === band;
}

const DAYS_INTAKE_FILTERS = [
  { id: "all", label: "All" },
  { id: "green", label: "Green", swatch: "green" },
  { id: "yellow", label: "Yellow", swatch: "yellow" },
  { id: "orange", label: "Orange", swatch: "orange" },
  { id: "red", label: "Red", swatch: "red" },
  { id: "none", label: "No kcal", swatch: "none" },
];

function daysRangeSelectHtml() {
  return periodRangeSelectHtml("days-range", state.daysRange, esc, { label: "Range" });
}

function daysStatusFilterBar(counts) {
  const cur = state.daysStatusFilter || "all";
  const c = counts || { all: 0, green: 0, yellow: 0, orange: 0, red: 0, none: 0 };
  const pills = DAYS_INTAKE_FILTERS.map(({ id, label, swatch }) => {
    const active = cur === id ? " btn-primary" : " btn-ghost";
    const chip = swatch ? `${intakeSwatchHtml(swatch)} ` : "";
    const n = id === "all" ? c.all : c[id] || 0;
    return `<button type="button" class="btn btn-sm gap-1${active}" data-days-filter="${id}" aria-pressed="${cur === id}">${chip}${label}<span class="font-mono tabular-nums opacity-70">${n}</span></button>`;
  }).join("");
  return `<div class="join flex flex-wrap gap-1" role="group" aria-label="Filter by daily kcal band">${pills}</div>`;
}

function daysListCountLine(visibleCount, totalCount, bandFilter) {
  const total = Number(totalCount) || 0;
  const visible = Number(visibleCount) || 0;
  const filtered = bandFilter && bandFilter !== "all";
  const dayWord = total === 1 ? "day" : "days";
  if (!filtered) {
    return `<p class="text-sm text-base-content/60" role="status"><span class="font-mono tabular-nums">${total}</span> ${dayWord} in range</p>`;
  }
  return `<p class="text-sm text-base-content/60" role="status"><span class="font-mono tabular-nums">${visible}</span> of <span class="font-mono tabular-nums">${total}</span> ${dayWord} in range</p>`;
}

function daysFiltersToolbar(counts) {
  return `<div class="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-end mb-3">${daysRangeSelectHtml()}${daysStatusFilterBar(counts)}</div>`;
}

function daysView() {
  if (state.tabLoading && !(state.days || []).length) {
    return el(`<div class="text-base-content/60" role="status"><span class="loading loading-spinner loading-sm"></span> Loading days…</div>`);
  }
  const periodDays = state.days || [];
  const bandCounts = dayBandCounts(periodDays, state.kcalBudget);
  const visibleDays = periodDays.filter(dayMatchesStatusFilter);
  const cards = visibleDays.map((d) => {
    const meals = ["lunch", "dinner", "snack"].map((s) => mealLine(s, d)).join("");
    const meta = [d.bp ? `BP ${esc(d.bp)}` : "", d.walking ? esc(d.walking) : ""].filter(Boolean).join(", ");
    const st = dayStatusForList(d);
    const kbar = renderDayKcalBar(d.totals?.kcal, d);
    const totalsLine = dayHeaderTotalMarkup(d, d.totals);
    return `<article class="${CARD_SECTION}${kbar.cardClass}" data-open-day="${esc(d.date)}" role="button" tabindex="0"${kbar.tip ? ` title="${esc(kbar.tip)}"` : ""}>
      <div class="card-body p-4 gap-2">
      <div class="flex flex-wrap gap-2 justify-between items-baseline mb-2">
        <span class="font-mono tabular-nums whitespace-nowrap">${esc(d.date)}</span>
        ${totalsLine}
      </div>
      ${meals || `<p class="text-base-content/60">No meal text</p>`}
      ${meta ? `<p class="text-base-content/60 text-sm">${meta}</p>` : ""}
      ${dayCardListDotMarkup(st)}
      </div>
    </article>`;
  }).join("");
  const root = el(`<div>
    ${dataBackupBarHtml("days")}
    ${daysKcalChartSectionHtml()}
    <div class="flex flex-wrap gap-2 items-end mb-3">
      <label class="form-control flex-[2] min-w-0"><span class="${FF_LABEL}">Search meals</span><input class="${FF_INP}" id="days-q" value="${esc(state.daysQ)}" placeholder="moussaka, pasta..."></label>
      <button class="btn btn-primary btn-sm" id="days-go">Search</button>
    </div>
    ${daysFiltersToolbar(bandCounts)}
    ${daysListCountLine(visibleDays.length, periodDays.length, state.daysStatusFilter)}
    <div class="flex flex-col gap-3">${
      cards ||
      `<p class="text-base-content/60">${
        state.daysStatusFilter && state.daysStatusFilter !== "all" ? "No days match this filter" : "No days"
      }</p>`
    }</div>
  </div>`);
  mountDaysKcalChart(
    root.querySelector("[data-days-kcal-chart]"),
    visibleDays,
    state.kcalBudget,
    sedentaryKcalForBudget(),
  );
  return root;
}

function stockLotActive(lot) {
  if ((lot.remaining_grams || 0) <= 0.01) return false;
  if (lot.active != null) return !!lot.active;
  return true;
}

function ensureStockMealDraft() {
  if (!state.stockForm) state.stockForm = emptyStockForm();
  const set = state.stockForm.set || emptyStockSet();
  if (!set.parts?.length) set.parts = [emptyItem()];
  set.is_set = true;
  state.stockForm.set = set;
  if (!state.mealDraft) state.mealDraft = emptyDraft();
  state.mealDraft.stock = { items: [set], note: "" };
  return set;
}

function serializeStockParts(set) {
  if (!set) return [];
  pruneBlankSetParts(set);
  return (set.parts || [])
    .map((p) => serializeItem({ ...p, parts: undefined, is_set: false }))
    .filter(Boolean);
}

function stockComputeMealBody() {
  ensureStockMealDraft();
  const set = state.mealDraft.stock.items[0];
  const parts = (set.parts || [])
    .map((p) => serializeItem({ ...p, parts: undefined, is_set: false }))
    .filter(Boolean);
  const grams = stockDerivedBatchGrams(set, state.stockForm);
  return {
    items: [{
      name: (set?.name || "").trim() || "Batch",
      grams: grams > 0 ? grams : 0,
      parts,
    }],
  };
}

function refreshStockNutritionThen(overlay, after) {
  refreshStockNutrition(stockComputeMealBody())
    .then(() => {
      if (overlay && !overlay.isConnected) return;
      after?.();
    })
    .catch(() => {
      after?.();
    });
}

function partNutrientPreview(p) {
  if (!p) return { kcal: null, na: null };
  const g = Number(p.grams) || 0;
  const f = foodForItem(p);
  if (!f || g <= 0) return { kcal: null, na: null };
  const kcal = f.kcal_per_100g != null ? (f.kcal_per_100g * g) / 100 : null;
  const na = f.sodium_mg_per_100g != null ? (f.sodium_mg_per_100g * g) / 100 : null;
  return { kcal, na };
}

function stockReadonlyPartRow(p) {
  const g = Number(p.grams) || 0;
  const { kcal, na } = partNutrientPreview(p);
  return `<div class="${ITEM_ROW}" data-meal-row data-meal-part>
    <span class="${MEAL_ICON_SPACER}" aria-hidden="true"></span>
    <span class="min-w-0 truncate meal-item-name">${esc(p.name)}</span>
    <span class="text-right font-mono tabular-nums text-sm">—</span>
    <span class="text-right font-mono tabular-nums text-sm">${g ? fmt(g, 0) : "-"}</span>
    <span class="text-right font-mono tabular-nums text-sm">${kcal != null ? fmt(kcal, 0) : "-"}</span>
    <span class="text-right font-mono tabular-nums text-sm">${na != null ? fmt(na, 0) : "-"}</span>
    <span class="${MEAL_ICON_SPACER}"></span>
  </div>`;
}

function stockLotIngredientsBlock(lot) {
  const parts = (lot.items || []).filter((p) => (p.name || "").trim());
  if (!parts.length) return "";
  const stats = setPartStats({ parts });
  return `<div class="mt-2">
    ${mealItemsGridHead()}
    <div class="item-block w-full min-w-0" data-meal-set>
      <div class="${ITEM_ROW}" data-meal-row>
        <span class="${MEAL_ICON_SPACER}"></span>
        <span class="text-base-content/60 text-sm meal-item-name">Ingredients (full batch)</span>
        <span class="text-right text-base-content/60 text-sm">—</span>
        <span class="text-right font-mono tabular-nums text-sm" title="ingredient rows / recipe total g">${stats.count} ${fmt(stats.grams, 0)}</span>
        <span class="text-right font-mono tabular-nums text-sm">-</span>
        <span class="text-right font-mono tabular-nums text-sm">-</span>
        <span class="${MEAL_ICON_SPACER}"></span>
      </div>
      <div class="${MEAL_PARTS}">${parts.map(stockReadonlyPartRow).join("")}</div>
    </div>
  </div>`;
}

function stockUsageRows(lot) {
  const rows = lot.usage || lot.Usage || [];
  if (!rows.length) {
    return `<p class="text-xs text-base-content/60 mt-2 m-0">No usage logged yet. <strong>left</strong> = batch minus lines below.</p>`;
  }
  const lines = rows.map((m) => {
    const when = m.meal_date
      ? `${esc(m.meal_date)} ${esc(m.meal_slot || "")}`.trim()
      : esc((m.created_at || "").slice(0, 10));
    const what =
      m.kind === "meal"
        ? when || "meal"
        : m.kind === "import"
          ? "before import"
          : m.kind === "household"
            ? "no meal log"
            : m.kind === "discarded"
              ? "discarded"
              : esc(m.kind || "adjustment");
    const note = m.note ? ` <span class="text-base-content/60">(${esc(m.note)})</span>` : "";
    const g = m.grams < 0 ? `put back ${fmt(-m.grams, 0)} g` : `-${fmt(m.grams, 0)} g`;
    return `<tr><td class="font-mono tabular-nums">${what}${note}</td><td class="font-mono tabular-nums">${g}</td><td class="text-base-content/60">${when}</td></tr>`;
  });
  return `<div class="mt-2 overflow-x-auto">
    <p class="text-xs text-base-content/60 m-0 mb-1"><strong>Usage</strong> (batch minus these = left)</p>
    <table class="table table-xs"><tbody>${lines.join("")}</tbody></table>
  </div>`;
}

function stockDensity(lot) {
  const k = lot.kcal_per_100g != null ? `${fmt(lot.kcal_per_100g, 0)} kcal` : "";
  const n = lot.sodium_mg_per_100g != null ? `${fmt(lot.sodium_mg_per_100g, 0)} mg` : "";
  if (!k && !n) return "density unknown";
  return `${k}${k && n ? ", " : ""}${n} / 100g`;
}

/** Batch size from ingredient rows; falls back to saved batch only when parts have no grams yet. */
function stockDerivedBatchGrams(set, form) {
  recomputeSetGrams(set);
  const { grams } = setPartStats({ parts: set?.parts || [] });
  if (grams > 0.01) return grams;
  const saved = numOrUndef(form?.batch_grams);
  return saved != null && saved > 0 ? saved : 0;
}

function stockBatchEditorHtml() {
  ensureStockMealDraft();
  const f = state.stockForm;
  const set = state.mealDraft.stock.items[0];
  const batchG = stockDerivedBatchGrams(set, f);
  const leftNow = f?.id ? Math.max(0, numOrUndef(f.remaining_grams) ?? batchG) : batchG;
  return `<p class="text-sm text-base-content/60 m-0">Name the batch and list ingredients with <strong>grams for the full batch</strong>. Batch total is the sum of those grams.</p>
    ${mealItemsGridHead()}
    ${itemBlock("stock", 0, set)}
    <input type="hidden" id="stock-batch" value="${esc(batchG > 0 ? batchG : "")}">
    <p class="text-sm text-base-content/60 mt-2 m-0">Batch total: <span class="font-mono tabular-nums" id="stock-batch-display">${batchG > 0 ? `${fmt(batchG, 0)} g` : "add ingredient grams"}</span></p>
    ${f?.id ? `<p class="text-xs text-base-content/60 mt-1 m-0">In fridge: <span class="font-mono tabular-nums">${fmt(leftNow, 0)} g</span> left (updated by logging and use, not here)</p>` : ""}
    <div class="flex flex-wrap gap-2 items-end mt-2">
      <label class="form-control min-w-0"><span class="${FF_LABEL}">Prepared</span>${isoDateInputHtml({ id: "stock-date", value: f.prepared_date, extraClass: FF_INP })}</label>
    </div>
    <label class="form-control min-w-0 flex-1 mt-2"><span class="${FF_LABEL}">Notes (optional per-100g in text)</span><textarea id="stock-notes" class="${FF_TEXTAREA}" rows="2">${esc(f.notes)}</textarea></label>`;
}

function refreshStockModalBody(overlay) {
  const body = overlay.querySelector("[data-modal-body]");
  if (body) body.innerHTML = stockBatchEditorHtml();
  bindStockModal(overlay);
}

function bindStockModal(overlay) {
  overlay.dataset.stockBatchModal = "1";
  bindIsoInputs(overlay);
  const refresh = () => {
    refreshStockNutritionThen(overlay, () => {
      const cap = captureFieldFocus(overlay);
      refreshStockModalBody(overlay);
      restoreFieldFocus(cap, overlay);
    });
  };
  overlay.querySelectorAll("[data-field]").forEach((inp) => {
    inp.addEventListener("input", () => {
      const target = draftTarget(inp.dataset.slot, inp.dataset.idx, inp.dataset.part);
      if (!target) return;
      target[inp.dataset.field] = inp.value;
      if (inp.dataset.slot === "stock") state.stockForm.set = state.mealDraft.stock.items[0];
      if (inp.dataset.part != null && (inp.dataset.field === "grams" || inp.dataset.field === "name")) {
        const set = state.mealDraft[inp.dataset.slot]?.items[Number(inp.dataset.idx)];
        if (set) recomputeSetGrams(set);
      }
      if (inp.dataset.field === "name") {
        state.foodPick = null;
        state.suggest = {
          slot: inp.dataset.slot,
          i: Number(inp.dataset.idx),
          p: inp.dataset.part == null ? -1 : Number(inp.dataset.part),
          q: inp.value,
        };
        linkDraftItemStock(target);
        linkDraftItemFood(target);
        scheduleStockModalFieldRefresh(overlay);
      } else if (inp.dataset.field === "grams" || inp.dataset.field === "count") {
        scheduleStockModalFieldRefresh(overlay);
      }
    });
  });
  overlay.querySelectorAll("[data-food-pick]").forEach((b) => {
    b.addEventListener("click", (e) => {
      e.stopPropagation();
      const parts = b.dataset.foodPick.split(":");
      const slot = parts[0];
      const i = Number(parts[1]);
      const p = parts.length > 2 ? Number(parts[2]) : -1;
      const open = foodPickOpen(slot, i, p < 0 ? null : p);
      state.foodPick = open ? null : { slot, i, p };
      state.suggest = { slot: "", i: -1, p: -1, q: "" };
      refresh();
    });
  });
  overlay.querySelectorAll("[data-resolve-food]").forEach((b) => {
    b.addEventListener("click", () => {
      const [foodID, slot, idx, part] = b.dataset.resolveFood.split(":");
      const [cut, prep] = (b.dataset.pickVariant || "").split(":");
      resolveFoodPick(foodID, slot, idx, part === "" ? null : part, { cut, prep });
    });
  });
  overlay.querySelectorAll("[data-pick-food]").forEach((b) => {
    b.addEventListener("click", () => {
      const food = state.foods.find((f) => f.id === b.dataset.pickFood);
      const target = draftTarget(b.dataset.slot, b.dataset.idx, b.dataset.part);
      if (food && target) {
        const [cut, prep] = (b.dataset.pickVariant || "").split(":");
        applyCatalogFoodToMealItem(target, food, b.dataset.pickLabel, { cut, prep });
      }
      clearNameSuggest();
      refresh();
    });
  });
  overlay.querySelectorAll("[data-pick-set-name]").forEach((b) => {
    b.addEventListener("click", () => {
      const target = state.mealDraft[b.dataset.slot]?.items[Number(b.dataset.idx)];
      if (target) target.name = b.dataset.pickSetName;
      clearNameSuggest();
      refresh();
    });
  });
  overlay.querySelectorAll("[data-make-dish]").forEach((b) => {
    b.addEventListener("click", () => {
      const [slot, i] = b.dataset.makeDish.split(":");
      const it = state.mealDraft[slot].items[Number(i)];
      if (it && !itemHasParts(it)) {
        const first = cloneDraftItem(it);
        first.parts = undefined;
        first.is_set = false;
        it.parts = [first];
        it.name = defaultSetName(slot);
        it.is_set = true;
        recomputeSetGrams(it);
      }
      refresh();
    });
  });
  overlay.querySelectorAll("[data-add-part]").forEach((b) => {
    b.addEventListener("click", () => {
      const [slot, i] = b.dataset.addPart.split(":");
      const it = state.mealDraft[slot].items[Number(i)];
      if (!it.parts) it.parts = [];
      it.parts.push(emptyItem());
      refresh();
    });
  });
  overlay.querySelectorAll("[data-eject-part]").forEach((b) => {
    b.addEventListener("click", () => {
      const [slot, i, j] = b.dataset.ejectPart.split(":");
      ejectPartFromSet(slot, Number(i), Number(j));
      refresh();
    });
  });
  overlay.querySelectorAll("[data-del]").forEach((b) => {
    b.addEventListener("click", () => {
      const [slot, i] = b.dataset.del.split(":");
      if (slot === "stock") {
        state.mealDraft.stock.items[0] = emptyStockSet();
        state.stockForm.set = state.mealDraft.stock.items[0];
        refresh();
        return;
      }
      state.mealDraft[slot].items.splice(Number(i), 1);
      if (!state.mealDraft[slot].items.length) state.mealDraft[slot].items.push(emptyItem());
      refresh();
    });
  });
  overlay.querySelectorAll("[data-del-part]").forEach((b) => {
    b.addEventListener("click", () => {
      const [, i, j] = b.dataset.delPart.split(":");
      const it = state.mealDraft.stock.items[Number(i)];
      it.parts.splice(Number(j), 1);
      recomputeSetGrams(it);
      refresh();
    });
  });
}

function closeStockModalCleanup() {
  state.stockForm = emptyStockForm();
  delete state.mealDraft?.stock;
  clearStockNutrition();
}

async function openStockBatchModal() {
  const editing = !!state.stockForm?.id;
  const f = state.stockForm;
  const title = editing ? "Edit batch" : "New batch";
  const activeOn = f?.active !== false;
  await customModal({
    title,
    titleAside: `<label class="label cursor-pointer gap-2" title="Active in fridge">
      <span class="label-text text-sm">Active</span>
      ${daisyToggle({ checked: activeOn, extraAttrs: 'id="stock-active"' })}
    </label>`,
    bodyHtml: stockBatchEditorHtml(),
    submitLabel: editing ? "Save changes" : "Save batch",
    extraClass: "",
    deleteLabel: editing ? "Delete" : undefined,
    onDelete: editing
      ? () => deleteStock(state.stockForm?.id)
      : undefined,
    onMount: (overlay) => {
      bindStockModal(overlay);
      scheduleStockModalFieldRefresh(overlay);
    },
    onSubmit: async () => {
      try {
        syncStockFormFromDOM();
        const ok = await saveStock();
        return ok !== false;
      } catch (e) {
        toast(e.message);
        return false;
      }
    },
  });
  closeStockModalCleanup();
  render();
}

function beginNewStockBatch() {
  state.stockForm = emptyStockForm();
  openStockBatchModal();
}

function beginEditStockBatch(lotId) {
  const lot = state.stock.find((x) => x.id === lotId);
  if (!lot) return;
  const set = emptyStockSet();
  set.name = lot.name || "";
  set.is_set = true;
  set.stock_lot_id = "";
  set.parts = (lot.items || []).map((raw) => {
    const p = cloneDraftItem(raw);
    p.parts = undefined;
    p.is_set = false;
    if (p.stock_lot_id === lot.id) p.stock_lot_id = "";
    return p;
  });
  if (!set.parts.length) set.parts = [emptyItem()];
  state.stockForm = {
    id: lot.id,
    prepared_date: lot.prepared_date || "",
    batch_grams: lot.batch_grams,
    remaining_grams: lot.remaining_grams,
    active: stockLotActive(lot),
    notes: lot.notes || "",
    set,
  };
  ensureStockMealDraft();
  state.mealDraft.stock.items[0] = set;
  state.stockForm.set = set;
  openStockBatchModal();
}

function stockView() {
  const lotCard = (lot) => {
    const active = stockLotActive(lot);
    const left = Math.max(0, lot.remaining_grams || 0);
    const batch = lot.batch_grams || 0;
    const hasLeft = left > 0.01;
    const qtyLabel = batch > 0 ? `${fmt(left, 0)}/${fmt(batch, 0)} g` : `${fmt(left, 0)} g`;
    return `<article class="${CARD_SECTION} mb-3${active ? "" : " border-base-content/20"}" data-open-stock="${esc(lot.id)}" role="button" tabindex="0">
      <div class="card-body gap-2 p-4">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <label class="label cursor-pointer gap-2 py-0 min-h-0" title="Active in fridge">
          ${daisyToggle({ checked: active, extraAttrs: `data-stock-active="${esc(lot.id)}"` })}
          <span class="label-text text-xs">${active ? "On" : "Off"}</span>
        </label>
        <h3 class="card-title text-base flex-1 m-0">${esc(lot.name)}</h3>
        <span class="font-mono tabular-nums text-sm text-base-content/60" title="current / original batch">${qtyLabel}</span>
      </div>
      <p class="text-sm text-base-content/60 m-0">${esc(stockDensity(lot))}${lot.prepared_date ? `, ${esc(lot.prepared_date)}` : ""}</p>
      ${stockLotIngredientsBlock(lot)}
      ${stockUsageRows(lot)}
      ${lot.notes ? `<p class="text-xs text-base-content/60 max-h-20 overflow-auto mt-2 m-0">${esc(lot.notes)}</p>` : ""}
      ${!active && hasLeft ? `<p class="text-xs text-base-content/60 mt-1 m-0">Imported from an old “still in stock” note. Toggle <strong>On</strong> only if it is still in the fridge.</p>` : ""}
      <div class="flex flex-wrap gap-2 mt-2" data-stock-actions>
        <button type="button" class="btn btn-outline btn-sm" data-edit-stock="${esc(lot.id)}">Edit</button>
        ${hasLeft && active ? `<button type="button" class="btn btn-primary btn-sm" data-draw-stock="${esc(lot.id)}">Log my meal</button>` : ""}
        ${hasLeft ? `<button type="button" class="btn btn-outline btn-sm" data-stock-use="${esc(lot.id)}">Use (no log)</button>
        <button type="button" class="btn btn-outline btn-sm" data-stock-discard="${esc(lot.id)}">Discard</button>` : ""}
      </div>
      </div>
    </article>`;
  };
  const lots = state.stock || [];
  const activeLots = lots.filter((lot) => stockLotActive(lot));
  const usedLots = lots.filter((lot) => !stockLotActive(lot));
  const activeRows = activeLots.map(lotCard).join("");
  const usedRows = usedLots.map(lotCard).join("");
  return el(`<div>
    ${dataBackupBarHtml("stock", { extra: `<button type="button" class="btn btn-primary btn-sm" id="stock-add">New batch</button>` })}
    <h2 class="text-lg font-semibold mt-2">Active (${activeLots.length})</h2>
    <div class="flex flex-col gap-2">${activeRows || `<p class="text-base-content/60">Nothing left in the fridge</p>`}</div>
    <h2 class="text-lg font-semibold mt-6">Used up (${usedLots.length})</h2>
    <div class="flex flex-col gap-2 opacity-75">${usedRows || `<p class="text-base-content/60">No history yet</p>`}</div>
  </div>`);
}

function foodDash(v) {
  if (v == null || v === "" || Number.isNaN(Number(v))) return EMPTY_CELL;
  return fmt(Number(v), v < 10 ? 1 : 0);
}

function foodEnergyEU(f) {
  const kcal = f.kcal_per_100g;
  if (kcal == null || Number.isNaN(Number(kcal))) return EMPTY_CELL;
  const kj = f.energy_kj_per_100g != null ? Number(f.energy_kj_per_100g) : Math.round(Number(kcal) * 4.184);
  return `${Math.round(kj)} kJ / ${fmt(kcal, 0)} kcal`;
}

function foodSaltG(f) {
  if (f.salt_g_per_100g != null && !Number.isNaN(Number(f.salt_g_per_100g))) return Number(f.salt_g_per_100g);
  if (f.sodium_mg_per_100g != null) return saltGFromSodiumMg(Number(f.sodium_mg_per_100g));
  return null;
}

/** Shorter catalog title; whole Foods tab is per 100 g. */
function foodCatalogDisplayName(name) {
  if (!name) return "";
  let n = name;
  n = n.replace(/\s*\(label,\s*per\s*100\s*g\)/gi, "");
  n = n.replace(/\s*\(per\s*100\s*g\)/gi, "");
  n = n.replace(/\s*,\s*per\s*100\s*g/gi, "");
  n = n.replace(/\s*\(label\)/gi, "");
  return n.replace(/\s{2,}/g, " ").replace(/,\s*$/, "").trim();
}

function foodEuLabelHtml(f, { compact = false } = {}) {
  const salt = foodSaltG(f);
  const line = (label, val, sub = false) => {
    if (val == null || val === EMPTY_CELL) return "";
    const labCls = sub ? "pl-3 text-base-content/70" : "";
    return `<tr><td class="text-xs ${labCls}">${label}</td><td class="text-xs text-right font-mono tabular-nums">${val}</td></tr>`;
  };
  const rows = [
    line("Energy", foodEnergyEU(f)),
    line("Fat", foodDash(f.fat_g_per_100g) !== EMPTY_CELL ? `${foodDash(f.fat_g_per_100g)} g` : EMPTY_CELL),
    line("of which saturates", foodDash(f.saturates_g_per_100g) !== EMPTY_CELL ? `${foodDash(f.saturates_g_per_100g)} g` : null, true),
    line("Carbohydrate", foodDash(f.carbs_g_per_100g) !== EMPTY_CELL ? `${foodDash(f.carbs_g_per_100g)} g` : EMPTY_CELL),
    line("of which sugars", foodDash(f.sugars_g_per_100g) !== EMPTY_CELL ? `${foodDash(f.sugars_g_per_100g)} g` : null, true),
    line("Protein", foodDash(f.protein_g_per_100g) !== EMPTY_CELL ? `${foodDash(f.protein_g_per_100g)} g` : EMPTY_CELL),
    line("Salt", salt != null ? `${foodDash(salt, 2)} g` : EMPTY_CELL),
  ].filter(Boolean).join("");
  const title = compact
    ? "Nutrition"
    : `Nutrition <span class="text-base-content/60 font-normal">(per 100 g)</span>`;
  return `<div class="card card-compact bg-base-200 border border-base-300 shadow-none min-w-[11rem]">
    <div class="card-body p-3 gap-1">
      <h4 class="card-title text-xs">${title}</h4>
      ${rows
    ? `<div class="overflow-x-auto"><table class="table table-xs table-pin-rows"><tbody>${rows}</tbody></table></div>`
    : `<p class="text-base-content/60 text-xs m-0">No label data</p>`}
    </div>
  </div>`;
}

function foodAliasesForId(foodId) {
  if (!foodId) return [];
  return (state.foodAliases || [])
    .filter((a) => a.food_id === foodId)
    .map((a) => a.alias)
    .sort((a, b) => a.localeCompare(b, undefined, { sensitivity: "base" }));
}

function foodAliasChipsHtml(aliases) {
  if (!aliases?.length) {
    return `<p class="text-sm text-base-content/60 m-0">No aliases yet — meal lines use exact catalog name or fuzzy match.</p>`;
  }
  return `<div class="flex flex-wrap gap-2" role="list">${aliases.map((alias) =>
    `<span class="badge badge-outline gap-1 pr-0 font-normal" role="listitem">
      <span class="font-mono tabular-nums">${esc(alias)}</span>
      <button type="button" class="btn btn-ghost btn-xs min-h-0 h-5 px-1" data-food-alias-remove="${esc(alias)}" title="Remove alias" aria-label="Remove alias ${esc(alias)}">×</button>
    </span>`
  ).join("")}</div>`;
}

function readFoodFormPendingAliases(overlay) {
  const box = overlay.querySelector("[data-food-alias-bulk]");
  if (!box) return [];
  return [...new Set((box.value || "").split(/\n+/).map((n) => n.trim()).filter(Boolean))];
}

async function addFoodAliases(foodId, names) {
  const uniq = [...new Set((names || []).map((n) => n.trim()).filter(Boolean))];
  if (!foodId || !uniq.length) return 0;
  for (const alias of uniq) {
    await api(`/foods/${foodId}/aliases`, { method: "POST", body: JSON.stringify({ alias }) });
  }
  return uniq.length;
}

function foodFormAliasesSectionHtml(foodId) {
  const listInner = foodId
    ? foodAliasChipsHtml(foodAliasesForId(foodId))
    : `<p class="text-sm text-base-content/60 m-0">Aliases saved with the food — one per line on the right.</p>`;
  const saveHint = foodId
    ? ""
    : `<p class="text-xs text-base-content/60 mt-1 m-0">Lines above are saved when you click Add food.</p>`;
  const addBtn = foodId
    ? `<button type="button" class="btn btn-outline btn-sm" data-food-alias-bulk-add>Add</button>`
    : "";
  return `<section class="flex flex-col gap-2"${foodId ? ` data-food-alias-section="${esc(foodId)}"` : ""}>
    <div class="flex flex-col gap-3 lg:flex-row">
      <div class="flex-[3] min-w-0 flex flex-col gap-1">
        <p class="label-text text-xs uppercase opacity-60">Aliases</p>
        <div data-food-alias-list>${listInner}</div>
      </div>
      <div class="flex-[2] min-w-0 flex flex-col gap-2">
        <div class="flex flex-wrap gap-2 items-end">
          <label class="form-control flex-[2] min-w-0"><span class="${FF_LABEL}">Add aliases (one per line)</span>
            <textarea class="${FF_TEXTAREA}" data-food-alias-bulk rows="3" placeholder="peppers&#10;grilled peppers&#10;florina pepper"></textarea>
          </label>
          ${addBtn}
        </div>
        ${saveHint}
      </div>
    </div>
  </section>`;
}

function mountFoodFormAliases(overlay, foodId) {
  const section = overlay.querySelector("[data-food-alias-section]");
  if (!section || !foodId) return;

  const refresh = () => {
    const list = section.querySelector("[data-food-alias-list]");
    if (list) list.innerHTML = foodAliasChipsHtml(foodAliasesForId(foodId));
    list?.querySelectorAll("[data-food-alias-remove]").forEach((btn) => {
      btn.addEventListener("click", () => removeFoodAlias(foodId, btn.dataset.foodAliasRemove, refresh));
    });
  };

  const addNames = async (names) => {
    try {
      const n = await addFoodAliases(foodId, names);
      if (!n) return;
      await loadFoods();
      refresh();
      render();
      toast(n > 1 ? `Added ${n} aliases` : "Alias added");
    } catch (e) {
      toast(e.message || "alias failed");
    }
  };

  section.querySelector("[data-food-alias-bulk-add]")?.addEventListener("click", () => {
    const box = section.querySelector("[data-food-alias-bulk]");
    const lines = (box?.value || "").split(/\n+/);
    addNames(lines).then(() => { if (box) box.value = ""; });
  });

  refresh();
}

async function removeFoodAlias(foodId, alias, refresh) {
  try {
    await api(`/foods/${foodId}/aliases`, { method: "DELETE", body: JSON.stringify({ alias }) });
    await loadFoods();
    refresh?.();
    render();
    toast("Alias removed");
  } catch (e) {
    toast(e.message || "alias failed");
  }
}

function catalogFoodPackGrams(food) {
  const g = Number(food?.label_grams);
  return g > 0 ? g : null;
}

function foodFormKindFromFood(f) {
  const k = String(f?.kind || "").toLowerCase();
  if (k === "raw" || k === "packed") return k;
  return "packed";
}

function foodCatalogPickHint(f) {
  const g = catalogFoodPackGrams(f);
  const k = Number(f?.label_kcal);
  if (g != null && k > 0 && g !== 100) return `${fmt(k, 0)} kcal / ${fmt(g, 0)} g pack`;
  if (foodFormKindFromFood(f) === "raw" && f?.variants?.enabled && f?.kcal_per_100g != null) {
    const cut = (f.variants.default_cut || "skinless").replace(/_/g, " ");
    const prep = f.variants.default_prep || "cooked";
    return `${fmt(f.kcal_per_100g, 0)} kcal/100g · ${cut} · ${prep}`;
  }
  if (f?.kcal_per_100g != null) return `${fmt(f.kcal_per_100g, 0)} kcal/100g`;
  return "";
}

function applyCatalogFoodToMealItem(target, food, labelName, { cut, prep } = {}) {
  if (!target || !food) return;
  target.name = (labelName || "").trim() || food.name;
  target.food_id = food.id;
  target.kcal_per_100g = "";
  target.sodium_mg_per_100g = "";
  const block = foodVariantNutritionBlock(food, cut, prep);
  if (block?.kcal_per_100g != null) target.kcal_per_100g = String(block.kcal_per_100g);
  if (block?.sodium_mg_per_100g != null) target.sodium_mg_per_100g = String(block.sodium_mg_per_100g);
  const g = catalogFoodPackGrams(food);
  if (g != null) target.grams = String(g);
}

function foodPickButtonHtml({ foodId, pickLabel, variant, label, hint, slot, idx, partAttr = "" }) {
  const vAttr = variant ? ` data-pick-variant="${esc(variant)}"` : "";
  return `<li><button type="button" class="${SUGGEST_BTN}" data-pick-food="${esc(foodId)}" data-pick-label="${esc(pickLabel)}" data-slot="${esc(slot)}" data-idx="${idx}"${partAttr}${vAttr}>
      ${esc(label)} <span class="text-base-content/60">${esc(hint || "—")}</span>
    </button></li>`;
}

function expandFoodPickButtons(f, pickLabel, slot, idx, partAttr = "") {
  const choices = foodVariantPickChoices(f);
  const baseLabel = pickLabel || cleanFoodDisplayName(f.name);
  return choices.map((ch) => foodPickButtonHtml({
    foodId: f.id,
    pickLabel: baseLabel,
    variant: ch.value,
    label: ch.label,
    hint: ch.hint,
    slot,
    idx,
    partAttr,
  })).join("");
}

function foodFormLabelGrams(f) {
  const g = Number(f?.label_grams);
  return g > 0 ? g : 100;
}

function foodLabelKcalDisplay(f) {
  if (f?.label_kcal != null && Number(f.label_kcal) > 0) return String(Math.round(Number(f.label_kcal)));
  return foodLabelAmount(f?.kcal_per_100g, foodFormLabelGrams(f), { decimals: 0 });
}

function foodLabelSodiumTotalDisplay(f) {
  if (f?.label_sodium_mg != null && Number(f.label_sodium_mg) >= 0) {
    const n = Number(f.label_sodium_mg);
    if (Number.isFinite(n)) return String(Math.round(n * 10) / 10);
  }
  return foodLabelAmount(f?.sodium_mg_per_100g, foodFormLabelGrams(f), { decimals: 1 });
}

/** Salt total for the label amount — only when salt was stored, not derived from sodium. */
function foodLabelSaltTotalDisplay(f) {
  if (f?.salt_g_per_100g != null && !Number.isNaN(Number(f.salt_g_per_100g))) {
    return foodLabelAmount(f.salt_g_per_100g, foodFormLabelGrams(f), { decimals: 3 });
  }
  return "";
}

function foodFormUseSaltMode(f) {
  if (!f) return false;
  const hasSod = f.label_sodium_mg != null && Number(f.label_sodium_mg) > 0;
  const hasSalt = f.salt_g_per_100g != null && Number(f.salt_g_per_100g) > 0;
  return hasSalt && !hasSod;
}

function foodLabelAmount(per100, labelG, { decimals = 1 } = {}) {
  if (per100 == null || per100 === "" || Number.isNaN(Number(per100))) return "";
  const g = labelG > 0 ? labelG : 100;
  const v = (Number(per100) * g) / 100;
  if (decimals === 0) return String(Math.round(v));
  const r = Math.round(v * 10 ** decimals) / 10 ** decimals;
  return String(r);
}

function foodFormPer100Cell(val, unit = "") {
  if (val == null || !Number.isFinite(val)) return `<span class="font-mono tabular-nums text-base-content/60">—</span>`;
  const n = unit === "kcal" ? fmt(val, 0) : fmt(val, val < 10 ? 2 : 1);
  return `<span class="font-mono tabular-nums">${esc(n)}${unit ? ` <span class="text-base-content/60">${unit}</span>` : ""}</span>`;
}

function foodFormPer100GridHtml(rows) {
  return `<div class="${GRID_4}">${rows.map(([lab, cell]) =>
    `<div class="form-control min-w-0">
      <span class="label-text text-xs uppercase opacity-60">${esc(lab)}</span>
      <div class="text-sm">${cell}</div>
    </div>`
  ).join("")}</div>`;
}

function refreshFoodFormPer100Preview(overlay) {
  const host = overlay.querySelector("[data-food-form-per100]");
  if (!host) return;
  const val = (k) => overlay.querySelector(`[data-ff="${k}"]`)?.value ?? "";
  const labelG = numOrUndef(overlay.querySelector("[data-ff-label-g]")?.value) ?? numOrUndef(val("label_g"));
  if (!labelG || labelG <= 0) {
    host.innerHTML = `<p class="text-sm text-base-content/60 m-0">Enter label total g to see per 100 g.</p>`;
    return;
  }
  const toPer100 = (raw) => {
    const n = numOrUndef(raw);
    return n == null ? null : (n / labelG) * 100;
  };
  let kcalT = numOrUndef(val("kcal"));
  let kjT = numOrUndef(val("kj"));
  if (kjT == null && kcalT != null) kjT = Math.round(kcalT * 4.184 * 10) / 10;
  const kcal = toPer100(kcalT);
  let kj = toPer100(kjT);
  if (kj == null && kcal != null) kj = Math.round(kcal * 4.184 * 10) / 10;
  const useSalt = !!overlay.querySelector("[data-ff-use-salt]")?.checked;
  const naTotal = numOrUndef(val("na"));
  const naPer100 = toPer100(naTotal);
  const rows = [
    ["kcal", foodFormPer100Cell(kcal, "kcal")],
    ["kJ", foodFormPer100Cell(kj, "kJ")],
    ["fat", foodFormPer100Cell(toPer100(val("fat")), "g")],
    ["sat", foodFormPer100Cell(toPer100(val("sat")), "g")],
    ["carbs", foodFormPer100Cell(toPer100(val("carbs")), "g")],
    ["sugars", foodFormPer100Cell(toPer100(val("sugars")), "g")],
    ["protein", foodFormPer100Cell(toPer100(val("protein")), "g")],
    [useSalt ? "salt" : "Na", foodFormPer100Cell(naPer100, useSalt ? "g" : "mg")],
  ];
  host.innerHTML = `<p class="text-xs text-base-content/60 mb-1 m-0">Stored in catalog (per 100 g)</p>
    ${foodFormPer100GridHtml(rows)}`;
}

function bindFoodFormNaField(overlay) {
  const toggle = overlay.querySelector("[data-ff-use-salt]");
  const inp = overlay.querySelector('[data-ff="na"]');
  const lab = overlay.querySelector("[data-ff-na-label]");
  if (!toggle || !inp || !lab) return;
  const sodiumTitle = "Sodium mg for the label total. Meal math uses this value.";
  const saltTitle = "Salt g (NaCl) on the EU label for this amount. 393 mg Na per 1 g salt when saved.";
  const stash = overlay._foodNaStash || { sodium: "", salt: "" };
  overlay._foodNaStash = stash;
  if (!stash._inited) {
    stash._inited = true;
    stash.sodium = overlay.querySelector("[data-ff-na-seed-sodium]")?.value ?? "";
    stash.salt = overlay.querySelector("[data-ff-na-seed-salt]")?.value ?? "";
    if (toggle.checked) stash.salt = inp.value || stash.salt;
    else stash.sodium = inp.value || stash.sodium;
  }
  const syncLabel = () => {
    const salt = toggle.checked;
    lab.textContent = salt ? "Salt g total" : "Sodium mg total";
    lab.title = salt ? saltTitle : sodiumTitle;
  };
  const onToggle = () => {
    if (toggle.checked) {
      stash.sodium = inp.value;
      inp.value = stash.salt ?? "";
    } else {
      stash.salt = inp.value;
      inp.value = stash.sodium ?? "";
    }
    syncLabel();
    refreshFoodFormPer100Preview(overlay);
  };
  syncLabel();
  toggle.addEventListener("change", onToggle);
  inp.addEventListener("input", () => {
    if (toggle.checked) stash.salt = inp.value;
    else stash.sodium = inp.value;
  });
}

function nutritionBlockFromFood(f) {
  if (!f) return {};
  return {
    kcal_per_100g: f.kcal_per_100g,
    sodium_mg_per_100g: f.sodium_mg_per_100g,
    protein_g_per_100g: f.protein_g_per_100g,
    fat_g_per_100g: f.fat_g_per_100g,
    carbs_g_per_100g: f.carbs_g_per_100g,
  };
}

function foodLooksCookedName(f) {
  const s = `${f?.name || ""} ${f?.notes || ""}`.toLowerCase();
  return /cooked|grilled|roasted/.test(s);
}

const FOOD_VARIANT_DEF_OPTS = [
  ["skinless:raw", "Skinless · raw"],
  ["skinless:cooked", "Skinless · cooked"],
  ["with_skin:raw", "With skin · raw"],
  ["with_skin:cooked", "With skin · cooked"],
];

function foodVariantsFromFood(f) {
  if (f?.variants?.enabled) return { ...f.variants, enabled: true };
  const block = nutritionBlockFromFood(f);
  const has = block.kcal_per_100g != null || block.sodium_mg_per_100g != null || block.protein_g_per_100g != null;
  const prep = foodLooksCookedName(f) ? "cooked" : "raw";
  const v = {
    enabled: true,
    default_cut: f?.variants?.default_cut || "skinless",
    default_prep: f?.variants?.default_prep || prep,
    skinless: { ...(f?.variants?.skinless || {}) },
    with_skin: { ...(f?.variants?.with_skin || {}) },
  };
  if (has && !v.skinless.raw && !v.skinless.cooked && !v.with_skin.raw && !v.with_skin.cooked) {
    if (prep === "cooked") v.skinless.cooked = { ...block };
    else v.skinless.raw = { ...block };
  }
  if (f?.edible_fraction != null && v.skinless.edible_fraction == null && v.with_skin.edible_fraction == null) {
    v.skinless.edible_fraction = f.edible_fraction;
  }
  return v;
}

function variantBlockHasData(block) {
  if (!block) return false;
  return block.kcal_per_100g != null || block.sodium_mg_per_100g != null || block.protein_g_per_100g != null;
}

function foodFormCookedEnabledInitial(v) {
  if (variantBlockHasData(v?.skinless?.cooked) || variantBlockHasData(v?.with_skin?.cooked)) return true;
  return (v?.default_prep || "cooked") !== "raw";
}

function foodHasVariantProfiles(f) {
  if (foodFormKindFromFood(f) === "packed") return false;
  return !!f?.variants?.enabled || variantBlockHasData(f?.variants?.skinless?.raw)
    || variantBlockHasData(f?.variants?.skinless?.cooked)
    || variantBlockHasData(f?.variants?.with_skin?.raw)
    || variantBlockHasData(f?.variants?.with_skin?.cooked);
}

function foodVariantNutritionBlock(f, cut, prep) {
  if (!cut || !prep) return nutritionBlockFromFood(f);
  const prof = f?.variants?.[cut]?.[prep];
  if (prof && variantBlockHasData(prof)) return prof;
  const defCut = f?.variants?.default_cut || "skinless";
  const defPrep = f?.variants?.default_prep || "cooked";
  if (cut === defCut && prep === defPrep) return nutritionBlockFromFood(f);
  return null;
}

function foodVariantPickChoices(f) {
  const baseName = cleanFoodDisplayName(f?.name || "");
  if (!foodHasVariantProfiles(f)) {
    return [{ value: "", label: baseName, hint: foodCatalogPickHint(f) }];
  }
  const out = [];
  for (const [value, shortLab] of FOOD_VARIANT_DEF_OPTS) {
    const [cut, prep] = value.split(":");
    const block = foodVariantNutritionBlock(f, cut, prep);
    if (!variantBlockHasData(block)) continue;
    const hint = block?.kcal_per_100g != null ? `${fmt(block.kcal_per_100g, 0)} kcal/100g` : foodCatalogPickHint(f);
    out.push({ value, label: `${baseName} — ${shortLab}`, hint });
  }
  if (!out.length) {
    const value = `${f?.variants?.default_cut || "skinless"}:${f?.variants?.default_prep || "cooked"}`;
    return [{ value, label: baseName, hint: foodCatalogPickHint(f) }];
  }
  return out;
}

function foodVariantVal(block, apiKey) {
  const v = block?.[apiKey];
  if (v == null || !Number.isFinite(Number(v))) return "";
  return String(v);
}

function foodVariantDualFieldHtml(cut, label, fieldKey, apiKey, rawBlock, cookedBlock) {
  const rawVal = foodVariantVal(rawBlock || {}, apiKey);
  const cookedVal = foodVariantVal(cookedBlock || {}, apiKey);
  return `<label class="${FF_VARIANT_MACRO}">
    <span class="${FF_LABEL}">${esc(label)} <span class="normal-case opacity-80">/ 100 g</span></span>
    <div class="grid grid-cols-2 gap-1" role="group" aria-label="${esc(label)}">
      <span class="flex flex-col gap-0.5">
        <span class="text-[0.65rem] uppercase opacity-50">raw</span>
        <input class="${FF_INP}" data-ff-v-cut="${esc(cut)}" data-ff-v-prep="raw" data-ff-v-field="${esc(fieldKey)}"
          value="${esc(rawVal)}" inputmode="decimal" placeholder="—" title="Raw">
      </span>
      <span class="flex flex-col gap-0.5" data-ff-v-cooked-col>
        <span class="text-[0.65rem] uppercase opacity-50">cooked</span>
        <input class="${FF_INP}" data-ff-v-cut="${esc(cut)}" data-ff-v-prep="cooked" data-ff-v-field="${esc(fieldKey)}"
          value="${esc(cookedVal)}" inputmode="decimal" placeholder="—" title="Cooked">
      </span>
    </div>
  </label>`;
}

function foodEdiblePercentDisplay(prof) {
  const f = prof?.edible_fraction ?? prof?.edibleFraction;
  if (f == null || !Number.isFinite(Number(f))) return "100";
  const pct = Number(f) * 100;
  if (pct >= 99.95) return "100";
  return String(Math.round(pct * 10) / 10);
}

function openEdibleYieldCalcModal() {
  return new Promise((resolve) => {
    const box = el(`<div class="modal modal-open modal-middle" role="dialog" aria-modal="true">
      <div class="modal-box max-w-sm">
        <h3 class="font-bold text-lg">Edible % from weighing</h3>
        <p class="text-sm text-base-content/60 m-0">One-off weigh-in. Updates edible % on this food only (meal grams unchanged).</p>
        <div class="flex flex-col gap-2 py-3">
          <label class="form-control w-full">
            <span class="${FF_LABEL}">Total weight g</span>
            <input id="eyc-total" class="${FF_INP}" type="number" min="0" step="1" inputmode="decimal" placeholder="e.g. 440" autofocus>
          </label>
          <label class="form-control w-full">
            <span class="${FF_LABEL}">Waste after g</span>
            <input id="eyc-waste" class="${FF_INP}" type="number" min="0" step="1" inputmode="decimal" placeholder="e.g. 265 (bones, seeds, peel)">
          </label>
          <p id="eyc-err" class="text-error text-sm m-0 hidden"></p>
        </div>
        <div class="modal-action">
          <button type="button" class="btn btn-outline btn-sm" data-eyc-cancel>Cancel</button>
          <button type="button" class="btn btn-primary btn-sm" data-eyc-ok>Set edible %</button>
        </div>
      </div>
      <div class="modal-backdrop modal-backdrop-inert" aria-hidden="true"></div>
    </div>`);
    const close = (pct) => {
      box.remove();
      resolve(pct);
    };
    box.querySelector("[data-eyc-cancel]")?.addEventListener("click", () => close(null));
    box.addEventListener("keydown", (e) => {
      if (e.key === "Escape") close(null);
    });
    box.querySelector("[data-eyc-ok]")?.addEventListener("click", () => {
      const total = numOrUndef(box.querySelector("#eyc-total")?.value);
      const waste = numOrUndef(box.querySelector("#eyc-waste")?.value);
      const err = box.querySelector("#eyc-err");
      if (total == null || total <= 0 || waste == null || waste < 0 || waste >= total) {
        err.classList.remove("hidden");
        err.textContent = "Need total g > 0 and waste g ≥ 0, waste less than total.";
        return;
      }
      const pct = Math.round(((total - waste) / total) * 1000) / 10;
      close(pct);
    });
    document.body.appendChild(box);
    box.querySelector("#eyc-total")?.focus();
  });
}

function foodFormActiveCut(section) {
  return section?.dataset?.ffActiveCut || "skinless";
}

function syncEdiblePercentUi(overlay, { toTop, cut } = {}) {
  const section = overlay.querySelector("[data-food-form-variants]");
  const active = cut || foodFormActiveCut(section);
  const top = overlay.querySelector("[data-ff-edible-top]");
  const store = overlay.querySelector(`[data-ff-v-edible-pct="${active}"]`);
  if (!top || !store) return;
  if (toTop) store.value = top.value;
  else top.value = store.value || "100";
}

function foodVariantCutPanelHtml(cut, prof, active) {
  const p = prof || {};
  const dual = (lab, key, api) => foodVariantDualFieldHtml(cut, lab, key, api, p.raw, p.cooked);
  const row = (cells) => `<div class="${GRID_FF_VARIANT}">${cells.join("")}</div>`;
  return `<div class="flex flex-col gap-2 w-fit max-w-full" data-ff-v-panel="${esc(cut)}"${active ? "" : " hidden"}>
    ${row([
      dual("Kcal", "kcal", "kcal_per_100g"),
      dual("Fat g", "fat", "fat_g_per_100g"),
      dual("Sat g", "sat", "saturates_g_per_100g"),
      dual("Protein g", "protein", "protein_g_per_100g"),
    ])}
    ${row([
      dual("Carbs g", "carbs", "carbs_g_per_100g"),
      dual("Sugars g", "sugars", "sugars_g_per_100g"),
      dual("Sodium mg", "na", "sodium_mg_per_100g"),
      `<div aria-hidden="true"></div>`,
    ])}
  </div>`;
}

function foodFormVariantsSectionHtml(f) {
  const v = foodVariantsFromFood(f);
  const defVal = `${v.default_cut || "skinless"}:${v.default_prep || "cooked"}`;
  const cookedOn = foodFormCookedEnabledInitial(v);
  const defOpts = FOOD_VARIANT_DEF_OPTS;
  return `<section class="flex flex-col gap-2 w-full" data-food-form-variants data-ff-active-cut="skinless">
      <input type="hidden" data-ff-v-edible-pct="skinless" value="${esc(foodEdiblePercentDisplay(v.skinless))}">
      <input type="hidden" data-ff-v-edible-pct="with_skin" value="${esc(foodEdiblePercentDisplay(v.with_skin))}">
      <div class="flex flex-wrap items-center gap-2">
        <div role="tablist" class="tabs tabs-boxed tabs-sm bg-base-300/40 p-0.5 flex-1 min-w-0">
          <button type="button" role="tab" class="tab tab-active" data-ff-v-tab="skinless">Skinless</button>
          <button type="button" role="tab" class="tab" data-ff-v-tab="with_skin">With skin</button>
        </div>
        <label class="label cursor-pointer gap-2 py-0 min-h-0 shrink-0">
          <span class="label-text text-xs">Cooked fields</span>
          ${daisyToggle({ checked: cookedOn, extraAttrs: "data-ff-v-cooked-enabled" })}
        </label>
      </div>
      ${foodVariantCutPanelHtml("skinless", v.skinless, true)}
      ${foodVariantCutPanelHtml("with_skin", v.with_skin, false)}
      <label class="form-control min-w-0 w-full sm:max-w-[14rem]">
        <span class="${FF_LABEL}">Default for meals &amp; catalog card</span>
        <select class="select select-bordered select-sm w-full" data-ff-v-default>${defOpts.map(([val, lab]) =>
    `<option value="${esc(val)}"${val === defVal ? " selected" : ""} data-ff-v-def-opt="${esc(val)}">${esc(lab)}</option>`
  ).join("")}</select>
      </label>
      <aside class="card card-compact bg-base-200 border border-base-300 p-3" data-food-form-per100-variant></aside>
    </section>`;
}

function readFoodFormVariantBlock(overlay, cut, prep) {
  const pick = (field) => overlay.querySelector(
    `[data-ff-v-cut="${cut}"][data-ff-v-prep="${prep}"][data-ff-v-field="${field}"]`
  )?.value ?? "";
  const kcal = numOrUndef(pick("kcal"));
  const protein = numOrUndef(pick("protein"));
  const fat = numOrUndef(pick("fat"));
  const carbs = numOrUndef(pick("carbs"));
  const sugars = numOrUndef(pick("sugars"));
  const sat = numOrUndef(pick("sat"));
  const na = numOrUndef(pick("na"));
  if (kcal == null && protein == null && fat == null && carbs == null && sugars == null && sat == null && na == null) {
    return undefined;
  }
  const out = {};
  if (kcal != null) out.kcal_per_100g = kcal;
  if (protein != null) out.protein_g_per_100g = protein;
  if (fat != null) out.fat_g_per_100g = fat;
  if (carbs != null) out.carbs_g_per_100g = carbs;
  if (sugars != null) out.sugars_g_per_100g = sugars;
  if (sat != null) out.saturates_g_per_100g = sat;
  if (na != null) out.sodium_mg_per_100g = na;
  if (kcal != null) out.energy_kj_per_100g = Math.round(kcal * 4.184 * 10) / 10;
  return out;
}

function foodFormCookedEnabled(overlay) {
  return !!overlay.querySelector("[data-ff-v-cooked-enabled]")?.checked;
}

function readFoodFormCutProfile(overlay, cut, cookedOn) {
  const pct = numOrUndef(overlay.querySelector(`[data-ff-v-edible-pct="${cut}"]`)?.value);
  const out = {
    raw: readFoodFormVariantBlock(overlay, cut, "raw"),
    cooked: cookedOn ? readFoodFormVariantBlock(overlay, cut, "cooked") : undefined,
  };
  if (pct != null && pct > 0 && pct < 100) {
    out.edible_fraction = Math.round((pct / 100) * 10000) / 10000;
  }
  return out;
}

function readFoodFormVariants(overlay) {
  const cookedOn = foodFormCookedEnabled(overlay);
  const def = overlay.querySelector("[data-ff-v-default]")?.value ?? "skinless:cooked";
  const [default_cut, default_prep] = def.split(":");
  return {
    enabled: true,
    default_cut: default_cut || "skinless",
    default_prep: default_prep || "cooked",
    skinless: readFoodFormCutProfile(overlay, "skinless", cookedOn),
    with_skin: readFoodFormCutProfile(overlay, "with_skin", cookedOn),
  };
}

function syncFoodFormDefaultSelect(overlay) {
  const sel = overlay.querySelector("[data-ff-v-default]");
  if (!sel) return;
  const cookedOn = foodFormCookedEnabled(overlay);
  const cur = sel.value;
  let needPick = false;
  sel.querySelectorAll("option").forEach((opt) => {
    const isCooked = String(opt.value || "").endsWith(":cooked");
    opt.hidden = !cookedOn && isCooked;
    opt.disabled = !cookedOn && isCooked;
    if (!cookedOn && isCooked && opt.value === cur) needPick = true;
  });
  if (needPick) {
    const raw = sel.querySelector('option[value="skinless:raw"]:not([hidden])');
    sel.value = raw?.value || "skinless:raw";
  }
}

function setFoodFormCookedFields(overlay, on) {
  overlay.querySelectorAll('[data-ff-v-prep="cooked"]').forEach((inp) => {
    inp.disabled = !on;
    if (!on) inp.value = "";
  });
  overlay.querySelectorAll("[data-ff-v-cooked-col]").forEach((col) => {
    col.classList.toggle("opacity-40", !on);
  });
  syncFoodFormDefaultSelect(overlay);
  refreshFoodFormVariantPreview(overlay);
}

function refreshFoodFormVariantPreview(overlay) {
  const host = overlay.querySelector("[data-food-form-per100-variant]");
  if (!host || !overlay.querySelector("[data-food-form-variants]")) return;
  const def = overlay.querySelector("[data-ff-v-default]")?.value ?? "skinless:cooked";
  const [cut, prep] = def.split(":");
  const block = readFoodFormVariantBlock(overlay, cut, prep) || {};
  let kj = block.energy_kj_per_100g;
  if (kj == null && block.kcal_per_100g != null) kj = Math.round(block.kcal_per_100g * 4.184 * 10) / 10;
  const rows = [
    ["kcal", foodFormPer100Cell(block.kcal_per_100g, "kcal")],
    ["kJ", foodFormPer100Cell(kj, "kJ")],
    ["fat", foodFormPer100Cell(block.fat_g_per_100g, "g")],
    ["sat", foodFormPer100Cell(block.saturates_g_per_100g, "g")],
    ["carbs", foodFormPer100Cell(block.carbs_g_per_100g, "g")],
    ["sugars", foodFormPer100Cell(block.sugars_g_per_100g, "g")],
    ["protein", foodFormPer100Cell(block.protein_g_per_100g, "g")],
    ["Na", foodFormPer100Cell(block.sodium_mg_per_100g, "mg")],
  ];
  const pct = numOrUndef(overlay.querySelector("[data-ff-edible-top]")?.value
    ?? overlay.querySelector(`[data-ff-v-edible-pct="${cut}"]`)?.value);
  const yieldNote = pct != null && pct > 0 && pct < 100
    ? ` · meal g × ${fmt(pct, 1)}% edible`
    : "";
  host.innerHTML = `<p class="text-xs text-base-content/60 mb-1 m-0">Catalog default · ${esc(cut.replace("_", " "))} · ${esc(prep)} (per 100 g edible)${esc(yieldNote)}</p>
    ${foodFormPer100GridHtml(rows)}`;
}

function foodFormKindValue(overlay) {
  return overlay.querySelector("[data-ff-kind]")?.value === "raw" ? "raw" : "packed";
}

function applyFoodFormKindUI(overlay) {
  const raw = foodFormKindValue(overlay) === "raw";
  const variants = overlay.querySelector("[data-food-form-variants]");
  const single = overlay.querySelector("[data-ff-single-label]");
  const edible = overlay.querySelector("[data-ff-edible-wrap]");
  if (variants) variants.hidden = !raw;
  if (single) {
    single.hidden = raw;
    single.classList.toggle("hidden", raw);
  }
  if (edible) edible.hidden = !raw;
  if (raw) refreshFoodFormVariantPreview(overlay);
  else refreshFoodFormPer100Preview(overlay);
}

function bindFoodFormKind(overlay) {
  overlay.querySelector("[data-ff-kind]")?.addEventListener("change", () => {
    const g = overlay.querySelector("[data-ff-label-g]")?.value;
    const lg = overlay.querySelector('[data-ff="label_g"]');
    if (lg && g) lg.value = g;
    applyFoodFormKindUI(overlay);
  });
}

function bindFoodFormVariants(overlay) {
  const section = overlay.querySelector("[data-food-form-variants]");
  if (!section) return;
  const cookedToggle = overlay.querySelector("[data-ff-v-cooked-enabled]");
  if (cookedToggle) {
    setFoodFormCookedFields(overlay, cookedToggle.checked);
    cookedToggle.addEventListener("change", () => setFoodFormCookedFields(overlay, cookedToggle.checked));
  }
  const showCut = (cut) => {
    syncEdiblePercentUi(overlay, { toTop: true });
    section.dataset.ffActiveCut = cut;
    section.querySelectorAll("[data-ff-v-panel]").forEach((p) => {
      p.hidden = p.dataset.ffVPanel !== cut;
    });
    section.querySelectorAll("[data-ff-v-tab]").forEach((t) => {
      t.classList.toggle("tab-active", t.dataset.ffVTab === cut);
    });
    syncEdiblePercentUi(overlay, { toTop: false, cut });
    refreshFoodFormVariantPreview(overlay);
  };
  section.querySelectorAll("[data-ff-v-tab]").forEach((tab) => {
    tab.addEventListener("click", () => showCut(tab.dataset.ffVTab));
  });
  showCut("skinless");
  overlay.querySelector("[data-ff-edible-top]")?.addEventListener("input", () => {
    syncEdiblePercentUi(overlay, { toTop: true });
    refreshFoodFormVariantPreview(overlay);
  });
  section.querySelectorAll("[data-ff-v-field]").forEach((inp) => {
    inp.addEventListener("input", () => refreshFoodFormVariantPreview(overlay));
  });
  section.querySelector("[data-ff-v-default]")?.addEventListener("change", () => refreshFoodFormVariantPreview(overlay));
  overlay.querySelector("[data-ff-yield-calc]")?.addEventListener("click", async () => {
    const pct = await openEdibleYieldCalcModal();
    if (pct == null) return;
    const top = overlay.querySelector("[data-ff-edible-top]");
    if (top) top.value = String(pct);
    syncEdiblePercentUi(overlay, { toTop: true });
    refreshFoodFormVariantPreview(overlay);
  });
}

function bindFoodFormPer100Preview(overlay) {
  bindFoodFormNaField(overlay);
  bindFoodFormVariants(overlay);
  bindFoodFormKind(overlay);
  applyFoodFormKindUI(overlay);
  refreshFoodFormVariantPreview(overlay);
  overlay.querySelectorAll("[data-ff]").forEach((inp) => {
    inp.addEventListener("input", () => refreshFoodFormPer100Preview(overlay));
    inp.addEventListener("change", () => refreshFoodFormPer100Preview(overlay));
  });
  overlay.querySelector("[data-ff-label-g]")?.addEventListener("input", () => {
    const g = overlay.querySelector("[data-ff-label-g]")?.value;
    const lg = overlay.querySelector('[data-ff="label_g"]');
    if (lg && g) lg.value = g;
    refreshFoodFormPer100Preview(overlay);
  });
  overlay.querySelector("[data-ff-use-salt]")?.addEventListener("change", () => refreshFoodFormPer100Preview(overlay));
}

function foodFormFieldsHtml(f) {
  const labelG = foodFormLabelGrams(f);
  const dispKcal = foodLabelKcalDisplay(f);
  const dispKj = f?.label_kcal != null
    ? String(Math.round(Number(f.label_kcal) * 4.184))
    : foodLabelAmount(f?.energy_kj_per_100g, labelG, { decimals: 0 });
  const euTitle = "EU nutrition declaration (Reg. 1169/2011). Values are for the label total g below; per 100 g is derived in the preview.";
  const useSalt = foodFormUseSaltMode(f);
  const naInitial = useSalt ? foodLabelSaltTotalDisplay(f) : foodLabelSodiumTotalDisplay(f);
  const ffLab = (t, tip = "") => `<span class="${FF_LABEL}"${tip ? ` title="${esc(tip)}"` : ""}>${t}</span>`;
  const gramsVal = labelG > 0 ? labelG : 100;
  const v = foodVariantsFromFood(f);
  const edibleTop = foodEdiblePercentDisplay(v.skinless);
  const kind = foodFormKindFromFood(f);
  return `<div class="${FF_FOOD_FORM}">
    <div class="grid grid-cols-1 sm:grid-cols-[7.5rem_minmax(0,1fr)_4.5rem_5.25rem] gap-2 items-end w-full">
    <label class="form-control min-w-0">${ffLab("Type")}
      <select class="select select-bordered select-sm w-full" data-ff-kind>
        <option value="packed"${kind === "packed" ? " selected" : ""}>Packed</option>
        <option value="raw"${kind === "raw" ? " selected" : ""}>Raw</option>
      </select>
    </label>
    <label class="form-control min-w-0">${ffLab("Name")}<input class="${FF_INP}" data-ff="name" value="${esc(f?.name || "")}"></label>
    <label class="${FF_FOOD_G}">${ffLab("Grams")}
      <input class="${FF_INP}" data-ff-label-g type="number" min="1" step="1" inputmode="decimal" value="${esc(gramsVal)}"
        title="Serving / pack grams for this catalog row (often 100).">
    </label>
    <label class="${FF_FOOD_ED}" data-ff-edible-wrap>
      <span class="inline-flex items-center gap-0.5 w-full min-h-5">
        ${ffLab("Edible %")}
        ${rowIconBtn("data-ff-yield-calc", "Calc from weighing", "⚖")}
      </span>
      <input class="${FF_INP}" data-ff-edible-top type="number" min="1" max="100" step="0.1" value="${esc(edibleTop)}" inputmode="decimal"
        title="Default 100. Per cut when using Skinless / With skin tabs.">
    </label>
    </div>
    ${foodFormVariantsSectionHtml(f)}
    <div class="flex flex-col gap-2" data-ff-single-label>
    <label class="form-control min-w-0 sm:max-w-[6rem] hidden">${ffLab("Label total g")}
      <input class="${FF_INP}" data-ff="label_g" value="${esc(f?.label_grams > 0 ? f.label_grams : 100)}" inputmode="decimal" placeholder="100"></label>
    <div class="${GRID_4}" title="${esc(euTitle)}">
      <label class="form-control min-w-0 sm:max-w-[4.2rem]">${ffLab("kcal total", euTitle)}<input class="${FF_INP}" data-ff="kcal" value="${esc(dispKcal)}" inputmode="decimal"></label>
      <label class="form-control min-w-0 sm:max-w-[4.2rem]">${ffLab("kJ total", euTitle)}<input class="${FF_INP}" data-ff="kj" value="${esc(dispKj)}" inputmode="decimal" placeholder="auto"></label>
      <label class="form-control min-w-0 sm:max-w-[4.2rem]">${ffLab("Fat g total", euTitle)}<input class="${FF_INP}" data-ff="fat" value="${esc(foodLabelAmount(f?.fat_g_per_100g, labelG))}" inputmode="decimal"></label>
      <label class="form-control min-w-0 sm:max-w-[4.2rem]">${ffLab("Sat g total", euTitle)}<input class="${FF_INP}" data-ff="sat" value="${esc(foodLabelAmount(f?.saturates_g_per_100g, labelG))}" inputmode="decimal"></label>
    </div>
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 items-end">
      <label class="form-control min-w-0 sm:max-w-[4.2rem]">${ffLab("Carbs g total", euTitle)}<input class="${FF_INP}" data-ff="carbs" value="${esc(foodLabelAmount(f?.carbs_g_per_100g, labelG))}" inputmode="decimal"></label>
      <label class="form-control min-w-0 sm:max-w-[4.2rem]">${ffLab("Sugars g total", euTitle)}<input class="${FF_INP}" data-ff="sugars" value="${esc(foodLabelAmount(f?.sugars_g_per_100g, labelG))}" inputmode="decimal"></label>
      <label class="form-control min-w-0 sm:max-w-[4.2rem]">${ffLab("Protein g total", euTitle)}<input class="${FF_INP}" data-ff="protein" value="${esc(foodLabelAmount(f?.protein_g_per_100g, labelG))}" inputmode="decimal"></label>
      <label class="form-control min-w-0 flex-1">
        <span class="flex flex-wrap items-center justify-between gap-2">
          <span class="${FF_LABEL}" data-ff-na-label>${useSalt ? "Salt g total" : "Sodium mg total"}</span>
          <label class="label cursor-pointer gap-1.5 py-0 min-h-0" title="EU label: salt (NaCl) or sodium mg — one field; toggle switches unit (values are not converted).">
            <span class="label-text-alt text-xs opacity-60">Na</span>
            ${daisyToggle({ checked: useSalt, extraAttrs: 'data-ff-use-salt aria-label="Use salt g instead of sodium mg"' })}
            <span class="label-text-alt text-xs opacity-60">salt</span>
          </label>
        </span>
        <input class="${FF_INP}" data-ff="na" value="${esc(naInitial)}" inputmode="decimal" placeholder="0">
        <input type="hidden" data-ff-na-seed-sodium value="${esc(foodLabelSodiumTotalDisplay(f))}">
        <input type="hidden" data-ff-na-seed-salt value="${esc(foodLabelSaltTotalDisplay(f))}">
      </label>
    </div>
    <aside class="card card-compact bg-base-200 border border-base-300 p-3 mt-1" data-food-form-per100></aside>
    </div>
    <label class="form-control min-w-0 w-full">${ffLab("Notes")}<textarea class="${FF_TEXTAREA}" data-ff="notes" rows="2">${esc(f?.notes || "")}</textarea></label>
    ${foodFormAliasesSectionHtml(f?.id)}
    </div>`;
}

function readFoodForm(overlay) {
  const kind = foodFormKindValue(overlay);
  if (kind === "raw") {
    syncEdiblePercentUi(overlay, { toTop: true });
    const variants = readFoodFormVariants(overlay);
    const def = overlay.querySelector("[data-ff-v-default]")?.value ?? "skinless:cooked";
    const [default_cut, default_prep] = def.split(":");
    variants.default_cut = default_cut;
    variants.default_prep = default_prep || "cooked";
    const block = readFoodFormVariantBlock(overlay, default_cut, default_prep) || {};
    const cutProf = variants[default_cut] || variants.skinless;
    const g = numOrUndef(overlay.querySelector("[data-ff-label-g]")?.value) || 100;
    if (g <= 0) return { error: "Label / pack g must be greater than 0" };
    return {
      name: (overlay.querySelector('[data-ff="name"]')?.value ?? "").trim(),
      notes: (overlay.querySelector('[data-ff="notes"]')?.value ?? "").trim(),
      kind: "raw",
      variants,
      edible_fraction: cutProf?.edible_fraction,
      label_grams: g,
      label_kcal: block.kcal_per_100g,
      label_sodium_mg: block.sodium_mg_per_100g,
      kcal_per_100g: block.kcal_per_100g,
      energy_kj_per_100g: block.energy_kj_per_100g,
      fat_g_per_100g: block.fat_g_per_100g,
      saturates_g_per_100g: block.saturates_g_per_100g,
      carbs_g_per_100g: block.carbs_g_per_100g,
      sugars_g_per_100g: block.sugars_g_per_100g,
      protein_g_per_100g: block.protein_g_per_100g,
      sodium_mg_per_100g: block.sodium_mg_per_100g,
    };
  }
  const val = (k) => overlay.querySelector(`[data-ff="${k}"]`)?.value ?? "";
  const packG = numOrUndef(overlay.querySelector("[data-ff-label-g]")?.value);
  const labelG = packG || numOrUndef(val("label_g"));
  if (!labelG || labelG <= 0) {
    return { error: "Label total g must be greater than 0" };
  }
  const toPer100 = (n) => (n == null ? undefined : (n / labelG) * 100);

  let kcalTotal = numOrUndef(val("kcal"));
  let kjTotal = numOrUndef(val("kj"));
  if (kjTotal == null && kcalTotal != null) kjTotal = Math.round(kcalTotal * 4.184 * 10) / 10;

  const kcal = toPer100(kcalTotal);
  let kj = toPer100(kjTotal);
  if (kj == null && kcal != null) kj = Math.round(kcal * 4.184 * 10) / 10;

  const useSalt = !!overlay.querySelector("[data-ff-use-salt]")?.checked;
  const naTotal = numOrUndef(val("na"));
  let labelSodium;
  let sodPer100;
  let saltPer100;
  if (useSalt) {
    saltPer100 = naTotal != null ? toPer100(naTotal) : undefined;
    labelSodium = naTotal != null ? sodiumMgFromSaltG(naTotal) : undefined;
    sodPer100 = labelSodium != null ? toPer100(labelSodium) : undefined;
  } else {
    labelSodium = naTotal;
    sodPer100 = naTotal != null ? toPer100(naTotal) : undefined;
    saltPer100 = undefined;
  }

  return {
    name: val("name").trim(),
    kind: "packed",
    label_grams: labelG,
    label_kcal: kcalTotal,
    label_sodium_mg: labelSodium,
    kcal_per_100g: kcal,
    energy_kj_per_100g: kj,
    fat_g_per_100g: toPer100(numOrUndef(val("fat"))),
    saturates_g_per_100g: toPer100(numOrUndef(val("sat"))),
    carbs_g_per_100g: toPer100(numOrUndef(val("carbs"))),
    sugars_g_per_100g: toPer100(numOrUndef(val("sugars"))),
    protein_g_per_100g: toPer100(numOrUndef(val("protein"))),
    salt_g_per_100g: saltPer100,
    sodium_mg_per_100g: sodPer100,
    notes: val("notes").trim(),
  };
}

function foodsView() {
  const q = (state.foodsQ || "").toLowerCase();
  const list = q
    ? state.foods.filter((f) => {
        if (f.name.toLowerCase().includes(q) || (f.id || "").includes(q)) return true;
        return foodAliasesForId(f.id).some((a) => a.toLowerCase().includes(q));
      })
    : state.foods;
  const cards = list.slice(0, 200).map((f) => {
    const aliases = foodAliasesForId(f.id);
    const aliasHint = aliases.length
      ? `<p class="text-sm text-base-content/60 m-0" title="Meal-line aliases">${aliases.map((a) => esc(a)).join(" · ")}</p>`
      : "";
    const pickHint = foodCatalogPickHint(f);
    const packLine = pickHint
      ? `<p class="text-sm text-base-content/60 m-0">${esc(pickHint)}</p>`
      : "";
    return `<article class="${CARD_SECTION}" data-open-food="${esc(f.id)}" role="button" tabindex="0">
    <div class="card-body p-4 flex flex-col gap-3 lg:flex-row lg:items-start">
      <div class="flex-1 min-w-0">
        <h3 class="card-title text-base">${esc(foodCatalogDisplayName(f.name))}</h3>
        ${packLine}
        ${aliasHint}
        ${f.notes ? `<p class="text-sm text-base-content/60 m-0">${esc(f.notes)}</p>` : ""}
      </div>
      <div class="shrink-0">${foodEuLabelHtml(f, { compact: true })}</div>
    </div>
  </article>`;
  }).join("");
  return el(`<div>
    ${dataBackupBarHtml("foods", { extra: `<button type="button" class="btn btn-primary btn-sm" id="foods-add">Add food</button>` })}
    <div class="flex flex-wrap gap-2 items-end mb-2">
      <label class="form-control flex-[2] min-w-0"><span class="${FF_LABEL}">Search catalog</span><input class="${FF_INP}" id="foods-q" value="${esc(state.foodsQ || "")}" placeholder="pasta, chicken..."></label>
    </div>
    <p class="text-sm text-base-content/60 mb-3 m-0">${list.length} foods (per 100 g)</p>
    <div class="flex flex-col gap-3">${cards || `<p class="text-base-content/60">No foods</p>`}</div>
  </div>`);
}

function weighInEnergy(w) {
  const b = mergedKcalBudget(state.kcalBudget);
  return energyFromInputs({
    kg: w?.kg,
    heightCm: b.height_cm,
    ageYears: b.age_years,
    sex: b.sex,
    sedentary_discount_percent: b.sedentary_discount_percent,
  });
}

function weightView() {
  const latest = state.overview?.latest_weight;
  const weightsInPeriod = recordsInRange(state.weights, state.weightRange, resolveTodayDate(state));
  const periodSelect = periodRangeSelectHtml("weight-range", state.weightRange, esc, { label: "" });
  const rows = weightsInPeriod.slice().reverse().map((w) => {
    const est = weighInEnergy(w);
    return `<tr tabindex="0" data-open-weight="${esc(w.id)}">
    <td class="font-mono tabular-nums">${esc(w.date)}</td>
    <td>${fmt(w.kg, 1)}</td>
    <td class="font-mono tabular-nums">${est ? fmt(est.bmr, 0) : "-"}</td>
    <td>${w.rhr_bpm ?? ""}</td>
    <td>${w.protocol ? "yes" : ""}</td>
    <td>${esc(w.note || "")}</td>
  </tr>`;
  }).join("");
  const root = el(`<div>
    ${dataBackupBarHtml("weight", { extra: `<button type="button" class="btn btn-primary btn-sm" id="weight-add">Add weigh-in</button>` })}
    <div class="mb-2">${periodSelect}</div>
    ${weightChartSectionHtml()}
    <div class="flex flex-wrap gap-2 items-end mb-2 mt-3">
      <p class="text-base-content/60 m-0 flex-1">Latest: ${latest ? `${fmt(latest.kg, 1)} kg on ${esc(latest.date)}` : "none"}</p>
    </div>
    <div class="card card-bordered bg-base-100 shadow-sm mt-3 overflow-x-auto">
      <table class="table table-zebra table-sm">
        <thead><tr><th>Date</th><th>kg</th><th>BMR</th><th>RHR (bpm)</th><th>Protocol</th><th>Note</th></tr></thead>
        <tbody>${rows || `<tr><td colspan="6" class="text-base-content/60">No weigh-ins</td></tr>`}</tbody>
      </table>
    </div>
  </div>`);
  mountWeightChart(root.querySelector("[data-weight-chart]"), weightsInPeriod, {
    bmrForKg: (kg) => weighInEnergy({ kg })?.bmr ?? null,
  });
  return root;
}

function latestEnergyKg() {
  const fromOv = Number(state.overview?.latest_weight?.kg);
  if (fromOv > 0) return fromOv;
  const w = (state.weights || []).find((row) => Number(row.kg) > 0);
  return Number(w?.kg) || 0;
}

function sedentaryKcalForBudget(budget = state.kcalBudget) {
  const b = mergedKcalBudget(budget);
  const kg = latestEnergyKg();
  const est = energyFromInputs({
    kg,
    heightCm: b.height_cm,
    ageYears: b.age_years,
    sex: b.sex,
    sedentary_discount_percent: b.sedentary_discount_percent,
  });
  if (est?.sedentary > 0) return est.sedentary;
  const fromApi = Number(budget?.sedentary_kcal);
  return fromApi > 0 ? Math.round(fromApi) : null;
}

function latestEnergyKgLabel() {
  const w = state.overview?.latest_weight;
  if (w && Number(w.kg) > 0) {
    return `${fmt(w.kg, 1)} kg on ${w.date}`;
  }
  return "";
}

function energyEstimateHtml(b, kg) {
  const est = energyFromInputs({
    kg,
    heightCm: b.height_cm,
    ageYears: b.age_years,
    sex: b.sex,
    sedentary_discount_percent: b.sedentary_discount_percent,
  });
  if (!kg) {
    return `<p class="text-base-content/60 text-sm mt-2 m-0" id="energy-estimate">Add a weigh-in to compute daily burn.</p>`;
  }
  if (!est) {
    return `<p class="text-base-content/60 text-sm mt-2 m-0" id="energy-estimate">Fill sex, age, and height. Weight used: ${fmt(kg, 1)} kg.</p>`;
  }
  const discNote = est.sedentaryDiscountPercent > 0
    ? ` (${fmt(est.sedentaryDiscountPercent, 0)}% below standard ${fmt(est.sedentaryBase, 0)} kcal)`
    : "";
  return `<p class="text-base-content/60 text-sm mt-2 m-0" id="energy-estimate">
    Using ${fmt(est.kg, 1)} kg: BMR ${fmt(est.bmr, 0)} kcal, sedentary ${fmt(est.sedentary, 0)} kcal/day${discNote}.
  </p>`;
}

function settingsBandLegend(items) {
  return `<ul class="text-sm text-base-content/60 space-y-1 my-2">${items.map(
    ([band, text]) => `<li class="flex items-center gap-2">${intakeSwatchHtml(band)}<span>${text}</span></li>`
  ).join("")}</ul>`;
}

function settingsEnergyBudgetFromForm() {
  return {
    sex: document.getElementById("set-sex")?.value || "",
    age_years: Number(document.getElementById("set-age")?.value) || 0,
    height_cm: Number(document.getElementById("set-height")?.value) || 0,
    sedentary_discount_percent: Number(document.getElementById("set-sedentary-discount")?.value) || 0,
  };
}

function settingsView() {
  const b = mergedKcalBudget(state.kcalBudget);
  const kg = latestEnergyKg();
  const kgLine = latestEnergyKgLabel();
  const sex = b.sex === "female" ? "female" : b.sex === "male" ? "male" : "";
  const sedDisc = b.sedentary_discount_percent || 0;
  const setLab = (t) => `<span class="${FF_LABEL}">${t}</span>`;
  return el(`<div class="flex flex-col gap-0">
    <section class="${SETTINGS_CARD}"><div class="card-body gap-3 p-4">
      <h2 class="card-title text-base">Sedentary energy</h2>
      <p class="text-sm text-base-content/60 m-0">Mifflin-St Jeor resting burn (BMR), then ×1.2 for a sedentary day. Slide below to use a lower sedentary burn (mass hints use that value). Weight is your latest weigh-in.</p>
      <div class="flex flex-wrap gap-2 items-end mt-3">
        <label class="form-control min-w-0 flex-1">${setLab("Sex")}
          <select id="set-sex" class="${SETTINGS_SEL}">
            <option value=""${sex ? "" : " selected"}>Select</option>
            <option value="male"${sex === "male" ? " selected" : ""}>Male</option>
            <option value="female"${sex === "female" ? " selected" : ""}>Female</option>
          </select>
        </label>
        <label class="form-control min-w-0 flex-1">${setLab("Age (years)")}<input id="set-age" class="${SETTINGS_INP}" type="number" min="12" max="120" step="1" value="${esc(b.age_years ? String(b.age_years) : "")}"></label>
        <label class="form-control min-w-0 flex-1">${setLab("Height (cm)")}<input id="set-height" class="${SETTINGS_INP}" type="number" min="80" max="250" step="0.5" value="${esc(b.height_cm ? String(b.height_cm) : "")}"></label>
      </div>
      <label class="form-control min-w-0 flex-1 mt-3">
        ${setLab("Lower sedentary burn")}
        <div class="flex items-center gap-2 mt-1">
          <input type="range" id="set-sedentary-discount" class="range range-primary range-sm flex-1" min="0" max="50" step="1" value="${esc(String(sedDisc))}">
          <span class="font-mono tabular-nums min-w-10 text-right text-sm" id="set-sedentary-discount-val">${esc(String(sedDisc))}%</span>
        </div>
        <p class="text-xs text-base-content/60 mt-1 m-0">0% = full sedentary (BMR × 1.2). Up to 50% less kcal/day for estimates.</p>
      </label>
      <p class="text-sm text-base-content/60 mt-2 m-0">${kgLine ? `Latest weigh-in: ${esc(kgLine)}` : "No weigh-in yet."}</p>
      ${energyEstimateHtml(b, kg)}
    </div></section>
    <section class="${SETTINGS_CARD}"><div class="card-body gap-3 p-4">
      <h2 class="card-title text-base">Daily kcal bands</h2>
      <p class="text-sm text-base-content/60 m-0">The colored strip on the left of Today and each day card uses catalog day totals.</p>
      ${settingsBandLegend([
    ["green", "Green: below soft band low"],
    ["yellow", "Yellow: in soft band (target intake)"],
    ["orange", "Orange: above soft band, below gain threshold"],
    ["red", "Red: gain territory"],
  ])}
      <div class="flex flex-wrap gap-2 items-end mt-3">
        <label class="form-control min-w-0 flex-1">${setLab("Soft band low (kcal)")}<input id="set-soft-low" class="${SETTINGS_INP}" type="number" min="1" step="50" value="${esc(String(b.soft_low_kcal))}"></label>
        <label class="form-control min-w-0 flex-1">${setLab("Soft band high (kcal)")}<input id="set-soft-high" class="${SETTINGS_INP}" type="number" min="1" step="50" value="${esc(String(b.soft_high_kcal))}"></label>
        <label class="form-control min-w-0 flex-1">${setLab("Gain from (kcal)")}<input id="set-gain-min" class="${SETTINGS_INP}" type="number" min="1" step="50" value="${esc(String(b.gain_min_kcal))}"></label>
      </div>
      <p class="text-sm text-base-content/60 mt-2 m-0">Seeded from your Energy baseline notes: 1,500-1,700 kcal target; 2,200 kcal gain line.</p>
    </div></section>
    <section class="${SETTINGS_CARD}"><div class="card-body gap-3 p-4">
      <h2 class="card-title text-base">Daily sodium bands</h2>
      <p class="text-sm text-base-content/60 m-0">Colors the day total mg Na (same traffic-light pattern as kcal).</p>
      ${settingsBandLegend([
    ["green", "Green: below soft low"],
    ["yellow", "Yellow: soft low – soft high (target band)"],
    ["orange", "Orange: above soft high, below ceiling"],
    ["red", "Red: at/above ceiling"],
  ])}
      <div class="flex flex-wrap gap-2 items-end mt-3">
        <label class="form-control min-w-0 flex-1">${setLab("Soft low (mg)")}<input id="set-sodium-low" class="${SETTINGS_INP}" type="number" min="1" step="50" value="${esc(String(b.soft_low_sodium_mg))}"></label>
        <label class="form-control min-w-0 flex-1">${setLab("Soft high (mg)")}<input id="set-sodium-high" class="${SETTINGS_INP}" type="number" min="1" step="50" value="${esc(String(b.soft_high_sodium_mg))}"></label>
        <label class="form-control min-w-0 flex-1">${setLab("Ceiling (mg)")}<input id="set-sodium-ceiling" class="${SETTINGS_INP}" type="number" min="1" step="50" value="${esc(String(b.ceiling_sodium_mg))}"></label>
      </div>
      <p class="text-sm text-base-content/60 mt-2 m-0">Defaults: 1,200 / 2,000 / 2,400 mg (historical ~2 g day ceiling).</p>
    </div></section>
    <section class="${SETTINGS_CARD}"><div class="card-body gap-3 p-4">
      <h2 class="card-title text-base">Markdown diff bands (dots after kcal / mg)</h2>
      <p class="text-sm text-base-content/60 m-0">Small dots show <span class="font-mono tabular-nums">|catalog − markdown day log|</span> per nutrient. Same scale for kcal and sodium.</p>
      ${settingsBandLegend([
    ["green", "Green: diff &lt; green limit"],
    ["yellow", "Yellow: diff &lt; yellow limit"],
    ["orange", "Orange: diff &lt; orange limit"],
    ["red", "Red: diff ≥ orange limit"],
  ])}
      <div class="flex flex-wrap gap-2 items-end mt-3">
        <label class="form-control min-w-0 flex-1">${setLab("Green below")}<input id="set-diff-green" class="${SETTINGS_INP}" type="number" min="1" step="1" value="${esc(String(b.legacy_diff_green_lt))}"></label>
        <label class="form-control min-w-0 flex-1">${setLab("Yellow below")}<input id="set-diff-yellow" class="${SETTINGS_INP}" type="number" min="2" step="1" value="${esc(String(b.legacy_diff_yellow_lt))}"></label>
        <label class="form-control min-w-0 flex-1">${setLab("Orange below")}<input id="set-diff-orange" class="${SETTINGS_INP}" type="number" min="3" step="1" value="${esc(String(b.legacy_diff_orange_lt))}"></label>
      </div>
      <p class="text-sm text-base-content/60 mt-2 m-0">Defaults: 100 / 200 / 300 (e.g. 92 kcal off → green).</p>
    </div></section>
    <section class="${SETTINGS_CARD}"><div class="card-body gap-3 p-4">
      <h2 class="card-title text-base">Meal total margin</h2>
      <p class="text-sm text-base-content/60 m-0">Adds this percent <strong>once</strong> to the catalog day total (lunch + dinner + snack). Meal slot headers and item lines stay unpadded. Changes save automatically.</p>
      <label class="form-control min-w-0 flex-1 mt-3">
        ${setLab("Margin (%)")}
        <div class="flex items-center gap-2 mt-1">
          <input type="range" id="set-meal-margin" class="range range-primary range-sm flex-1" min="0" max="100" step="1" value="${esc(String(Math.round(Number(b.meal_margin_percent) || 0)))}">
          <span class="font-mono tabular-nums min-w-10 text-right text-sm" id="set-meal-margin-val">${esc(String(Math.round(Number(b.meal_margin_percent) || 0)))}%</span>
        </div>
      </label>
      <div class="card-actions justify-start px-0">
        <button type="button" class="btn btn-outline btn-sm" id="settings-reset">Reset defaults</button>
      </div>
    </div></section>
    <section class="${SETTINGS_CARD}"><div class="card-body gap-3 p-4">
      <h2 class="card-title text-base">Backup &amp; restore</h2>
      <p class="text-sm text-base-content/60 m-0">Full JSON export (foods, aliases, days, stock, weight, settings). Import here replaces everything; use tab-specific import on Foods/Days/etc. to merge only that section.</p>
      ${dataBackupBarHtml("all", { fullBackup: true })}
      <label class="flex items-center gap-2 cursor-pointer py-1 mt-2 select-none">
        ${daisyToggle({ extraAttrs: 'id="import-all-replace"' })}
        <span class="text-sm">Full import replaces all data (dangerous)</span>
      </label>
    </div></section>
    <section class="${SETTINGS_CARD}"><div class="card-body gap-3 p-4">
      <h2 class="card-title text-base">Fix imported days</h2>
      <p class="text-sm text-base-content/60 m-0">Re-parse every saved meal line (strips day-totals glued to snacks, × counts, catalog math). Restores markdown legacy totals from <span class="font-mono tabular-nums">mddata/</span>.</p>
      <div class="card-actions justify-start px-0 mt-2">
        <button type="button" class="btn btn-primary btn-sm" id="settings-repair-days">Repair all days from raw</button>
      </div>
    </div></section>
  </div>`);
}

function applyKcalBudgetFromOverview(overview) {
  if (overview?.settings) {
    state.kcalBudget = mergedKcalBudget(overview.settings);
  }
}

let settingsSaveTimer = null;
let settingsSaveInFlight = false;

function scheduleSettingsSave() {
  clearTimeout(settingsSaveTimer);
  settingsSaveTimer = setTimeout(() => {
    saveKcalBudgetFromForm({ quiet: true }).catch((e) => toast(e.message));
  }, 450);
}

function refreshDaysKcalChartIfPresent() {
  if (state.tab !== "days") return;
  const host = document.querySelector("[data-days-kcal-chart]");
  if (!host) return;
  const periodDays = recordsInRange(state.days || [], state.daysRange, resolveTodayDate(state));
  const visibleDays = filterDaysByStatus(periodDays, state.daysStatusFilter);
  mountDaysKcalChart(host, visibleDays, state.kcalBudget, sedentaryKcalForBudget());
}

async function saveKcalBudgetFromForm({ quiet = false } = {}) {
  if (settingsSaveInFlight) {
    scheduleSettingsSave();
    return;
  }
  settingsSaveInFlight = true;
  const body = {
    soft_low_kcal: Number(document.getElementById("set-soft-low")?.value),
    soft_high_kcal: Number(document.getElementById("set-soft-high")?.value),
    gain_min_kcal: Number(document.getElementById("set-gain-min")?.value),
    soft_low_sodium_mg: Number(document.getElementById("set-sodium-low")?.value),
    soft_high_sodium_mg: Number(document.getElementById("set-sodium-high")?.value),
    ceiling_sodium_mg: Number(document.getElementById("set-sodium-ceiling")?.value),
    meal_margin_percent: Number(document.getElementById("set-meal-margin")?.value),
    legacy_diff_green_lt: Number(document.getElementById("set-diff-green")?.value),
    legacy_diff_yellow_lt: Number(document.getElementById("set-diff-yellow")?.value),
    legacy_diff_orange_lt: Number(document.getElementById("set-diff-orange")?.value),
    sex: document.getElementById("set-sex")?.value || "",
    age_years: Number(document.getElementById("set-age")?.value) || 0,
    height_cm: Number(document.getElementById("set-height")?.value) || 0,
    sedentary_discount_percent: Number(document.getElementById("set-sedentary-discount")?.value) || 0,
  };
  try {
    const data = await api("/settings", { method: "PUT", body: JSON.stringify(body) });
    state.kcalBudget = mergedKcalBudget(data.settings);
    if (state.overview) state.overview.settings = data.settings;
    if (!quiet) toast("Settings saved");
    if (state.tab === "days") await loadDays();
    if (state.tab === "today" && state.date) {
      const dayData = await api(`/days/${encodeURIComponent(state.date)}`);
      state.day = dayData.day || state.day;
      fillDraftFromDay(state.day);
    }
    if (state.tab === "settings") {
      refreshEnergyHintFromDom();
      refreshDaysKcalChartIfPresent();
    } else {
      render();
    }
  } finally {
    settingsSaveInFlight = false;
  }
}

function refreshEnergyHintFromDom() {
  const elHint = document.getElementById("energy-estimate");
  if (!elHint) return;
  const wrap = document.createElement("div");
  wrap.innerHTML = energyEstimateHtml(settingsEnergyBudgetFromForm(), latestEnergyKg());
  const next = wrap.firstElementChild;
  if (next) elHint.replaceWith(next);
}

function bindSettingsAutoSave() {
  const ids = [
    "set-soft-low", "set-soft-high", "set-gain-min",
    "set-sodium-low", "set-sodium-high", "set-sodium-ceiling",
    "set-meal-margin", "set-diff-green", "set-diff-yellow", "set-diff-orange",
    "set-sex", "set-age", "set-height", "set-sedentary-discount",
  ];
  const schedule = () => scheduleSettingsSave();
  ids.forEach((id) => {
    const el = document.getElementById(id);
    if (!el) return;
    el.addEventListener("change", schedule);
    if (el.type === "range" || el.type === "number") {
      el.addEventListener("input", schedule);
    }
  });
}

async function reloadAfterDataImport(scope) {
  const jobs = [];
  if (scope === "foods" || scope === "all") jobs.push(loadFoods());
  if (scope === "stock" || scope === "all") jobs.push(loadStock());
  if (scope === "weight" || scope === "all") jobs.push(loadWeights());
  if (scope === "all") {
    jobs.push(
      api("/settings").then((data) => {
        state.kcalBudget = mergedKcalBudget(data.settings);
      }),
    );
  }
  await Promise.all(jobs);
  if (scope === "days" || scope === "all") {
    if (state.tab === "days") await loadDays();
    if (state.tab === "today" && state.date) await loadDay();
  }
  render();
}

function bind() {
  ensureNameSuggestDismiss();
  bindDataBackupHandlers({ onImported: reloadAfterDataImport });
  const app = document.getElementById("app");
  document.getElementById("theme-pick")?.addEventListener("change", (e) => {
    saveTheme(e.target.value);
    render();
  });
  document.getElementById("logout")?.addEventListener("click", () => {
    localStorage.removeItem(AUTH_KEY);
    state.user = "";
    state.token = "";
    render();
  });
  document.querySelectorAll("[data-nav-tab]").forEach((a) => {
    a.addEventListener("click", (e) => {
      e.preventDefault();
      goToTab(a.dataset.navTab).catch((err) => toast(err.message));
    });
  });
  document.querySelector("[data-nav-brand]")?.addEventListener("click", (e) => {
    if (state.tab === "today") return;
    e.preventDefault();
    goToTab("today").catch((err) => toast(err.message));
  });

  const dateEl = document.getElementById("day-date");
  dateEl?.addEventListener("change", async () => {
    const next = parseISODate(dateEl.value);
    if (!next) {
      dateEl.value = state.date;
      toast("Date must be YYYY-MM-DD");
      return;
    }
    state.date = next;
    syncTodayInUrl();
    await loadDay();
    render();
  });
  document.querySelectorAll("[data-shift]").forEach((b) => {
    b.addEventListener("click", async () => {
      const id = b.dataset.shift;
      const inp = document.getElementById(id);
      const cur = parseISODate(inp?.value || state.date);
      if (!cur) {
        toast("Date must be YYYY-MM-DD");
        return;
      }
      const next = shiftISODate(cur, Number(b.dataset.delta));
      if (inp) {
        inp.value = next;
        inp.dispatchEvent(new Event("input", { bubbles: true }));
      }
      if (id === "day-date") {
        state.date = next;
        syncTodayInUrl();
        await loadDay();
        render();
      }
    });
  });

  app?.querySelectorAll("[data-field]").forEach((inp) => {
    inp.addEventListener("input", () => {
      const target = draftTarget(inp.dataset.slot, inp.dataset.idx, inp.dataset.part);
      if (!target) return;
      target[inp.dataset.field] = inp.value;
      scheduleSaveDay();
      if (inp.dataset.part != null && (inp.dataset.field === "grams" || inp.dataset.field === "name")) {
        const set = state.mealDraft[inp.dataset.slot]?.items[Number(inp.dataset.idx)];
        if (set) recomputeSetGrams(set);
      }
      if (inp.dataset.field === "name") {
        state.foodPick = null;
        state.suggest = {
          slot: inp.dataset.slot,
          i: Number(inp.dataset.idx),
          p: inp.dataset.part == null ? -1 : Number(inp.dataset.part),
          q: inp.value,
        };
        linkDraftItemStock(target);
        linkDraftItemFood(target);
        scheduleMealFieldRender();
      } else if (inp.dataset.field === "count") {
        linkDraftItemStock(target);
        linkDraftItemFood(target);
      }
    });
  });
  app?.querySelectorAll("[data-food-pick]").forEach((b) => {
    b.addEventListener("click", (e) => {
      e.stopPropagation();
      const parts = b.dataset.foodPick.split(":");
      const slot = parts[0];
      const i = Number(parts[1]);
      const p = parts.length > 2 ? Number(parts[2]) : -1;
      const open = foodPickOpen(slot, i, p < 0 ? null : p);
      state.foodPick = open ? null : { slot, i, p };
      state.suggest = { slot: "", i: -1, p: -1, q: "" };
      render();
    });
  });
  app?.querySelectorAll("[data-resolve-food]").forEach((b) => {
    b.addEventListener("click", () => {
      const [foodID, slot, idx, part] = b.dataset.resolveFood.split(":");
      const [cut, prep] = (b.dataset.pickVariant || "").split(":");
      resolveFoodPick(foodID, slot, idx, part === "" ? null : part, { cut, prep });
    });
  });
  app?.querySelectorAll("[data-pick-food]").forEach((b) => {
    b.addEventListener("click", () => {
      const food = state.foods.find((f) => f.id === b.dataset.pickFood);
      const target = draftTarget(b.dataset.slot, b.dataset.idx, b.dataset.part);
      if (food && target) {
        const [cut, prep] = (b.dataset.pickVariant || "").split(":");
        applyCatalogFoodToMealItem(target, food, b.dataset.pickLabel, { cut, prep });
        scheduleSaveDay();
      }
      clearNameSuggest();
      render();
    });
  });
  app?.querySelectorAll("[data-pick-set-name]").forEach((b) => {
    b.addEventListener("click", () => {
      const target = state.mealDraft[b.dataset.slot]?.items[Number(b.dataset.idx)];
      if (target) target.name = b.dataset.pickSetName;
      clearNameSuggest();
      scheduleSaveDay();
      render();
    });
  });
  app?.querySelectorAll("[data-add]").forEach((b) => {
    b.addEventListener("click", () => {
      state.mealDraft[b.dataset.add].items.push(emptyItem());
      scheduleSaveDay();
      render();
    });
  });
  app?.querySelectorAll("[data-make-dish]").forEach((b) => {
    b.addEventListener("click", () => {
      const [slot, i] = b.dataset.makeDish.split(":");
      const it = state.mealDraft[slot].items[Number(i)];
      if (it && !itemHasParts(it)) {
        const first = cloneDraftItem(it);
        first.parts = undefined;
        first.is_set = false;
        it.parts = [first];
        it.name = defaultSetName(slot);
        it.is_set = true;
        recomputeSetGrams(it);
      }
      scheduleSaveDay();
      render();
    });
  });
  app?.querySelectorAll("[data-eject-part]").forEach((b) => {
    b.addEventListener("click", () => {
      const [slot, i, j] = b.dataset.ejectPart.split(":");
      ejectPartFromSet(slot, Number(i), Number(j));
      scheduleSaveDay();
      render();
    });
  });
  app?.querySelectorAll("[data-add-part]").forEach((b) => {
    b.addEventListener("click", () => {
      const [slot, i] = b.dataset.addPart.split(":");
      const it = state.mealDraft[slot].items[Number(i)];
      if (!it.parts) it.parts = [];
      it.parts.push(emptyItem());
      scheduleSaveDay();
      render();
    });
  });
  app?.querySelectorAll("[data-into-set]").forEach((sel) => {
    sel.addEventListener("change", () => {
      const v = sel.value;
      if (!v) return;
      const [slot, i] = sel.dataset.intoSet.split(":");
      absorbIntoSet(slot, Number(v), Number(i));
      scheduleSaveDay();
      render();
    });
  });
  app?.querySelectorAll("[data-del]").forEach((b) => {
    b.addEventListener("click", () => {
      const [slot, i] = b.dataset.del.split(":");
      if (slot === "stock") {
        state.mealDraft.stock.items[0] = emptyStockSet();
        state.stockForm.set = state.mealDraft.stock.items[0];
        render();
        return;
      }
      state.mealDraft[slot].items.splice(Number(i), 1);
      if (!state.mealDraft[slot].items.length) state.mealDraft[slot].items.push(emptyItem());
      scheduleSaveDay();
      render();
    });
  });
  app?.querySelectorAll("[data-del-part]").forEach((b) => {
    b.addEventListener("click", () => {
      const [slot, i, j] = b.dataset.delPart.split(":");
      const it = state.mealDraft[slot].items[Number(i)];
      it.parts.splice(Number(j), 1);
      recomputeSetGrams(it);
      scheduleSaveDay();
      render();
    });
  });
  app?.querySelectorAll("[data-note]").forEach((inp) => {
    inp.addEventListener("change", () => {
      state.mealDraft[inp.dataset.note].note = inp.value;
      scheduleSaveDay();
    });
  });
  document.getElementById("day-bp")?.addEventListener("change", (e) => {
    state.mealDraft.bp = e.target.value;
    scheduleSaveDay();
    render();
  });
  document.getElementById("day-walking")?.addEventListener("change", (e) => {
    state.mealDraft.walking = e.target.value;
    scheduleSaveDay();
    render();
  });
  document.getElementById("delete-day")?.addEventListener("click", deleteDay);
  bindIsoInputs();

  document.getElementById("days-go")?.addEventListener("click", async () => {
    state.daysQ = document.getElementById("days-q").value;
    await loadDays();
    render();
  });
  document.getElementById("days-q")?.addEventListener("keydown", (e) => {
    if (e.key === "Enter") document.getElementById("days-go")?.click();
  });
  document.getElementById("days-range")?.addEventListener("change", async (e) => {
    state.daysRange = e.target.value || "28";
    await loadDays();
    render();
  });
  document.getElementById("weight-range")?.addEventListener("change", (e) => {
    state.weightRange = e.target.value || "28";
    render();
  });
  document.querySelectorAll("[data-days-filter]").forEach((btn) => {
    btn.addEventListener("click", () => {
      state.daysStatusFilter = btn.dataset.daysFilter || "all";
      render();
    });
  });
  const fq = document.getElementById("foods-q");
  fq?.addEventListener("input", () => {
    state.foodsQ = fq.value;
    render();
    const n = document.getElementById("foods-q");
    if (n) {
      n.focus();
      n.setSelectionRange(n.value.length, n.value.length);
    }
  });
  document.getElementById("stock-add")?.addEventListener("click", () => beginNewStockBatch());
  document.querySelectorAll("[data-edit-stock]").forEach((b) => {
    b.addEventListener("click", (e) => {
      e.stopPropagation();
      beginEditStockBatch(b.dataset.editStock);
    });
  });
  document.querySelectorAll("[data-draw-stock]").forEach((b) => {
    b.addEventListener("click", () => {
      drawStock(b.dataset.drawStock).catch((e) => toast(e.message));
    });
  });
  document.querySelectorAll("[data-stock-active]").forEach((inp) => {
    inp.addEventListener("change", () => {
      toggleStockActive(inp.dataset.stockActive, inp.checked).catch((e) => toast(e.message));
    });
  });
  document.querySelectorAll("[data-stock-use]").forEach((b) => {
    b.addEventListener("click", () => {
      adjustStock(b.dataset.stockUse, "household").catch((e) => toast(e.message));
    });
  });
  document.querySelectorAll("[data-stock-discard]").forEach((b) => {
    b.addEventListener("click", () => {
      adjustStock(b.dataset.stockDiscard, "discarded").catch((e) => toast(e.message));
    });
  });
  document.querySelectorAll("[data-open-stock]").forEach((card) => {
    card.addEventListener("click", (e) => {
      if (e.target.closest("button, label, input, a, [data-stock-actions]")) return;
      beginEditStockBatch(card.dataset.openStock);
    });
  });
  document.getElementById("foods-add")?.addEventListener("click", () => openFoodForm().catch((e) => toast(e.message)));
  document.querySelectorAll("[data-open-food]").forEach((tr) => {
    tr.addEventListener("click", () => openFoodForm(tr.dataset.openFood).catch((e) => toast(e.message)));
  });

  bindSettingsAutoSave();
  const refreshEnergyHint = () => refreshEnergyHintFromDom();
  const onSedentaryDiscountInput = () => {
    const inp = document.getElementById("set-sedentary-discount");
    const lab = document.getElementById("set-sedentary-discount-val");
    if (inp && lab) lab.textContent = `${inp.value}%`;
    refreshEnergyHint();
  };
  const onMealMarginInput = () => {
    const inp = document.getElementById("set-meal-margin");
    const lab = document.getElementById("set-meal-margin-val");
    if (inp && lab) lab.textContent = `${inp.value}%`;
  };
  ["set-sex", "set-age", "set-height"].forEach((id) => {
    document.getElementById(id)?.addEventListener("input", refreshEnergyHint);
    document.getElementById(id)?.addEventListener("change", refreshEnergyHint);
  });
  document.getElementById("set-sedentary-discount")?.addEventListener("input", onSedentaryDiscountInput);
  document.getElementById("set-sedentary-discount")?.addEventListener("change", onSedentaryDiscountInput);
  document.getElementById("set-meal-margin")?.addEventListener("input", onMealMarginInput);
  document.getElementById("set-meal-margin")?.addEventListener("change", onMealMarginInput);
  document.getElementById("settings-repair-days")?.addEventListener("click", async () => {
    const ok = await confirmModal({
      title: "Repair all days?",
      message: "<p class=\"text-sm text-base-content/60 m-0\">Re-parses meal text in the database and recomputes catalog totals. Legacy markdown day totals are restored from mddata.</p>",
      confirmLabel: "Repair",
    });
    if (!ok) return;
    try {
      const out = await api("/import/repair-days?from_md=1", { method: "POST" });
      toast(`Repaired ${out.repaired_days ?? 0} days`);
      if (state.tab === "days") await loadDays();
      if (state.tab === "today" && state.date) await loadDay();
      render();
    } catch (e) {
      toast(e.message);
    }
  });
  document.getElementById("settings-reset")?.addEventListener("click", () => {
    const cur = mergedKcalBudget(state.kcalBudget);
    api("/settings", { method: "PUT", body: JSON.stringify({
      ...DEFAULT_KCAL_BUDGET,
      sex: cur.sex || "",
      age_years: cur.age_years || 0,
      height_cm: cur.height_cm || 0,
    }) })
      .then((data) => {
        state.kcalBudget = mergedKcalBudget(data.settings);
        if (state.overview) state.overview.settings = data.settings;
        toast("Defaults restored");
        render();
      })
      .catch((e) => toast(e.message));
  });
  document.getElementById("weight-add")?.addEventListener("click", () => openWeightForm().catch((e) => toast(e.message)));
  document.querySelectorAll("[data-open-weight]").forEach((tr) => {
    tr.addEventListener("click", () => openWeightForm(tr.dataset.openWeight).catch((e) => toast(e.message)));
  });
}

function numOrUndef(v) {
  const n = Number(v);
  return v !== "" && !Number.isNaN(n) ? n : undefined;
}

function draftTarget(slot, idx, part) {
  const item = state.mealDraft[slot]?.items[Number(idx)];
  if (!item) return null;
  if (part == null || part === "") return item;
  if (!item.parts) item.parts = [];
  return item.parts[Number(part)] || null;
}

function serializeItem(it, { inSet = false } = {}) {
  const name = (it.name || "").trim();
  const parts = (it.parts || []).map((p) => serializeItem(p, { inSet: true })).filter(Boolean);
  if (!name && !parts.length) return null;
  const linkedFood = serializeItemFoodID(it);
  const hasStock = !!(it.stock_lot_id || stockLotForItem(it)?.id);
  if (inSet && name && !parts.length && ingredientMissingGrams(it) && !linkedFood && !hasStock) {
    return null;
  }
  const row = { name: name || "Dish" };
  const cnt = numOrUndef(it.count);
  if (cnt != null && cnt > 0) row.count = cnt;
  const gExplicit = numOrUndef(it.grams);
  row.grams = gExplicit != null && gExplicit > 0 ? gExplicit : 0;
  if (linkedFood) row.food_id = linkedFood;
  const lot = stockLotForItem(it);
  const editingId = state.stockForm?.id;
  if (lot?.id && lot.id !== editingId) row.stock_lot_id = lot.id;
  else if (it.stock_lot_id && it.stock_lot_id !== editingId) row.stock_lot_id = it.stock_lot_id;
  if (row.stock_lot_id) {
    delete row.parts;
  }
  const k = numOrUndef(it.kcal_per_100g);
  const n = numOrUndef(it.sodium_mg_per_100g);
  if (k != null) row.kcal_per_100g = k;
  if (n != null) row.sodium_mg_per_100g = n;
  if (it.note) row.note = it.note;
  if (parts.length) row.parts = parts;
  return row;
}

function draftItems(slot) {
  return (state.mealDraft[slot].items || []).map(serializeItem).filter(Boolean);
}

let saveDayTimer = null;
let saveDayInFlight = false;
let saveDayQueued = false;

function scheduleSaveDay() {
  if (state.tab !== "today" || !state.user || !state.token) return;
  scheduleDayNutritionCompute(draftItems, state.mealDraft, () => renderPreservingFocus());
  clearTimeout(saveDayTimer);
  saveDayTimer = setTimeout(() => saveDay({ silent: true }), 650);
}

async function saveDay(opts = {}) {
  const silent = Boolean(opts.silent);
  if (saveDayInFlight) {
    saveDayQueued = true;
    return;
  }
  if (!silent) {
    state.busy = true;
    render();
  }
  saveDayInFlight = true;
  try {
    const meals = [];
    const deleteMeals = [];
    for (const slot of SLOTS) {
      const items = draftItems(slot);
      const had = state.day?.meals?.[slot] && ((state.day.meals[slot].items || []).length || state.day.meals[slot].raw);
      if (!items.length) {
        if (had) deleteMeals.push(slot);
        continue;
      }
      meals.push({
        slot,
        replace: true,
        items,
      });
    }
    const body = {
      date: state.date,
      meals,
      delete_meals: deleteMeals,
      bp: state.mealDraft.bp,
      walking: state.mealDraft.walking,
    };
    const out = await api("/log", { method: "POST", body: JSON.stringify(body) });
    state.day = out.day || null;
    if (!silent) fillDraftFromDay(state.day);
    await refreshDayNutrition(draftItems, state.mealDraft);
    if (out.foods_ensured?.length || out.foods_created?.length) await loadFoods();
    if (out.reply) {
      window.alert(out.reply);
    }
    if (!silent) toast("Saved");
    if (!silent) {
      render();
    } else if (!document.activeElement?.matches?.("[data-field], #day-bp, #day-walking")) {
      render();
    }
  } catch (e) {
    toast(e.message);
  } finally {
    saveDayInFlight = false;
    if (saveDayQueued) {
      saveDayQueued = false;
      scheduleSaveDay();
    }
    if (!silent) {
      state.busy = false;
      render();
    }
  }
}

async function deleteDay() {
  const ok = await confirmModal({
    title: "Delete this day?",
    message: `<p class="text-sm text-base-content/60 m-0">All meals and notes for <span class="font-mono tabular-nums">${esc(state.date)}</span> will be removed. This cannot be undone.</p>`,
    confirmLabel: "Delete day",
    danger: true,
  });
  if (!ok) return;
  try {
    await api(`/days/${state.date}`, { method: "DELETE" });
    state.day = null;
    state.mealDraft = emptyDraft();
    toast("Deleted");
    render();
  } catch (e) {
    toast(e.message);
  }
}

async function openFoodForm(id) {
  const f = id ? state.foods.find((x) => x.id === id) : null;
  const editing = !!f;
  const ok = await customModal({
    title: editing ? "Edit food" : "Add food",
    boxClass: FF_MODAL_BOX,
    bodyHtml: `<div class="flex flex-col gap-2">${foodFormFieldsHtml(f || {})}</div>`,
    submitLabel: editing ? "Save" : "Add",
    onMount: (overlay) => {
      bindFoodFormPer100Preview(overlay);
      if (editing) mountFoodFormAliases(overlay, f.id);
      if (editing) {
        const actions = overlay.querySelector(".modal-action");
        if (actions && !actions.querySelector("[data-ff-delete]")) {
          const del = document.createElement("button");
          del.type = "button";
          del.className = "btn btn-outline btn-error btn-sm";
          del.dataset.ffDelete = "1";
          del.textContent = "Delete";
          actions.insertBefore(del, actions.firstChild);
          del.addEventListener("click", async () => {
            const gone = await delFood(f.id);
            if (gone) {
              overlay.remove();
              toast("Food deleted");
              render();
            }
          });
        }
      }
    },
    onSubmit: async (overlay) => {
      const body = readFoodForm(overlay);
      if (body.error) {
        toast(body.error);
        return false;
      }
      if (!body.name) {
        toast("Name required");
        return false;
      }
      try {
        let foodId = f?.id;
        if (f) {
          await api(`/foods/${f.id}`, { method: "PUT", body: JSON.stringify(body) });
        } else {
          const data = await api("/foods/batch", { method: "POST", body: JSON.stringify({ upsert: [body] }) });
          foodId = data?.foods?.[0]?.id;
        }
        const pendingAliases = readFoodFormPendingAliases(overlay);
        if (foodId && pendingAliases.length) {
          await addFoodAliases(foodId, pendingAliases);
        }
        await loadFoods();
        toast(pendingAliases.length ? "Food and aliases saved" : "Food saved");
        render();
        return true;
      } catch (e) {
        toast(e.message);
        return false;
      }
    },
  });
  if (!ok) return;
}

function foodDeleteModal(foodId) {
  const food = state.foods.find((x) => x.id === foodId);
  if (!food) return Promise.resolve(null);
  const opts = state.foods
    .filter((x) => x.id !== foodId)
    .sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: "base" }))
    .map((x) => `<option value="${esc(x.id)}">${esc(x.name)}</option>`)
    .join("");
  return new Promise((resolve) => {
    const overlay = el(`<div class="modal modal-open modal-middle" role="dialog" aria-modal="true" aria-labelledby="food-delete-title">
      <div class="modal-box max-w-md">
        <h3 class="font-bold text-lg" id="food-delete-title">Delete “${esc(food.name)}”?</h3>
        <p class="text-sm text-base-content/60 m-0">Meal lines keep their saved <span class="font-mono">food_id</span>. Pick a replacement catalog food to store aliases (including this food’s id and name) so old days still resolve.</p>
        <label class="form-control w-full mt-3">
          <span class="label-text text-xs uppercase opacity-60">Replacement</span>
          <select class="select select-bordered w-full" id="food-delete-replace">
            <option value="">No replacement</option>
            ${opts}
          </select>
        </label>
        <div class="modal-action">
          <button type="button" class="btn btn-outline btn-sm" data-fd-cancel>Cancel</button>
          <button type="button" class="btn btn-outline btn-error btn-sm" data-fd-ok>Delete</button>
        </div>
      </div>
      <div class="modal-backdrop modal-backdrop-inert" aria-hidden="true"></div>
    </div>`);
    const done = (v) => {
      overlay.remove();
      resolve(v);
    };
    overlay.querySelector("[data-fd-cancel]")?.addEventListener("click", () => done(null));
    overlay.querySelector("[data-fd-ok]")?.addEventListener("click", () => {
      done(overlay.querySelector("#food-delete-replace")?.value ?? "");
    });
    overlay.addEventListener("keydown", (e) => {
      if (e.key === "Escape") done(null);
    });
    document.body.appendChild(overlay);
    overlay.querySelector("#food-delete-replace")?.focus();
  });
}

/** @returns {Promise<boolean>} */
async function delFood(id, { skipReload = false } = {}) {
  const replaceWith = await foodDeleteModal(id);
  if (replaceWith === null) return false;
  try {
    await api(`/foods/${id}`, {
      method: "DELETE",
      body: JSON.stringify({ replace_with: replaceWith || "" }),
    });
    await loadFoods();
    if (!skipReload && state.tab === "today" && state.date) {
      await loadDay();
    }
    render();
    return true;
  } catch (e) {
    toast(e.message);
    return false;
  }
}

async function openWeightForm(id) {
  const w = id ? state.weights.find((x) => x.id === id) : null;
  const vals = await formModal({
    title: w ? "Edit weigh-in" : "Add weigh-in",
    submitLabel: w ? "Save" : "Add",
    deleteLabel: w ? "Delete" : undefined,
    onDelete: w
      ? () => delWeight(w.id).then((ok) => {
          if (ok) toast("Weigh-in deleted");
          return ok;
        })
      : undefined,
    fields: [
      {
        type: "row",
        fields: [
          { key: "date", label: "Date", value: w?.date || state.date, date: true },
          { key: "kg", label: "kg", value: w?.kg ?? "", type: "decimal", tight: true },
          { key: "rhr", label: "RHR (bpm)", value: w?.rhr_bpm ?? "", type: "numeric", tight: true, tooltip: "Resting heart rate in beats per minute" },
        ],
      },
      { key: "note", label: "Note", value: w?.note || "", grow: true },
      { key: "protocol", label: "Protocol weigh-in", value: w ? !!w.protocol : true, type: "checkbox" },
    ],
  });
  if (!vals) return;
  const date = parseISODate(vals.date);
  if (!date) {
    toast("Date must be YYYY-MM-DD");
    return;
  }
  const kg = numOrUndef(vals.kg);
  const rhr = numOrUndef(vals.rhr);
  try {
    if (w) {
      await api(`/weight/${w.id}`, {
        method: "PUT",
        body: JSON.stringify({
          date,
          kg,
          note: (vals.note || "").trim(),
          protocol: vals.protocol,
          rhr_bpm: rhr != null ? Math.round(rhr) : undefined,
        }),
      });
    } else {
      await api("/log", {
        method: "POST",
        body: JSON.stringify({
          date,
          weight: {
            kg,
            note: (vals.note || "").trim(),
            protocol: vals.protocol,
            rhr_bpm: rhr != null ? Math.round(rhr) : undefined,
          },
        }),
      });
    }
    await loadWeights();
    toast("Weigh-in saved");
    render();
  } catch (e) {
    toast(e.message);
  }
}

/** @returns {Promise<boolean>} */
async function delWeight(id) {
  const ok = await confirmModal({
    title: "Delete this weigh-in?",
    message: "<p class=\"text-sm text-base-content/60 m-0\">This cannot be undone.</p>",
    confirmLabel: "Delete",
    danger: true,
  });
  if (!ok) return false;
  try {
    await api(`/weight/${id}`, { method: "DELETE" });
    await loadWeights();
    render();
    return true;
  } catch (e) {
    toast(e.message);
    return false;
  }
}

function fillDraftFromDay(day) {
  const d = emptyDraft();
  if (day) {
    d.bp = day.bp || "";
    d.walking = day.walking || "";
    for (const slot of SLOTS) {
      const m = day.meals?.[slot];
      if (!m) continue;
      d[slot].note = "";
      const struct = m.items && m.items.length && !mealItemsLookLikeBlob(m.items);
      d[slot].items = struct ? m.items.map((raw) => {
        const c = cloneDraftItem(raw);
        if (c.stock_lot_id && c.parts?.length) {
          c.parts = undefined;
          c.is_set = false;
        }
        return c;
      }) : emptySlot().items;
      const walk = (it) => {
        linkDraftItemStock(it);
        linkDraftItemFood(it);
        if (itemHasParts(it)) {
          pruneBlankSetParts(it);
          recomputeSetGrams(it);
          (it.parts || []).forEach(walk);
        }
      };
      d[slot].items.forEach(walk);
    }
  }
  state.mealDraft = d;
}

async function loadFoods() {
  const data = await api("/foods");
  state.foods = data.foods || [];
  state.foodAliases = data.aliases || [];
}

async function loadDay({ compute = true } = {}) {
  try {
    const data = await api(`/days/${state.date}`);
    state.day = data.day;
    state.computedDay = state.day;
  } catch (e) {
    if (e.status === 404) {
      state.day = null;
      state.computedDay = null;
    } else throw e;
  }
  fillDraftFromDay(state.day);
  if (!state.day) return;
  if (compute) {
    await refreshDayNutrition(draftItems, state.mealDraft);
  } else {
    state.computedDay = state.day;
  }
}

function stockFormRoot() {
  return document.querySelector("[data-stock-batch-modal]") || document;
}

function syncStockFormFromDOM() {
  ensureStockMealDraft();
  const f = state.stockForm;
  const root = stockFormRoot();
  const set = state.mealDraft.stock.items[0];
  f.set = set;
  f.prepared_date = root.querySelector("#stock-date")?.value?.trim() || f.prepared_date;
  f.batch_grams = stockDerivedBatchGrams(set, f);
  const hidden = root.querySelector("#stock-batch");
  if (hidden) hidden.value = f.batch_grams > 0 ? String(f.batch_grams) : "";
  f.active = stockFormRoot().querySelector("#stock-active")?.checked !== false;
  f.notes = root.querySelector("#stock-notes")?.value ?? f.notes;
}

async function loadStock() {
  const data = await api("/stock");
  state.stock = data.lots || [];
}

async function saveStock() {
  syncStockFormFromDOM();
  const f = state.stockForm;
  const set = f.set;
  const batch = stockDerivedBatchGrams(set, f);
  const setName = (set?.name || "").trim();
  if (!setName || !batch || batch <= 0) {
    toast("Set name and at least one ingredient with grams required");
    return false;
  }
  const cur = f.id ? (state.stock || []).find((l) => l.id === f.id) : null;
  const oldBatch = Number(cur?.batch_grams) || 0;
  const oldLeft = numOrUndef(f.remaining_grams) ?? (Number(cur?.remaining_grams) || 0);
  const used = f.id ? Math.max(0, oldBatch - oldLeft) : 0;
  f.batch_grams = batch;
  const left = f.id ? Math.max(0, batch - used) : batch;
  const items = serializeStockParts(set);
  const body = {
    name: setName,
    prepared_date: f.prepared_date?.trim() || undefined,
    batch_grams: batch,
    remaining_grams: left,
    active: left > 0.01 && f.active !== false,
    notes: f.notes || "",
    items,
  };
  if (f.id) {
    await api(`/stock/${f.id}`, { method: "PUT", body: JSON.stringify(body) });
  } else {
    await api("/stock", { method: "POST", body: JSON.stringify(body) });
  }
  state.stockForm = emptyStockForm();
  await loadStock();
  toast("Batch saved");
  return true;
}

function askStockAction(lot, kind) {
  return new Promise((resolve) => {
    if (!lot) {
      resolve(null);
      return;
    }
    const left = Number(lot.remaining_grams) || 0;
    const titles = {
      meal: "Log my meal",
      household: "Use without meal log",
      discarded: "Discard from stock",
    };
    const overlay = el(`<div class="modal modal-open modal-middle" role="dialog" aria-modal="true">
      <div class="modal-box max-w-md">
        <h3 class="text-lg font-bold">${esc(titles[kind] || "Stock")}</h3>
        <p class="text-sm text-base-content/60 py-2">${esc(lot.name)}: <span class="font-mono tabular-nums">${fmt(left, 0)} g</span> left</p>
        <label class="form-control w-full"><span class="${FF_LABEL}">Grams (required)</span>
          <input id="sd-grams" class="${FF_INP}" type="number" inputmode="decimal" min="0.1" step="1" placeholder="e.g. 150" autofocus>
        </label>
        ${kind === "meal" ? `
        <label class="form-control w-full"><span class="${FF_LABEL}">Meal slot</span>
          <select id="sd-slot" class="${SETTINGS_SEL}">
            <option value="lunch">lunch</option>
            <option value="dinner" selected>dinner</option>
            <option value="snack">snack</option>
          </select>
        </label>
        <label class="form-control w-full"><span class="${FF_LABEL}">Date</span>
          ${isoDateInputHtml({ id: "sd-date", value: state.date || "", extraClass: FF_INP })}
        </label>` : ""}
        <div class="modal-action">
          <button type="button" class="btn btn-outline btn-sm" data-sd-cancel>Cancel</button>
          <button type="button" class="btn btn-outline btn-sm" data-sd-all title="Remove all remaining">All ${fmt(left, 0)} g</button>
          <button type="button" class="btn btn-primary btn-sm" data-sd-ok>OK</button>
        </div>
      </div>
      <div class="modal-backdrop modal-backdrop-inert" aria-hidden="true"></div>
    </div>`);
    const close = (val) => {
      overlay.remove();
      resolve(val);
    };
    const readGrams = () => {
      const raw = overlay.querySelector("#sd-grams")?.value?.trim();
      const n = numOrUndef(raw);
      if (raw === "" || n == null || n <= 0) return null;
      if (n > left + 0.01) {
        toast(`Max ${fmt(left, 0)} g left`);
        return null;
      }
      return n;
    };
    overlay.querySelector("[data-sd-cancel]")?.addEventListener("click", () => close(null));
    overlay.querySelector("[data-sd-all]")?.addEventListener("click", () => {
      const slot = kind === "meal" ? overlay.querySelector("#sd-slot")?.value : "";
      const dateIn = kind === "meal" ? overlay.querySelector("#sd-date")?.value : "";
      const date = kind === "meal" ? parseISODate(dateIn) || state.date : "";
      if (kind === "meal" && !date) {
        toast("Date required");
        return;
      }
      close({ grams: left, slot, date });
    });
    overlay.querySelector("[data-sd-ok]")?.addEventListener("click", () => {
      const grams = readGrams();
      if (grams == null) {
        toast("Enter grams");
        return;
      }
      if (kind === "meal") {
        const slot = (overlay.querySelector("#sd-slot")?.value || "").trim().toLowerCase();
        const dateIn = overlay.querySelector("#sd-date")?.value;
        const date = parseISODate(dateIn) || state.date;
        if (!SLOTS.includes(slot)) {
          toast("Invalid slot");
          return;
        }
        if (!date) {
          toast("Date required");
          return;
        }
        close({ grams, slot, date });
        return;
      }
      close({ grams });
    });
    overlay.addEventListener("keydown", (e) => {
      if (e.key === "Escape") close(null);
    });
    document.body.appendChild(overlay);
    bindIsoInputs(overlay);
    overlay.querySelector("#sd-grams")?.focus();
  });
}

function confirmStockDelete(lot) {
  if (!lot) return Promise.resolve(false);
  const left = Number(lot.remaining_grams) || 0;
  const batch = Number(lot.batch_grams) || 0;
  const qty = batch > 0 ? `${fmt(left, 0)}/${fmt(batch, 0)} g` : `${fmt(left, 0)} g`;
  return confirmModal({
    title: "Delete this batch?",
    message: `<p class="text-sm text-base-content/60 m-0">${esc(lot.name)} <span class="font-mono tabular-nums">(${esc(qty)})</span></p><p class="text-sm text-base-content/60 mt-2 m-0">Removes the lot and its usage history. This cannot be undone.</p>`,
    confirmLabel: "Delete",
    danger: true,
  });
}

async function drawStock(id) {
  const lot = state.stock.find((x) => x.id === id);
  const ans = await askStockAction(lot, "meal");
  if (!ans) return;
  await api(`/stock/${id}/draw`, {
    method: "POST",
    body: JSON.stringify({ date: ans.date, slot: ans.slot, grams: ans.grams }),
  });
  await loadStock();
  if (state.date === ans.date) await loadDay();
  toast(`Logged ${ans.grams} g to ${ans.slot} on ${ans.date}`);
  render();
}

async function deleteStock(id) {
  const lot = state.stock.find((x) => x.id === id);
  const ok = await confirmStockDelete(lot);
  if (!ok) return false;
  try {
    await api(`/stock/${id}`, { method: "DELETE" });
  } catch (e) {
    toast(e.message);
    return false;
  }
  if (state.stockForm?.id === id) {
    state.stockForm = emptyStockForm();
    delete state.mealDraft?.stock;
  }
  await loadStock();
  toast("Batch deleted");
  render();
  return true;
}

async function toggleStockActive(id, active) {
  await api(`/stock/${id}/active`, { method: "PATCH", body: JSON.stringify({ active }) });
  await loadStock();
  toast(active ? "Marked active" : "Marked finished");
  render();
}

async function adjustStock(id, mode) {
  const lot = state.stock.find((x) => x.id === id);
  const ans = await askStockAction(lot, mode === "discarded" ? "discarded" : "household");
  if (!ans) return;
  await api(`/stock/${id}/adjust`, { method: "POST", body: JSON.stringify({ grams: ans.grams, mode }) });
  await loadStock();
  toast(mode === "discarded" ? `Discarded ${ans.grams} g` : `Used ${ans.grams} g (no meal log)`);
  render();
}

function indexDayGapItems(items) {
  const byDate = new Map();
  for (const g of items || []) {
    if (!g?.date) continue;
    if (!byDate.has(g.date)) byDate.set(g.date, []);
    byDate.get(g.date).push(g);
  }
  state.dayGapsByDate = byDate;
}

async function fetchDayGapsForLoadedDays() {
  const dates = (state.days || []).map((d) => d.date).filter(Boolean);
  if (!dates.length) {
    state.dayGapItems = [];
    indexDayGapItems([]);
    return;
  }
  const from = dates.reduce((a, b) => (a < b ? a : b));
  const to = dates.reduce((a, b) => (a > b ? a : b));
  try {
    const gaps = await api(`/gaps?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}&limit=500`);
    state.dayGapItems = gaps.items || [];
  } catch {
    state.dayGapItems = [];
  }
  indexDayGapItems(state.dayGapItems);
}

async function loadDays({ fetchGaps = true } = {}) {
  const q = state.daysQ ? `&q=${encodeURIComponent(state.daysQ)}` : "";
  const rangeQ = daysRangeQueryString(state.daysRange, resolveTodayDate(state));
  const loads = [api(`/days?limit=500${q}${rangeQ}`)];
  if (!state.weights?.length) {
    loads.push(loadWeights().catch(() => {}));
  }
  const [data] = await Promise.all(loads);
  state.days = data.days || [];
  if (fetchGaps) await fetchDayGapsForLoadedDays();
  else indexDayGapItems(state.dayGapItems);
}

async function loadWeights() {
  const data = await api("/weight");
  state.weights = data.weights || [];
}

async function loadTodayTabData() {
  const jobs = [];
  if (!state.foods?.length) jobs.push(loadFoods());
  if (!state.stock?.length) {
    jobs.push(loadStock().catch(() => {}));
  }
  if (!state.weights?.length) {
    jobs.push(loadWeights().catch(() => {}));
  }
  if (jobs.length) await Promise.all(jobs);
  await loadDay({ compute: false });
}

async function loadTabData() {
  if (state.tab === "days") {
    await loadDays();
    return;
  }
  if (state.tab === "settings" && !state.kcalBudget) {
    try {
      const data = await api("/settings");
      state.kcalBudget = mergedKcalBudget(data.settings);
    } catch {
      state.kcalBudget = mergedKcalBudget(null);
    }
    return;
  }
  if (state.tab === "weight") {
    await loadWeights();
    return;
  }
  if (state.tab === "foods") {
    await loadFoods();
    return;
  }
  if (state.tab === "stock") {
    await loadStock();
    return;
  }
  if (state.tab === "today") await loadTodayTabData();
}

function syncTodayInUrl() {
  if (state.tab === "today") pushAppRoute(state, { replace: true });
}

export async function openDayFromDaysList(dateRaw) {
  const date = parseISODate(dateRaw);
  if (!date) {
    toast("Invalid date");
    return;
  }
  await goToTab("today", { date });
}

export async function goToTab(tab, { date, replace = false } = {}) {
  state.tab = tab;
  if (tab === "today") {
    state.date = date || resolveTodayDate(state);
    state.foodPick = null;
    state.suggest = { slot: "", i: -1, p: -1, q: "" };
  } else if (date) {
    state.date = date;
  }
  state.tabLoading = true;
  pushAppRoute(state, { replace });
  render();
  try {
    await loadTabData();
  } catch (e) {
    toast(e.message);
  }
  state.tabLoading = false;
  render();
}

export async function onHistoryNav() {
  if (!state.user || !state.token) return;
  applyRouteToState(state);
  try {
    await loadTabData();
  } catch (e) {
    toast(e.message);
  }
  render();
}

async function boot() {
  const [ov] = await Promise.all([
    api("/overview?days=14"),
    loadFoods(),
    loadStock(),
    loadWeights(),
  ]);
  state.overview = ov;
  applyKcalBudgetFromOverview(state.overview);
  applyRouteToState(state);
  if (!state.date) state.date = resolveTodayDate(state);
  await loadTabData();
  pushAppRoute(state, { replace: true });
  render();
}

export function bootIfAuthed() {
  loadAuth();
  if (state.user && state.token) {
    boot().catch((e) => {
      state.user = "";
      state.token = "";
      state.toast = e.message;
      render();
    });
  } else {
    render();
  }
}
