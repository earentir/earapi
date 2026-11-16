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
  legacyDayForDiff,
  mealSlotRaw,
} from "./nutrition-client.js";
import { parseSlotLegacyFromRaw, slotLegacyTooltip } from "./legacy-totals.js";
import { legacyDiffBandClass } from "./legacy-diff-bands.js";
import { themeSelectHtml, saveTheme } from "./theme.js";
import { state, emptyItem, emptySlot, emptyDraft, emptyStockSet, emptyStockForm } from "./state.js";
import {
  fmt, el, pad2, isoDate, parseISODate, shiftISODate, parseTime24, dateField,
  bindIsoInputs, confirmModal, formModal, customModal, weekdayNameForISODate,
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
  mealLineFoodSuggestions, foodPickCandidates,
} from "./fuzzy.js";
import { matchStockLotByName } from "./stock-match.js";
import {
  DEFAULT_KCAL_BUDGET, mergedKcalBudget, dayKcalBarParts, dayIntakeBand, dayBandCounts,
} from "./kcal-budget.js";
import { dataBackupBarHtml, bindDataBackupHandlers } from "./data-backup.js";
import { weightChartSectionHtml, mountWeightChart } from "./weight-chart.js";
import { daysKcalChartSectionHtml, mountDaysKcalChart } from "./days-kcal-chart.js";
import { daysRangeQueryString, periodRangeSelectHtml, recordsInRange } from "./days-range.js";
import { energyFromInputs, estimatedMassDeltaGrams, kgOnOrBefore } from "./energy.js";

function defaultSetName(slot) {
  const s = (slot || "").trim().toLowerCase();
  if (!SLOTS.includes(s)) return "";
  return s.charAt(0).toUpperCase() + s.slice(1);
}

function loginView() {
  const box = el(`<div class="login card">
    <h1>Health tracker</h1>
    <p class="muted">Sign in with the same username and token as the API (<span class="mono">X-User</span> / <span class="mono">X-API-Key</span>).</p>
    <label class="field"><span>Username</span><input id="login-user" autocomplete="username"></label>
    <label class="field"><span>Token</span><input id="login-token" type="password" autocomplete="current-password"></label>
    <button class="btn primary" id="login-go">Open tracker</button>
    <p id="login-err" class="warn"></p>
  </div>`);
  const userEl = box.querySelector("#login-user");
  const tokenEl = box.querySelector("#login-token");
  const errEl = box.querySelector("#login-err");
  const submitLogin = async () => {
    state.user = userEl.value.trim();
    state.token = tokenEl.value.trim();
    errEl.textContent = "";
    try {
      await api("/overview?days=1");
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
  page.append(topbar(), content(), state.toast ? el(`<div class="toast">${esc(state.toast)}</div>`) : "");
  root.replaceChildren(page);
  bind();
}

function topbar() {
  const tabs = NAV_TABS.map((t) => {
    const href = t.id === "today" ? pathForTab("today") : pathForTab(t.id);
    const active = state.tab === t.id ? " tab-active" : "";
    return `<a role="tab" class="tab${active}" href="${esc(href)}" data-nav-tab="${t.id}">${esc(t.label)}</a>`;
  }).join("");
  return el(`<header class="navbar health-navbar topbar bg-base-200/95 border-b border-base-content/10 sticky top-0 z-20 backdrop-blur">
    <div class="health-navbar-inner">
      <div class="navbar-start gap-2 shrink-0">
        <a class="brand font-bold text-lg text-primary" href="${esc(pathForTab("today"))}">health</a>
        <span class="brand-sub text-sm opacity-60 hidden sm:inline">${esc(state.user)}</span>
      </div>
      <div class="navbar-center flex-1 min-w-0 justify-center px-1">
        <div role="tablist" class="tabs tabs-boxed tabs-sm health-tabs bg-base-300/40 p-0.5 max-w-full overflow-x-auto flex-nowrap">
          ${tabs}
        </div>
      </div>
      <div class="navbar-end health-navbar-actions shrink-0 items-center">
        ${themeSelectHtml(document.documentElement.dataset.theme || "dark")}
        <button type="button" class="btn btn-ghost btn-sm" id="logout">Log out</button>
      </div>
    </div>
  </header>`);
}

function content() {
  const wrap = el(`<main class="wrap"></main>`);
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

/** Sum legacy margin totals from meal slots that still have markdown raw (partial day). */
function slotLegacyTotalsAggregated() {
  let kcal = 0;
  let sodium_mg = 0;
  let slotsWithLegacy = 0;
  for (const slot of SLOTS) {
    const leg = parseSlotLegacyFromRaw(mealSlotRaw(slot));
    if (leg.kcal <= 0 && leg.sodium_mg <= 0) continue;
    kcal += leg.kcal;
    sodium_mg += leg.sodium_mg;
    slotsWithLegacy += 1;
  }
  return { kcal, sodium_mg, slotsWithLegacy };
}

function catalogTotalsForSlotsWithLegacy() {
  const day = nutritionDay();
  let kcal = 0;
  let sodium_mg = 0;
  for (const slot of SLOTS) {
    const leg = parseSlotLegacyFromRaw(mealSlotRaw(slot));
    if (leg.kcal <= 0 && leg.sodium_mg <= 0) continue;
    const m = day?.meals?.[slot];
    kcal += Number(m?.kcal) || 0;
    sodium_mg += Number(m?.sodium_mg) || 0;
  }
  return { kcal, sodium_mg };
}

function dayLegacyComparePair(field) {
  const isNa = field === "sodium";
  const agg = slotLegacyTotalsAggregated();
  if (agg.slotsWithLegacy > 0) {
    const cur = catalogTotalsForSlotsWithLegacy();
    return {
      partial: true,
      leg: isNa ? agg.sodium_mg : agg.kcal,
      cur: isNa ? cur.sodium_mg : cur.kcal,
    };
  }
  const leg = legacyDayForDiff()?.legacy_totals;
  const legVal = isNa ? Number(leg?.sodium_mg) : Number(leg?.kcal);
  const totals = dayTotalsView();
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
  const bandClass = ` day-legacy-diff-dot--${b}`;
  return `<span class="day-legacy-hover day-legacy-diff-dot-wrap" tabindex="0" aria-label="${esc(tip)}">
    <span class="day-legacy-diff-dot${bandClass}" aria-hidden="true"></span>
    <span class="day-legacy-hover-tip" role="tooltip">${esc(tip)}</span>
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
  return legacyDiffBandClass(Number(cur) - legVal, state.kcalBudget).replace("meal-band--", "");
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
  const band = legacyDiffBandClass(cur - legVal, state.kcalBudget).replace("meal-band--", "");
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
  });
  if (!est) return "";
  const grams = estimatedMassDeltaGrams(kcal, est.sedentary);
  if (grams == null) return "";
  const signed = grams > 0 ? `+${grams}` : String(grams);
  const kind = grams > 0 ? "gain" : grams < 0 ? "loss" : "flat";
  const label = grams > 0 ? "estimated gain" : grams < 0 ? "estimated loss" : "estimated even";
  return ` <span class="mono day-card-mass-delta day-card-mass-delta--${kind}" title="${esc(label)} vs sedentary ${fmt(est.sedentary, 0)} kcal">(${esc(signed)} g)</span>`;
}

function dayHeaderTotalMarkup(d, totals) {
  const kcal = Number(totals?.kcal) || 0;
  const sod = Number(totals?.sodium_mg) || 0;
  const intakeBand = mealBandClass(dayIntakeBand(d, state.kcalBudget));
  const naIntakeBand = mealBandClass(dayDisplay(d).sodium_intake_band);
  const kDot = legacyDiffDot(d, "kcal");
  const naDot = legacyDiffDot(d, "sodium");
  const mass = dayMassDeltaMarkup(d, kcal);
  return `<span class="mono day-card-head-totals"><span class="day-card-head-energy"><span class="meal-band-kcal ${intakeBand}">${fmt(kcal, 0)}</span><span class="day-card-head-energy-unit"> kcal</span>${mass}${kDot}</span>, <span class="day-card-head-energy"><span class="meal-band-na ${naIntakeBand}">${fmt(sod, 0)}</span><span class="day-card-head-energy-unit"> mg Na</span>${naDot}</span></span>`;
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
    kcal: `<span class="stat-value mono stat-value--energy"><span class="meal-band-kcal ${intakeBand}">${fmt(kcal, 0)}</span> <span class="stat-unit">kcal</span>${mass}${kDot}</span>`,
    sodium: `<span class="stat-value mono stat-value--energy"><span class="meal-band-na ${naIntakeBand}">${fmt(sod, 0)}</span> <span class="stat-unit">mg</span>${naDot}</span>`,
  };
}

function renderDayKcalBar(_kcal, d) {
  const band = dayIntakeBand(d, state.kcalBudget);
  const parts = dayKcalBarParts(band);
  const tip = parts.title ? ` title="${esc(parts.title)}"` : "";
  const bar = parts.barClass
    ? `<span class="day-kcal-bar day-kcal-bar--${parts.barClass}"${tip} aria-hidden="true"></span>`
    : "";
  return { hostClass: parts.hostClass, bar };
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
      ? `<div class="day-slot-metrics">
          <span class="day-slot-metric-line">
            <span class="day-slot-num day-slot-kcal mono">${fmt(t.kcal, 0)}</span>
            <span class="day-slot-unit stat-unit">kcal</span>
            ${slotLegacyDiffDot(slot, "kcal")}
          </span>
          <span class="day-slot-metric-line">
            <span class="day-slot-num day-slot-na mono">${fmt(t.sodium_mg, 0)}</span>
            <span class="day-slot-unit stat-unit">mg</span>
            ${slotLegacyDiffDot(slot, "sodium")}
          </span>
        </div>`
      : `<span class="day-slot-metrics-empty mono muted">${EMPTY_CELL}</span>`;
    return `<div class="day-slot-stat">
      <span class="day-slot-name">${esc(label)}</span>
      ${metricsHtml}
    </div>`;
  }).join("");
  const status = [];
  if ((d.bp || "").trim()) status.push(`BP ${(d.bp || "").trim()}`);
  if ((d.walking || "").trim()) status.push((d.walking || "").trim());
  const statusRow = status.length
    ? `<div class="day-stat day-stat-wide">
        <span class="stat-label">Status</span>
        <span class="stat-value">${esc(status.join(", "))}</span>
      </div>`
    : "";
  const weightRow = weight
    ? `<div class="day-stat">
        <span class="stat-label">Weight</span>
        <span class="stat-value mono">${fmt(weight.kg, 1)} <span class="stat-unit">kg</span></span>
      </div>`
    : "";
  const panelTotals = dayPanelEnergyMarkup(totals, nutritionDay());
  const headEnergySodium = `<div class="day-panel-head-stats" aria-label="Day totals">
      <div class="day-stat">
        <span class="stat-label">Energy</span>
        ${panelTotals.kcal}
      </div>
      <div class="day-stat">
        <span class="stat-label">Sodium</span>
        ${panelTotals.sodium}
      </div>
    </div>`;
  const bodyStats = statusRow ? `<div class="day-stats">${statusRow}</div>` : "";
  const weightUnderTitle = weightRow ? `<div class="day-panel-weight">${weightRow}</div>` : "";
  const kbar = renderDayKcalBar(totals.kcal, nutritionDay());
  return `<section class="card day-panel${kbar.hostClass}">
    ${kbar.bar}
    <div class="slot-head day-panel-head">
      <div class="day-panel-head-left">
        <h2 class="day-panel-title" style="margin:0">${dayStatusDotMarkup(daySt)}<span class="day-panel-title-text"><span class="mono">${esc(state.date)}</span> ${esc(weekdayNameForISODate(state.date))}</span></h2>
        ${weightUnderTitle}
      </div>
      ${headEnergySodium}
    </div>
    ${bodyStats}
    <div class="day-slot-stats">${slotRows}</div>
    <div class="day-panel-actions">
      ${dateField("day-date", state.date, { label: false })}
      <button type="button" class="btn danger" id="delete-day">Delete day</button>
    </div>
  </section>`;
}

function todayView() {
  const d = state.mealDraft;
  const box = el(`<div>
    ${dataBackupBarHtml("days")}
    ${daySummaryPanel()}
    <div class="grid">
      ${SLOTS.map((slot) => slotCard(slot, d[slot])).join("")}
      <section class="card">
        <h2>Notes</h2>
        <div class="row">
          <label class="field"><span>Blood pressure</span><input id="day-bp" value="${esc(d.bp)}" placeholder="128/82"></label>
          <label class="field"><span>Walking</span><input id="day-walking" value="${esc(d.walking)}" placeholder="park 40min"></label>
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
  if (id && state.foods.some((f) => f.id === id)) return id;
  const linked = foodForItem(it);
  return linked?.id || id || undefined;
}

function stockLotForItem(it) {
  if (!it) return null;
  const lots = state.stock || [];
  if (it.stock_lot_id) {
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
  if (it?.food_id && state.foods.some((f) => f.id === it.food_id)) return true;
  return !!bestFoodByAlias(state.foodAliases, state.foods, it?.name);
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
    return `<div class="food-pick-list food-pick-list-empty muted" role="status">No catalog matches. Add the food under Foods, or set per 100 g in the kcal / Na columns.</div>`;
  }
  return `<div class="food-pick-list" role="listbox">${opts.map(({ f, label }) =>
    `<button type="button" data-resolve-food="${esc(f.id)}:${esc(slot)}:${i}${partIdx == null ? "" : `:${partIdx}`}" title="Save alias for this line name">
      ${esc(label)} <span class="muted">${fmt(f.kcal_per_100g, 0)} kcal/100g</span>
    </button>`
  ).join("")}</div>`;
}

function refreshStockModalIfOpen() {
  const overlay = document.querySelector(".stock-batch-modal");
  if (overlay) refreshStockModalBody(overlay);
}

async function resolveFoodPick(foodID, slot, idx, part) {
  const target = draftTarget(slot, idx, part);
  if (!target) return;
  const alias = (target.name || "").trim();
  if (!alias) {
    toast("Name required for alias");
    return;
  }
  target.food_id = foodID;
  target.kcal_per_100g = "";
  target.sodium_mg_per_100g = "";
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
  return `<section class="card" data-slot="${slot}">
    <div class="slot-head">
      <h2 style="text-transform:capitalize;margin:0">${slot}</h2>
      <span class="muted mono">${summary}</span>
    </div>
    ${hasItems ? mealItemsGridHead() : ""}
    ${rows}
    <div class="row">
      <button class="btn" data-add="${slot}">Add food</button>
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
  const overlay = document.querySelector(".stock-batch-modal");
  const cap = captureFieldFocus(overlay || undefined);
  if (overlay) {
    refreshStockModalIfOpen();
    restoreFieldFocus(cap, overlay);
    return;
  }
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
  stockModalInputTimer = setTimeout(() => {
    const cap = captureFieldFocus(overlay);
    refreshStockModalBody(overlay);
    restoreFieldFocus(cap, overlay);
  }, 80);
}

let nameSuggestDismissBound = false;
function refreshAfterNameSuggestDismiss() {
  if (document.querySelector(".stock-batch-modal")) refreshStockModalIfOpen();
  else renderPreservingFocus();
}

function ensureNameSuggestDismiss() {
  if (nameSuggestDismissBound) return;
  nameSuggestDismissBound = true;
  document.addEventListener("mousedown", (e) => {
    if (e.target.closest?.(".suggest") || e.target.closest?.(".food-pick-list")) return;
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
    return `<div class="suggest-list" role="listbox">${names.map((name) =>
      `<button type="button" data-pick-set-name="${esc(name)}" data-slot="${esc(slot)}" data-idx="${i}">${esc(name)}</button>`
    ).join("")}</div>`;
  }
  const hits = mealLineFoodSuggestions(state.foods, state.foodAliases, query, 10);
  if (!hits.length) return "";
  const partAttr = partIdx == null ? "" : ` data-part="${partIdx}"`;
  return `<div class="suggest-list" role="listbox">${hits.map(({ f, label }) =>
    `<button type="button" data-pick-food="${esc(f.id)}" data-pick-label="${esc(label)}" data-slot="${esc(slot)}" data-idx="${i}"${partAttr}>${esc(label)} <span class="muted">${fmt(f.kcal_per_100g, 0)} kcal/100g</span></button>`
  ).join("")}</div>`;
}

function itemKcalCell(slot, idx, partIdx) {
  const it = computedMealItem(slot, idx, partIdx);
  const k = Number(it?.kcal);
  return Number.isFinite(k) && k > 0 ? fmt(k, 0) : EMPTY_CELL;
}

function itemNaCell(slot, idx, partIdx) {
  const it = computedMealItem(slot, idx, partIdx);
  const n = Number(it?.sodium_mg);
  return Number.isFinite(n) && n > 0 ? fmt(n, 0) : EMPTY_CELL;
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
  const gap = itemGapFor(slot, idx, partIdx);
  if (gap) {
    const msg = (gap.note || "").trim() || "Missing catalog data to compute this line";
    return { msg };
  }
  const it = partIdx == null
    ? state.mealDraft?.[slot]?.items?.[idx]
    : state.mealDraft?.[slot]?.items?.[idx]?.parts?.[partIdx];
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
  const cls = extraClass ? `btn icon-btn ${extraClass}` : "btn icon-btn";
  return `<button type="button" class="${cls}" title="${esc(title)}" aria-label="${esc(title)}" ${attrs}>${glyph}</button>`;
}

function mealItemsGridHead() {
  return `<div class="item-row item-row-head muted" aria-hidden="true">
    <span class="icon-spacer"></span>
    <span>Name</span>
    <span class="num">×</span>
    <span class="num">g</span>
    <span class="num">kcal</span>
    <span class="num">Na mg</span>
    <span class="icon-spacer"></span>
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
    ? `<span class="cell-num muted">${EMPTY_CELL}</span>`
    : `<input class="cell-count" data-field="count" data-slot="${esc(slot)}" data-idx="${i}"${partAttr} value="${esc(it.count)}" placeholder="×" inputmode="decimal" title="How many units (e.g. 4 wafers)">`;
  const setGrams = Number(computedSet?.grams) || Number(it.grams) || 0;
  const gramsCell = isSetHeader
    ? `<span class="cell-num mono cell-sum" title="${setPartCount} ingredients, ${fmt(setGrams, 0)} g total">${fmt(setGrams, 0)} <span class="muted">g, ${setPartCount} items</span></span>`
    : `<input class="cell-grams" data-field="grams" data-slot="${esc(slot)}" data-idx="${i}"${partAttr} value="${esc(it.grams)}" placeholder="${gramsPh}" inputmode="decimal" title="grams total (or leave blank with × count)">`;
  let lead;
  if (isPart) {
    const ejectTitle = nutritionIssue ? nutritionIssue.msg : "Remove from set";
    lead = rowIconBtn(`data-eject-part="${esc(slot)}:${i}:${partIdx}"`, ejectTitle, "↩", nutritionIssue ? "icon-btn--warn" : "");
  } else if (!itemHasParts(it)) {
    const leadTitle = nutritionIssue ? nutritionIssue.msg : "Convert to set";
    lead = rowIconBtn(`data-make-dish="${esc(slot)}:${i}"`, leadTitle, "⊞", nutritionIssue ? "icon-btn--warn" : "");
  } else {
    lead = `<span class="icon-spacer" aria-hidden="true"></span>`;
  }
  const addPartBtn = !isPart && itemHasParts(it)
    ? rowIconBtn(`data-add-part="${esc(slot)}:${i}"`, "New ingredient", "+")
    : "";
  const addToSet = !isPart && !itemHasParts(it) ? moveIntoSetControl(slot, i) : "";
  const pickOpts = foodPickOptions(it);
  const canPick = showFoodPickUi(it);
  const pickBtn = canPick
    ? `<button type="button" class="per100-tag row-vert-label pick-tag" data-food-pick="${esc(slot)}:${i}${partIdx == null ? "" : `:${partIdx}`}" title="${esc(pickOpts.length ? `${pickOpts.length} catalog match${pickOpts.length === 1 ? "" : "es"}: pick one` : "Pick food from catalog")}" aria-label="Pick food from catalog">pick</button>`
    : "";
  const stockLot = stockLotForItem(it);
  const stockHint = stockLot && !needPer100g
    ? `<span class="per100-tag row-vert-label stock-tag" title="${esc(stockLot.name)}">stock</span>`
    : "";
  const nameTitle = needPer100g ? "Not in Foods or Stock: set per 100 g in the kcal and Na columns" : "";
  const nameInput = `<div class="suggest name-field">
        <span class="name-input-row">${stockHint}${pickBtn}<input data-field="name" data-slot="${esc(slot)}" data-idx="${i}"${partAttr} value="${esc(it.name)}" placeholder="${namePh}" autocomplete="off"${nameTitle ? ` title="${esc(nameTitle)}"` : ""}></span>
        ${foodPickList(slot, i, partIdx, it, pickOpts)}
        ${suggestList(slot, i, partIdx, it.name, { isSetHeader })}
      </div>`;
  const kcalCell = needPer100g
    ? `<input class="cell-grams cell-per100g" data-field="kcal_per_100g" data-slot="${esc(slot)}" data-idx="${i}"${partAttr} value="${esc(it.kcal_per_100g)}" placeholder="kcal/100g" inputmode="decimal" title="kcal per 100 g (not in catalog)">`
    : "";
  const kcalDisplay = itemKcalCell(slot, i, partIdx);
  const naDisplay = itemNaCell(slot, i, partIdx);
  const kcalCellOut = needPer100g
    ? kcalCell
    : `<span class="cell-num mono">${kcalDisplay}</span>`;
  const naCell = needPer100g
    ? `<input class="cell-grams cell-per100g" data-field="sodium_mg_per_100g" data-slot="${esc(slot)}" data-idx="${i}"${partAttr} value="${esc(it.sodium_mg_per_100g)}" placeholder="Na/100g" inputmode="decimal" title="sodium mg per 100 g (not in catalog)">`
    : `<span class="cell-num mono">${naDisplay}</span>`;
  const nameCell = addPartBtn
    ? `<span class="name-with-actions">${nameInput}${addPartBtn}</span>`
    : nameInput;
  const rowCls = [isPart ? "part-row" : "", nutritionIssue ? "item-row--needs-data" : ""].filter(Boolean).join(" ");
  return `<div class="item-row ${rowCls}"${nutritionIssue ? ` title="${esc(nutritionIssue.msg)}"` : ""}>
    ${lead}
    <div class="cell-name cell-name-split">
      ${nameCell}
      ${addToSet}
    </div>
    ${countCell}
    ${gramsCell}
    ${kcalCellOut}
    ${naCell}
    ${isPart
      ? `<button type="button" class="btn icon-btn" title="Delete ingredient" data-del-part="${esc(slot)}:${i}:${partIdx}">×</button>`
      : `<button type="button" class="btn icon-btn" title="Delete" data-del="${esc(slot)}:${i}">×</button>`}
  </div>`;
}

function moveIntoSetControl(slot, i) {
  const sets = setsInSlot(slot, i);
  if (!sets.length) return "";
  const opts = sets.map(({ it, j }) =>
    `<option value="${j}">${esc((it.name || "Set").trim() || "Set")}</option>`
  ).join("");
  return `<select class="add-to-set" data-into-set="${esc(slot)}:${i}" title="Add to set" aria-label="Add to set">
    <option value="">→ set</option>${opts}
  </select>`;
}

function isStockMealLine(it) {
  return !!(it?.stock_lot_id || stockLotForItem(it));
}

function itemBlock(slot, i, it) {
  const dish = itemHasParts(it) && !isStockMealLine(it);
  const parts = (it.parts || []).map((p, j) => nameGramsRow(slot, i, p, j)).join("");
  return `<div class="item-block">
    ${nameGramsRow(slot, i, it, null)}
    ${dish ? `<div class="parts">${parts}</div>` : ""}
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
    ? `<div class="meal-line-totals mono muted">${fmt(m.kcal, 0)} kcal, ${fmt(m.sodium_mg, 0)} mg</div>`
    : "";
  return `<div class="meal-line"><span class="slot-lab">${slot}</span><div class="meal-line-main"><div class="meal-line-text">${esc(text)}</div>${totRow}</div></div>`;
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
    const chip = swatch
      ? `<span class="kcal-band-swatch kcal-band-swatch--${swatch}" aria-hidden="true"></span> `
      : "";
    const n = id === "all" ? c.all : c[id] || 0;
    return `<button type="button" class="btn btn-sm days-filter-pill${active}" data-days-filter="${id}" aria-pressed="${cur === id}">${chip}${label}<span class="days-filter-count mono">${n}</span></button>`;
  }).join("");
  return `<div class="days-filter-bar" role="group" aria-label="Filter by daily kcal band">${pills}</div>`;
}

function daysListCountLine(visibleCount, totalCount, bandFilter) {
  const total = Number(totalCount) || 0;
  const visible = Number(visibleCount) || 0;
  const filtered = bandFilter && bandFilter !== "all";
  const dayWord = total === 1 ? "day" : "days";
  if (!filtered) {
    return `<p class="days-list-count muted" role="status"><span class="mono">${total}</span> ${dayWord} in range</p>`;
  }
  return `<p class="days-list-count muted" role="status"><span class="mono">${visible}</span> of <span class="mono">${total}</span> ${dayWord} in range</p>`;
}

function daysFiltersToolbar(counts) {
  return `<div class="days-filters-toolbar">${daysRangeSelectHtml()}${daysStatusFilterBar(counts)}</div>`;
}

function daysView() {
  if (state.tabLoading && !(state.days || []).length) {
    return el(`<div class="days-loading muted" role="status">Loading days…</div>`);
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
    return `<article class="day-card clickable${kbar.hostClass}" data-open-day="${esc(d.date)}" role="button" tabindex="0">
      ${kbar.bar}
      <div class="day-card-head">
        <span class="mono date-iso">${esc(d.date)}</span>
        ${totalsLine}
      </div>
      ${meals || `<p class="muted">No meal text</p>`}
      ${meta ? `<p class="muted">${meta}</p>` : ""}
      <span class="day-card-dot-wrap">${dayCardListDotMarkup(st)}</span>
    </article>`;
  }).join("");
  const root = el(`<div>
    ${dataBackupBarHtml("days")}
    ${daysKcalChartSectionHtml()}
    <div class="row searchbar">
      <label class="field grow"><span>Search meals</span><input id="days-q" value="${esc(state.daysQ)}" placeholder="moussaka, pasta..."></label>
      <button class="btn primary" id="days-go">Search</button>
    </div>
    ${daysFiltersToolbar(bandCounts)}
    ${daysListCountLine(visibleDays.length, periodDays.length, state.daysStatusFilter)}
    <div class="day-list">${
      cards ||
      `<p class="muted">${
        state.daysStatusFilter && state.daysStatusFilter !== "all" ? "No days match this filter" : "No days"
      }</p>`
    }</div>
  </div>`);
  mountDaysKcalChart(
    root.querySelector("[data-days-kcal-chart]"),
    visibleDays,
    state.kcalBudget,
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

function partNutrientPreview(p) {
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
  return `<div class="item-row part-row stock-readonly-row">
    <span class="icon-spacer"></span>
    <span class="cell-name">${esc(p.name)}</span>
    <span class="cell-num mono">${g ? fmt(g, 0) : "-"}</span>
    <span class="cell-num mono">${kcal != null ? fmt(kcal, 0) : "-"}</span>
    <span class="cell-num mono">${na != null ? fmt(na, 0) : "-"}</span>
    <span class="icon-spacer"></span>
  </div>`;
}

function stockLotIngredientsBlock(lot) {
  const parts = (lot.items || []).filter((p) => (p.name || "").trim());
  if (!parts.length) return "";
  const stats = setPartStats({ parts });
  return `<div class="stock-lot-set" style="margin-top:0.45rem">
    ${mealItemsGridHead()}
    <div class="item-block">
      <div class="item-row">
        <span class="icon-spacer"></span>
        <span class="cell-name muted">Ingredients (full batch)</span>
        <span class="cell-num mono cell-sum" title="ingredient rows / recipe total g">${stats.count} ${fmt(stats.grams, 0)}</span>
        <span class="cell-num mono">-</span>
        <span class="cell-num mono">-</span>
        <span class="icon-spacer"></span>
      </div>
      <div class="parts">${parts.map(stockReadonlyPartRow).join("")}</div>
    </div>
  </div>`;
}

function stockUsageRows(lot) {
  const rows = lot.usage || lot.Usage || [];
  if (!rows.length) {
    return `<p class="muted" style="font-size:0.78rem;margin:0.35rem 0 0">No usage logged yet. <strong>left</strong> = batch minus lines below.</p>`;
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
    const note = m.note ? ` <span class="muted">(${esc(m.note)})</span>` : "";
    const g = m.grams < 0 ? `put back ${fmt(-m.grams, 0)} g` : `-${fmt(m.grams, 0)} g`;
    return `<tr><td class="mono">${what}${note}</td><td class="mono">${g}</td><td class="muted">${when}</td></tr>`;
  });
  return `<div class="stock-usage" style="margin-top:0.45rem">
    <p class="muted" style="font-size:0.78rem;margin:0 0 0.25rem"><strong>Usage</strong> (batch minus these = left)</p>
    <table class="stock-usage-table"><tbody>${lines.join("")}</tbody></table>
  </div>`;
}

function stockDensity(lot) {
  const k = lot.kcal_per_100g != null ? `${fmt(lot.kcal_per_100g, 0)} kcal` : "";
  const n = lot.sodium_mg_per_100g != null ? `${fmt(lot.sodium_mg_per_100g, 0)} mg` : "";
  if (!k && !n) return "density unknown";
  return `${k}${k && n ? ", " : ""}${n} / 100g`;
}

function stockBatchEditorHtml() {
  ensureStockMealDraft();
  const f = state.stockForm;
  const set = state.mealDraft.stock.items[0];
  return `<p class="muted form-modal-hint">Name the batch, list ingredients for the <strong>full batch</strong>, then weighed batch total and what is left.</p>
    ${mealItemsGridHead()}
    ${itemBlock("stock", 0, set)}
    <div class="row" style="margin-top:0.55rem;flex-wrap:wrap">
      <label class="field date-field"><span>Prepared</span><input type="text" class="iso-date mono" id="stock-date" value="${esc(f.prepared_date)}" placeholder="YYYY-MM-DD" maxlength="10" inputmode="numeric" autocomplete="off" lang="en"></label>
      <label class="field tight"><span>Batch g</span><input id="stock-batch" value="${esc(f.batch_grams)}" inputmode="decimal"></label>
      <label class="field tight"><span>Left g</span><input id="stock-left" value="${esc(f.remaining_grams)}" inputmode="decimal" placeholder="= batch"></label>
      <label class="field tight stock-form-active"><span>Active</span><input type="checkbox" id="stock-active" ${f.active !== false ? "checked" : ""}></label>
    </div>
    <label class="field" style="margin-top:0.45rem"><span>Notes (optional per-100g in text)</span><textarea id="stock-notes" rows="2">${esc(f.notes)}</textarea></label>`;
}

function refreshStockModalBody(overlay) {
  const body = overlay.querySelector(".custom-modal-body");
  if (body) body.innerHTML = stockBatchEditorHtml();
  bindStockModal(overlay);
}

function bindStockModal(overlay) {
  bindIsoInputs(overlay);
  const refresh = () => refreshStockModalBody(overlay);
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
      resolveFoodPick(foodID, slot, idx, part === "" ? null : part);
    });
  });
  overlay.querySelectorAll("[data-pick-food]").forEach((b) => {
    b.addEventListener("click", () => {
      const food = state.foods.find((f) => f.id === b.dataset.pickFood);
      const target = draftTarget(b.dataset.slot, b.dataset.idx, b.dataset.part);
      if (food && target) {
        target.name = (b.dataset.pickLabel || "").trim() || food.name;
        target.food_id = food.id;
        target.kcal_per_100g = "";
        target.sodium_mg_per_100g = "";
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
}

async function openStockBatchModal() {
  const editing = !!state.stockForm?.id;
  const title = editing ? "Edit batch" : "New batch";
  await customModal({
    title,
    bodyHtml: stockBatchEditorHtml(),
    submitLabel: editing ? "Save changes" : "Save batch",
    extraClass: "stock-batch-modal",
    deleteLabel: editing ? "Delete" : undefined,
    onDelete: editing
      ? () => deleteStock(state.stockForm?.id)
      : undefined,
    onMount: bindStockModal,
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
  set.parts = (lot.items || []).length ? lot.items.map(cloneDraftItem) : [emptyItem()];
  pruneBlankSetParts(set);
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
    return `<article class="card stock-card clickable${active ? "" : " stock-inactive"}" data-open-stock="${esc(lot.id)}" style="margin-bottom:0.65rem">
      <div class="slot-head stock-lot-head">
        <label class="stock-active-switch" title="Active in fridge">
          <input type="checkbox" data-stock-active="${esc(lot.id)}" ${active ? "checked" : ""}>
          <span class="stock-active-pill">${active ? "On" : "Off"}</span>
        </label>
        <h3 style="margin:0;flex:1">${esc(lot.name)}</h3>
        <span class="mono muted stock-qty" title="current / original batch">${qtyLabel}</span>
      </div>
      <p class="muted" style="margin:0.25rem 0;font-size:0.82rem">${esc(stockDensity(lot))}${lot.prepared_date ? `, ${esc(lot.prepared_date)}` : ""}</p>
      ${stockLotIngredientsBlock(lot)}
      ${stockUsageRows(lot)}
      ${lot.notes ? `<p class="muted" style="font-size:0.78rem;max-height:4.5rem;overflow:auto;margin-top:0.35rem">${esc(lot.notes)}</p>` : ""}
      ${!active && hasLeft ? `<p class="muted" style="font-size:0.78rem;margin-top:0.25rem">Imported from an old “still in stock” note. Toggle <strong>On</strong> only if it is still in the fridge.</p>` : ""}
      ${hasLeft ? `<div class="row stock-card-actions" style="margin-top:0.5rem;flex-wrap:wrap;gap:0.35rem">
        ${active ? `<button type="button" class="btn primary" data-draw-stock="${esc(lot.id)}">Log my meal</button>` : ""}
        <button type="button" class="btn" data-stock-use="${esc(lot.id)}">Use (no log)</button>
        <button type="button" class="btn" data-stock-discard="${esc(lot.id)}">Discard</button>
      </div>` : ""}
    </article>`;
  };
  const lots = state.stock || [];
  const activeLots = lots.filter((lot) => stockLotActive(lot));
  const usedLots = lots.filter((lot) => !stockLotActive(lot));
  const activeRows = activeLots.map(lotCard).join("");
  const usedRows = usedLots.map(lotCard).join("");
  return el(`<div>
    ${dataBackupBarHtml("stock", { extra: `<button type="button" class="btn primary" id="stock-add">New batch</button>` })}
    <h2>Active (${activeLots.length})</h2>
    <div class="stock-list">${activeRows || `<p class="muted">Nothing left in the fridge</p>`}</div>
    <h2 style="margin-top:1.25rem">Used up (${usedLots.length})</h2>
    <div class="stock-list stock-used">${usedRows || `<p class="muted">No history yet</p>`}</div>
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
    return `<div class="eu-row${sub ? " eu-sub" : ""}"><span>${label}</span><span class="mono">${val}</span></div>`;
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
    : `Nutrition declaration <span class="muted">(per 100 g)</span>`;
  return `<div class="eu-nutrition-label${compact ? " eu-nutrition-label-compact" : ""}">
    <div class="eu-label-title">${title}</div>
    ${rows || `<p class="muted" style="margin:0">No label data</p>`}
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
    return `<p class="muted food-alias-empty" style="margin:0;font-size:0.85rem">No aliases yet — meal lines use exact catalog name or fuzzy match.</p>`;
  }
  return `<div class="food-alias-chips" role="list">${aliases.map((alias) =>
    `<span class="food-alias-chip" role="listitem">
      <span class="mono">${esc(alias)}</span>
      <button type="button" class="food-alias-chip-remove" data-food-alias-remove="${esc(alias)}" title="Remove alias" aria-label="Remove alias ${esc(alias)}">×</button>
    </span>`
  ).join("")}</div>`;
}

function foodFormAliasesSectionHtml(foodId) {
  if (!foodId) return "";
  return `<section class="food-alias-section" data-food-alias-section="${esc(foodId)}">
    <p class="muted form-modal-hint" style="margin:0 0 0.45rem">Aliases — other names that should match this catalog food in meals and import.</p>
    <div class="food-alias-list">${foodAliasChipsHtml(foodAliasesForId(foodId))}</div>
    <label class="field" style="margin-top:0.55rem"><span>Add aliases (one per line)</span>
      <textarea data-food-alias-bulk rows="3" placeholder="peppers&#10;grilled peppers&#10;florina pepper"></textarea>
    </label>
    <button type="button" class="btn" data-food-alias-bulk-add style="margin-top:0.35rem">Add</button>
  </section>`;
}

function mountFoodFormAliases(overlay, foodId) {
  const section = overlay.querySelector("[data-food-alias-section]");
  if (!section || !foodId) return;

  const refresh = () => {
    const list = section.querySelector(".food-alias-list");
    if (list) list.innerHTML = foodAliasChipsHtml(foodAliasesForId(foodId));
    list?.querySelectorAll("[data-food-alias-remove]").forEach((btn) => {
      btn.addEventListener("click", () => removeFoodAlias(foodId, btn.dataset.foodAliasRemove, refresh));
    });
  };

  const addNames = async (names) => {
    const uniq = [...new Set(names.map((n) => n.trim()).filter(Boolean))];
    if (!uniq.length) return;
    try {
      for (const alias of uniq) {
        await api(`/foods/${foodId}/aliases`, { method: "POST", body: JSON.stringify({ alias }) });
      }
      await loadFoods();
      refresh();
      render();
      toast(uniq.length > 1 ? `Added ${uniq.length} aliases` : "Alias added");
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

function foodFormFieldsHtml(f) {
  const salt = foodSaltG(f);
  return `<label class="field grow"><span>Name</span><input data-ff="name" value="${esc(f?.name || "")}"></label>
    <p class="muted form-modal-hint" style="margin:0">EU nutrition declaration (Reg. 1169/2011)</p>
    <div class="form-modal-row row">
      <label class="field tight"><span>kcal</span><input data-ff="kcal" value="${esc(f?.kcal_per_100g ?? "")}" inputmode="decimal"></label>
      <label class="field tight"><span>kJ</span><input data-ff="kj" value="${esc(f?.energy_kj_per_100g ?? "")}" inputmode="decimal" placeholder="auto"></label>
    </div>
    <div class="form-modal-row row">
      <label class="field tight"><span>Fat g</span><input data-ff="fat" value="${esc(f?.fat_g_per_100g ?? "")}" inputmode="decimal"></label>
      <label class="field tight"><span>Sat g</span><input data-ff="sat" value="${esc(f?.saturates_g_per_100g ?? "")}" inputmode="decimal"></label>
    </div>
    <div class="form-modal-row row">
      <label class="field tight"><span>Carbs g</span><input data-ff="carbs" value="${esc(f?.carbs_g_per_100g ?? "")}" inputmode="decimal"></label>
      <label class="field tight"><span>Sugars g</span><input data-ff="sugars" value="${esc(f?.sugars_g_per_100g ?? "")}" inputmode="decimal"></label>
    </div>
    <div class="form-modal-row row">
      <label class="field tight"><span>Protein g</span><input data-ff="protein" value="${esc(f?.protein_g_per_100g ?? "")}" inputmode="decimal"></label>
      <label class="field tight"><span>Salt g</span><input data-ff="salt" value="${esc(salt ?? "")}" inputmode="decimal" title="EU label salt (NaCl g per 100 g)"></label>
      <p class="muted form-modal-hint" style="margin:0;grid-column:1/-1">Salt on EU labels is NaCl (g/100g). Meal sodium uses 393 mg Na per 1 g salt (not mg×1000). Greek chicken souvlaki skewers are typically ~1.2–1.3 g salt/100g (~470–510 mg Na).</p>
    </div>
    <label class="field"><span>Notes</span><textarea data-ff="notes" rows="2">${esc(f?.notes || "")}</textarea></label>
    ${foodFormAliasesSectionHtml(f?.id)}`;
}

function readFoodForm(overlay) {
  const val = (k) => overlay.querySelector(`[data-ff="${k}"]`)?.value ?? "";
  const salt = numOrUndef(val("salt"));
  const kcal = numOrUndef(val("kcal"));
  let kj = numOrUndef(val("kj"));
  if (kj == null && kcal != null) kj = Math.round(kcal * 4.184 * 10) / 10;
  return {
    name: val("name").trim(),
    kcal_per_100g: kcal,
    energy_kj_per_100g: kj,
    fat_g_per_100g: numOrUndef(val("fat")),
    saturates_g_per_100g: numOrUndef(val("sat")),
    carbs_g_per_100g: numOrUndef(val("carbs")),
    sugars_g_per_100g: numOrUndef(val("sugars")),
    protein_g_per_100g: numOrUndef(val("protein")),
    salt_g_per_100g: salt,
    sodium_mg_per_100g: salt != null ? sodiumMgFromSaltG(salt) : undefined,
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
      ? `<p class="food-catalog-aliases muted" title="Meal-line aliases">${aliases.map((a) => esc(a)).join(" · ")}</p>`
      : "";
    return `<article class="food-catalog-card clickable" data-open-food="${esc(f.id)}">
    <div class="food-catalog-layout">
      <div class="food-catalog-main">
        <h3 class="food-catalog-name">${esc(foodCatalogDisplayName(f.name))}</h3>
        ${aliasHint}
        ${f.notes ? `<p class="food-catalog-notes muted">${esc(f.notes)}</p>` : ""}
      </div>
      <div class="food-catalog-label-wrap">${foodEuLabelHtml(f, { compact: true })}</div>
    </div>
  </article>`;
  }).join("");
  return el(`<div>
    ${dataBackupBarHtml("foods", { extra: `<button type="button" class="btn primary" id="foods-add">Add food</button>` })}
    <div class="panel-toolbar row searchbar">
      <label class="field grow"><span>Search catalog</span><input id="foods-q" value="${esc(state.foodsQ || "")}" placeholder="pasta, chicken..."></label>
    </div>
    <p class="muted" style="margin:0 0 0.65rem">${list.length} foods (per 100 g)</p>
    <div class="foods-catalog-list">${cards || `<p class="muted">No foods</p>`}</div>
  </div>`);
}

function weighInEnergy(w) {
  const b = mergedKcalBudget(state.kcalBudget);
  return energyFromInputs({
    kg: w?.kg,
    heightCm: b.height_cm,
    ageYears: b.age_years,
    sex: b.sex,
  });
}

function weightView() {
  const latest = state.overview?.latest_weight;
  const weightsInPeriod = recordsInRange(state.weights, state.weightRange, resolveTodayDate(state));
  const periodSelect = periodRangeSelectHtml("weight-range", state.weightRange, esc, { label: "" });
  const rows = weightsInPeriod.slice().reverse().map((w) => {
    const est = weighInEnergy(w);
    return `<tr class="clickable" data-open-weight="${esc(w.id)}">
    <td class="mono">${esc(w.date)}</td>
    <td>${fmt(w.kg, 1)}</td>
    <td class="mono">${est ? fmt(est.bmr, 0) : "-"}</td>
    <td>${w.rhr_bpm ?? ""}</td>
    <td>${w.protocol ? "yes" : ""}</td>
    <td>${esc(w.note || "")}</td>
  </tr>`;
  }).join("");
  const root = el(`<div>
    ${dataBackupBarHtml("weight", { extra: `<button type="button" class="btn primary" id="weight-add">Add weigh-in</button>` })}
    <div class="weight-period-row">${periodSelect}</div>
    ${weightChartSectionHtml()}
    <div class="panel-toolbar row" style="margin-top:0.65rem">
      <p class="muted" style="margin:0;flex:1 1 auto">Latest: ${latest ? `${fmt(latest.kg, 1)} kg on ${esc(latest.date)}` : "none"}</p>
    </div>
    <div class="card" style="margin-top:0.65rem">
      <table>
        <thead><tr><th>Date</th><th>kg</th><th>BMR</th><th>RHR</th><th>Protocol</th><th>Note</th></tr></thead>
        <tbody>${rows || `<tr><td colspan="6" class="muted">No weigh-ins</td></tr>`}</tbody>
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
  });
  if (!kg) {
    return `<p class="muted" id="energy-estimate" style="margin:0.55rem 0 0">Add a weigh-in to compute daily burn.</p>`;
  }
  if (!est) {
    return `<p class="muted" id="energy-estimate" style="margin:0.55rem 0 0">Fill sex, age, and height. Weight used: ${fmt(kg, 1)} kg.</p>`;
  }
  return `<p class="muted" id="energy-estimate" style="margin:0.55rem 0 0">
    Using ${fmt(est.kg, 1)} kg: BMR ${fmt(est.bmr, 0)} kcal, sedentary ${fmt(est.sedentary, 0)} kcal/day.
  </p>`;
}

function settingsView() {
  const b = mergedKcalBudget(state.kcalBudget);
  const kg = latestEnergyKg();
  const kgLine = latestEnergyKgLabel();
  const sex = b.sex === "female" ? "female" : b.sex === "male" ? "male" : "";
  return el(`<div>
    <section class="card">
      <h2>Sedentary energy</h2>
      <p class="muted form-modal-hint" style="margin-top:0">Mifflin-St Jeor resting burn (BMR), then times 1.2 for a sedentary day (little or no exercise). Weight is your latest weigh-in.</p>
      <div class="row settings-kcal-fields" style="flex-wrap:wrap;gap:0.65rem;margin-top:0.85rem">
        <label class="field"><span>Sex</span>
          <select id="set-sex">
            <option value=""${sex ? "" : " selected"}>Select</option>
            <option value="male"${sex === "male" ? " selected" : ""}>Male</option>
            <option value="female"${sex === "female" ? " selected" : ""}>Female</option>
          </select>
        </label>
        <label class="field"><span>Age (years)</span><input id="set-age" type="number" min="12" max="120" step="1" value="${esc(b.age_years ? String(b.age_years) : "")}"></label>
        <label class="field"><span>Height (cm)</span><input id="set-height" type="number" min="80" max="250" step="0.5" value="${esc(b.height_cm ? String(b.height_cm) : "")}"></label>
      </div>
      <p class="muted" style="font-size:0.82rem;margin:0.55rem 0 0">${kgLine ? `Latest weigh-in: ${esc(kgLine)}` : "No weigh-in yet."}</p>
      ${energyEstimateHtml(b, kg)}
    </section>
    <section class="card" style="margin-top:0.85rem">
      <h2>Daily kcal bands</h2>
      <p class="muted form-modal-hint" style="margin-top:0">The colored strip on the left of Today and each day card uses catalog day totals.</p>
      <ul class="kcal-band-legend muted">
        <li><span class="kcal-band-swatch kcal-band-swatch--green" aria-hidden="true"></span> Green: below soft band low</li>
        <li><span class="kcal-band-swatch kcal-band-swatch--yellow" aria-hidden="true"></span> Yellow: in soft band (target intake)</li>
        <li><span class="kcal-band-swatch kcal-band-swatch--orange" aria-hidden="true"></span> Orange: above soft band, below gain threshold</li>
        <li><span class="kcal-band-swatch kcal-band-swatch--red" aria-hidden="true"></span> Red: gain territory</li>
      </ul>
      <div class="row settings-kcal-fields" style="flex-wrap:wrap;gap:0.65rem;margin-top:0.85rem">
        <label class="field"><span>Soft band low (kcal)</span><input id="set-soft-low" type="number" min="1" step="50" value="${esc(String(b.soft_low_kcal))}"></label>
        <label class="field"><span>Soft band high (kcal)</span><input id="set-soft-high" type="number" min="1" step="50" value="${esc(String(b.soft_high_kcal))}"></label>
        <label class="field"><span>Gain from (kcal)</span><input id="set-gain-min" type="number" min="1" step="50" value="${esc(String(b.gain_min_kcal))}"></label>
      </div>
      <p class="muted" style="font-size:0.82rem;margin:0.55rem 0 0">Seeded from your Energy baseline notes: 1,500-1,700 kcal target; 2,200 kcal gain line.</p>
    </section>
    <section class="card" style="margin-top:0.85rem">
      <h2>Daily sodium bands</h2>
      <p class="muted form-modal-hint" style="margin-top:0">Colors the day total mg Na (same traffic-light pattern as kcal).</p>
      <ul class="kcal-band-legend muted">
        <li><span class="kcal-band-swatch kcal-band-swatch--green" aria-hidden="true"></span> Green: below soft low</li>
        <li><span class="kcal-band-swatch kcal-band-swatch--yellow" aria-hidden="true"></span> Yellow: soft low – soft high (target band)</li>
        <li><span class="kcal-band-swatch kcal-band-swatch--orange" aria-hidden="true"></span> Orange: above soft high, below ceiling</li>
        <li><span class="kcal-band-swatch kcal-band-swatch--red" aria-hidden="true"></span> Red: at/above ceiling</li>
      </ul>
      <div class="row settings-kcal-fields" style="flex-wrap:wrap;gap:0.65rem;margin-top:0.85rem">
        <label class="field"><span>Soft low (mg)</span><input id="set-sodium-low" type="number" min="1" step="50" value="${esc(String(b.soft_low_sodium_mg))}"></label>
        <label class="field"><span>Soft high (mg)</span><input id="set-sodium-high" type="number" min="1" step="50" value="${esc(String(b.soft_high_sodium_mg))}"></label>
        <label class="field"><span>Ceiling (mg)</span><input id="set-sodium-ceiling" type="number" min="1" step="50" value="${esc(String(b.ceiling_sodium_mg))}"></label>
      </div>
      <p class="muted" style="font-size:0.82rem;margin:0.55rem 0 0">Defaults: 1,200 / 2,000 / 2,400 mg (historical ~2 g day ceiling).</p>
    </section>
    <section class="card" style="margin-top:0.85rem">
      <h2>Markdown diff bands (dots after kcal / mg)</h2>
      <p class="muted form-modal-hint" style="margin-top:0">Small dots show <span class="mono">|catalog − markdown day log|</span> per nutrient. Same scale for kcal and sodium.</p>
      <ul class="kcal-band-legend muted">
        <li><span class="kcal-band-swatch kcal-band-swatch--green" aria-hidden="true"></span> Green: diff &lt; green limit</li>
        <li><span class="kcal-band-swatch kcal-band-swatch--yellow" aria-hidden="true"></span> Yellow: diff &lt; yellow limit</li>
        <li><span class="kcal-band-swatch kcal-band-swatch--orange" aria-hidden="true"></span> Orange: diff &lt; orange limit</li>
        <li><span class="kcal-band-swatch kcal-band-swatch--red" aria-hidden="true"></span> Red: diff ≥ orange limit</li>
      </ul>
      <div class="row settings-kcal-fields" style="flex-wrap:wrap;gap:0.65rem;margin-top:0.85rem">
        <label class="field"><span>Green below</span><input id="set-diff-green" type="number" min="1" step="1" value="${esc(String(b.legacy_diff_green_lt))}"></label>
        <label class="field"><span>Yellow below</span><input id="set-diff-yellow" type="number" min="2" step="1" value="${esc(String(b.legacy_diff_yellow_lt))}"></label>
        <label class="field"><span>Orange below</span><input id="set-diff-orange" type="number" min="3" step="1" value="${esc(String(b.legacy_diff_orange_lt))}"></label>
      </div>
      <p class="muted" style="font-size:0.82rem;margin:0.55rem 0 0">Defaults: 100 / 200 / 300 (e.g. 92 kcal off → green).</p>
    </section>
    <section class="card" style="margin-top:0.85rem">
      <h2>Meal total margin</h2>
      <p class="muted form-modal-hint" style="margin-top:0">Adds this percent to each meal slot total (lunch, dinner, snack) after catalog math, same as your markdown <span class="mono">+10% margin</span> lines. Day total is the sum of those meal totals. Item lines stay unpadded.</p>
      <label class="field" style="max-width:12rem;margin-top:0.65rem">
        <span>Margin (%)</span>
        <input id="set-meal-margin" type="number" min="0" max="100" step="0.5" value="${esc(String(b.meal_margin_percent))}">
      </label>
      <div class="row" style="margin-top:0.85rem;flex-wrap:wrap;gap:0.45rem">
        <button type="button" class="btn primary" id="settings-save">Save</button>
        <button type="button" class="btn" id="settings-reset">Reset defaults</button>
      </div>
    </section>
    <section class="card" style="margin-top:0.85rem">
      <h2>Backup & restore</h2>
      <p class="muted form-modal-hint" style="margin:0">Full JSON export (foods, aliases, days, stock, weight, settings). Import here replaces everything; use tab-specific import on Foods/Days/etc. to merge only that section.</p>
      ${dataBackupBarHtml("all", { fullBackup: true })}
      <label class="check" style="margin-top:0.45rem;display:block">
        <input type="checkbox" id="import-all-replace">
        Full import replaces all data (dangerous)
      </label>
    </section>
    <section class="card" style="margin-top:0.85rem">
      <h2>Fix imported days</h2>
      <p class="muted form-modal-hint" style="margin:0">Re-parse every saved meal line (strips day-totals glued to snacks, × counts, catalog math). Restores markdown legacy totals from <span class="mono">mddata/</span>.</p>
      <button type="button" class="btn primary" id="settings-repair-days" style="margin-top:0.75rem">Repair all days from raw</button>
    </section>
  </div>`);
}

function applyKcalBudgetFromOverview(overview) {
  if (overview?.settings) {
    state.kcalBudget = mergedKcalBudget(overview.settings);
  }
}

async function saveKcalBudgetFromForm() {
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
  };
  const data = await api("/settings", { method: "PUT", body: JSON.stringify(body) });
  state.kcalBudget = mergedKcalBudget(data.settings);
  if (state.overview) state.overview.settings = data.settings;
  toast("Settings saved");
  if (state.tab === "days") await loadDays();
  if (state.tab === "today" && state.date) {
    const dayData = await api(`/days/${encodeURIComponent(state.date)}`);
    state.day = dayData.day || state.day;
    fillDraftFromDay(state.day);
  }
  render();
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
  document.querySelector(".health-navbar .brand")?.addEventListener("click", (e) => {
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
      if (inp) inp.value = next;
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
      resolveFoodPick(foodID, slot, idx, part === "" ? null : part);
    });
  });
  app?.querySelectorAll("[data-pick-food]").forEach((b) => {
    b.addEventListener("click", () => {
      const food = state.foods.find((f) => f.id === b.dataset.pickFood);
      const target = draftTarget(b.dataset.slot, b.dataset.idx, b.dataset.part);
      if (food && target) {
        target.name = (b.dataset.pickLabel || "").trim() || food.name;
        target.food_id = food.id;
        target.kcal_per_100g = "";
        target.sodium_mg_per_100g = "";
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
      if (e.target.closest("button, label, input, a, .stock-card-actions")) return;
      beginEditStockBatch(card.dataset.openStock);
    });
  });
  document.getElementById("foods-add")?.addEventListener("click", () => openFoodForm().catch((e) => toast(e.message)));
  document.querySelectorAll("[data-open-food]").forEach((tr) => {
    tr.addEventListener("click", () => openFoodForm(tr.dataset.openFood).catch((e) => toast(e.message)));
  });

  document.getElementById("settings-save")?.addEventListener("click", () => {
    saveKcalBudgetFromForm().catch((e) => toast(e.message));
  });
  const refreshEnergyHint = () => {
    const elHint = document.getElementById("energy-estimate");
    if (!elHint) return;
    const wrap = document.createElement("div");
    wrap.innerHTML = energyEstimateHtml({
      sex: document.getElementById("set-sex")?.value || "",
      age_years: Number(document.getElementById("set-age")?.value) || 0,
      height_cm: Number(document.getElementById("set-height")?.value) || 0,
    }, latestEnergyKg());
    const next = wrap.firstElementChild;
    if (next) elHint.replaceWith(next);
  };
  ["set-sex", "set-age", "set-height"].forEach((id) => {
    document.getElementById(id)?.addEventListener("input", refreshEnergyHint);
    document.getElementById(id)?.addEventListener("change", refreshEnergyHint);
  });
  document.getElementById("settings-repair-days")?.addEventListener("click", async () => {
    const ok = await confirmModal({
      title: "Repair all days?",
      message: "<p class=\"muted\" style=\"margin:0\">Re-parses meal text in the database and recomputes catalog totals. Legacy markdown day totals are restored from mddata.</p>",
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
  if (lot?.id) row.stock_lot_id = lot.id;
  else if (it.stock_lot_id) row.stock_lot_id = it.stock_lot_id;
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
    message: `<p class="muted" style="margin:0">All meals and notes for <span class="mono">${esc(state.date)}</span> will be removed. This cannot be undone.</p>`,
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
    bodyHtml: `<div class="food-form-modal stack-fields">${foodFormFieldsHtml(f || {})}</div>`,
    submitLabel: editing ? "Save" : "Add",
    extraClass: "entity-form-modal food-entity-modal",
    onMount: (overlay) => {
      if (editing) mountFoodFormAliases(overlay, f.id);
      if (editing) {
        const actions = overlay.querySelector(".modal-action");
        if (actions && !actions.querySelector("[data-ff-delete]")) {
          const del = document.createElement("button");
          del.type = "button";
          del.className = "btn danger";
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
      if (!body.name) {
        toast("Name required");
        return false;
      }
      try {
        if (f) {
          await api(`/foods/${f.id}`, { method: "PUT", body: JSON.stringify(body) });
        } else {
          await api("/foods/batch", { method: "POST", body: JSON.stringify({ upsert: [body] }) });
        }
        await loadFoods();
        toast("Food saved");
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

/** @returns {Promise<boolean>} */
async function delFood(id, { skipReload = false } = {}) {
  const ok = await confirmModal({
    title: "Delete this food?",
    message: "<p class=\"muted\" style=\"margin:0\">Removes it from the catalog. Meals already logged keep their lines.</p>",
    confirmLabel: "Delete",
    danger: true,
  });
  if (!ok) return false;
  try {
    await api(`/foods/${id}`, { method: "DELETE" });
    await loadFoods();
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
          { key: "date", label: "Date", value: w?.date || state.date, date: true, tight: true },
          { key: "kg", label: "kg", value: w?.kg ?? "", type: "decimal", tight: true },
          { key: "rhr", label: "RHR", value: w?.rhr_bpm ?? "", type: "numeric", tight: true },
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
    message: "<p class=\"muted\" style=\"margin:0\">This cannot be undone.</p>",
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

async function loadDay() {
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
  if (state.day) {
    await refreshDayNutrition(draftItems, state.mealDraft);
  }
}

function stockFormRoot() {
  return document.querySelector(".stock-batch-modal") || document;
}

function syncStockFormFromDOM() {
  ensureStockMealDraft();
  const f = state.stockForm;
  const root = stockFormRoot();
  f.set = state.mealDraft.stock.items[0];
  f.prepared_date = root.querySelector("#stock-date")?.value?.trim() || f.prepared_date;
  f.batch_grams = root.querySelector("#stock-batch")?.value ?? f.batch_grams;
  f.remaining_grams = root.querySelector("#stock-left")?.value ?? f.remaining_grams;
  f.active = root.querySelector("#stock-active")?.checked !== false;
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
  const batch = numOrUndef(f.batch_grams);
  const setName = (set?.name || "").trim();
  if (!setName || !batch) {
    toast("Set name and batch grams required");
    return false;
  }
  const left = numOrUndef(f.remaining_grams) ?? batch;
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
    const overlay = el(`<div class="stock-dialog-overlay" role="dialog" aria-modal="true">
      <div class="stock-dialog">
        <h3>${esc(titles[kind] || "Stock")}</h3>
        <p class="stock-dialog-sub">${esc(lot.name)}: <span class="mono">${fmt(left, 0)} g</span> left</p>
        <label class="field"><span>Grams (required)</span>
          <input id="sd-grams" type="number" inputmode="decimal" min="0.1" step="1" placeholder="e.g. 150" autofocus>
        </label>
        ${kind === "meal" ? `
        <label class="field"><span>Meal slot</span>
          <select id="sd-slot">
            <option value="lunch">lunch</option>
            <option value="dinner" selected>dinner</option>
            <option value="snack">snack</option>
          </select>
        </label>
        <label class="field"><span>Date</span>
          <input type="text" id="sd-date" class="iso-date mono" value="${esc(state.date || "")}" placeholder="YYYY-MM-DD" maxlength="10" inputmode="numeric" autocomplete="off" lang="en">
        </label>` : ""}
        <div class="row">
          <button type="button" class="btn" data-sd-cancel>Cancel</button>
          <button type="button" class="btn" data-sd-all title="Remove all remaining">All ${fmt(left, 0)} g</button>
          <button type="button" class="btn primary" data-sd-ok>OK</button>
        </div>
      </div>
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
    message: `<p class="muted" style="margin:0">${esc(lot.name)} <span class="mono">(${esc(qty)})</span></p><p class="muted" style="margin:0.5rem 0 0">Removes the lot and its usage history. This cannot be undone.</p>`,
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
  const [data, ov] = await Promise.all([
    api("/weight"),
    api("/overview?days=1").catch(() => null),
  ]);
  state.weights = data.weights || [];
  if (ov) {
    state.overview = ov;
    applyKcalBudgetFromOverview(ov);
  }
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
  await loadDay();
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
  state.overview = await api("/overview");
  applyKcalBudgetFromOverview(state.overview);
  applyRouteToState(state);
  if (!state.date) state.date = resolveTodayDate(state);
  await Promise.all([loadFoods(), loadStock(), loadWeights()]);
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
