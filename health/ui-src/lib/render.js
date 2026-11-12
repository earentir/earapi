import { AUTH_KEY, SLOTS } from "./constants.js";
import { themeSelectHtml, saveTheme } from "./theme.js";
import { state, emptyItem, emptySlot, emptyDraft, emptyStockSet, emptyStockForm } from "./state.js";
import { fmt, el, pad2, isoDate, parseISODate, shiftISODate, parseTime24, dateField, bindIsoInputs, confirmModal, weekdayNameForISODate } from "./util.js";
import { api, loadAuth, saveAuth, toast } from "./api.js";
import { esc } from "./strings.js";
import {
  itemHasParts,
  setsInSlot,
  pruneBlankSetParts,
  setPartStats,
  recomputeSetGrams,
  absorbIntoSet,
  ejectPartFromSet,
  cloneDraftItem,
} from "./meal-items.js";
import { bestFoodMatch, foodByAlias, foodMatchCandidates, catalogFoodSuggestions, foodPickCandidates } from "./fuzzy.js";
import { matchStockLotByName } from "./stock-match.js";

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
  box.querySelector("#login-go").addEventListener("click", async () => {
    state.user = box.querySelector("#login-user").value.trim();
    state.token = box.querySelector("#login-token").value.trim();
    const errEl = box.querySelector("#login-err");
    try {
      await api("/overview?days=1");
      saveAuth();
      await boot();
    } catch (e) {
      state.user = "";
      state.token = "";
      errEl.textContent = e.message || "login failed";
    }
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
  return el(`<header class="topbar"><div class="topbar-inner">
    <div><span class="brand">health</span><span class="brand-sub">${esc(state.user)}</span></div>
    <nav class="nav">
      ${["today", "days", "foods", "stock", "weight"].map((t) =>
        `<button data-tab="${t}" class="${state.tab === t ? "active" : ""}">${t}</button>`
      ).join("")}
    </nav>
    ${themeSelectHtml(document.documentElement.dataset.theme || "dark")}
    <button class="btn ghost" id="logout">Log out</button>
  </div></header>`);
}

function content() {
  const wrap = el(`<main class="wrap"></main>`);
  if (state.tab === "today") wrap.append(todayView());
  if (state.tab === "days") wrap.append(daysView());
  if (state.tab === "foods") wrap.append(foodsView());
  if (state.tab === "stock") wrap.append(stockView());
  if (state.tab === "weight") wrap.append(weightView());
  return wrap;
}

function slotMealTotals(slot) {
  const m = state.day?.meals?.[slot];
  if (!m) return null;
  const has = (m.items || []).length > 0 || (m.raw && !itemSkipRaw(m.raw));
  if (!has) return null;
  return { kcal: m.kcal, sodium_mg: m.sodium_mg };
}

function daySummaryPanel() {
  const totals = state.day?.totals || { kcal: 0, sodium_mg: 0 };
  const d = state.mealDraft;
  const weight = (state.weights || []).find((w) => w.date === state.date);
  const slotRows = SLOTS.map((slot) => {
    const label = slot.charAt(0).toUpperCase() + slot.slice(1);
    const t = slotMealTotals(slot);
    const kcalHtml = t
      ? `<span class="day-slot-kcal mono">${fmt(t.kcal, 0)} <span class="stat-unit">kcal</span></span>`
      : `<span class="day-slot-kcal mono muted">—</span>`;
    const naHtml = t
      ? `<span class="day-slot-na mono">${fmt(t.sodium_mg, 0)} <span class="stat-unit">mg</span></span>`
      : `<span class="day-slot-na mono muted">—</span>`;
    return `<div class="day-slot-stat">
      <span class="day-slot-name">${esc(label)}</span>
      ${kcalHtml}
      ${naHtml}
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
  const headEnergySodium = `<div class="day-panel-head-stats" aria-label="Day totals">
      <div class="day-stat">
        <span class="stat-label">Energy</span>
        <span class="stat-value mono">${fmt(totals.kcal, 0)} <span class="stat-unit">kcal</span></span>
      </div>
      <div class="day-stat">
        <span class="stat-label">Sodium</span>
        <span class="stat-value mono">${fmt(totals.sodium_mg, 0)} <span class="stat-unit">mg</span></span>
      </div>
    </div>`;
  const bodyStats = (weightRow || statusRow)
    ? `<div class="day-stats">${weightRow}${statusRow}</div>`
    : "";
  return `<section class="card day-panel">
    <div class="slot-head day-panel-head">
      <h2 class="day-panel-title" style="margin:0"><span class="mono">${esc(state.date)}</span> ${esc(weekdayNameForISODate(state.date))}</h2>
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
  const byAlias = foodByAlias(state.foodAliases, state.foods, it?.name);
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
  if (foodByAlias(state.foodAliases, state.foods, it.name)) return;
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
  return !!foodByAlias(state.foodAliases, state.foods, it?.name);
}

function showFoodPickUi(it) {
  if (!it || itemHasParts(it)) return false;
  if (foodItemLinked(it)) return false;
  if (stockLotForItem(it)) return false;
  return !!(it?.name || "").trim();
}

function foodPickOptions(it) {
  const name = (it?.name || "").trim();
  if (!name || itemHasParts(it)) return [];
  if (foodItemLinked(it)) return [];
  let opts = foodPickCandidates(state.foods, name);
  if (!opts.length) opts = catalogFoodSuggestions(state.foods, name, 15);
  return opts;
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
    opts = foodPickCandidates(state.foods, (it?.name || "").trim());
    if (!opts.length) opts = catalogFoodSuggestions(state.foods, (it?.name || "").trim(), 15);
  }
  if (!opts.length) {
    return `<div class="food-pick-list food-pick-list-empty muted" role="status">No catalog matches — add the food under Foods, or set per 100 g in the kcal / Na columns.</div>`;
  }
  return `<div class="food-pick-list" role="listbox">${opts.map((f) =>
    `<button type="button" data-resolve-food="${esc(f.id)}:${esc(slot)}:${i}${partIdx == null ? "" : `:${partIdx}`}" title="Save alias for this line name">
      ${esc(f.name)} <span class="muted">${fmt(f.kcal_per_100g, 0)} kcal/100g</span>
    </button>`
  ).join("")}</div>`;
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
    scheduleSaveDay();
    toast("Linked to catalog");
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

function itemNutrientPreview(it) {
  const g = Number(it.grams) || 0;
  if (g <= 0) return { kcal: null, na: null, fromStock: false };
  const lot = stockLotForItem(it);
  if (lot) {
    const k100 = lot.kcal_per_100g;
    const n100 = lot.sodium_mg_per_100g;
    const kcal = k100 != null ? (k100 * g) / 100 : null;
    const na = n100 != null ? (n100 * g) / 100 : null;
    if ((kcal != null && kcal > 0) || (na != null && na > 0)) {
      return { kcal, na, fromStock: true };
    }
  }
  const f = foodForItem(it);
  if (!f) return { kcal: null, na: null, fromStock: false };
  const kcal = f.kcal_per_100g != null ? (f.kcal_per_100g * g) / 100 : null;
  const na = f.sodium_mg_per_100g != null ? (f.sodium_mg_per_100g * g) / 100 : null;
  return { kcal, na, fromStock: false };
}

function slotCard(slot, draft) {
  const meal = state.day?.meals?.[slot];
  const summary = meal ? `${fmt(meal.kcal, 0)} kcal, ${fmt(meal.sodium_mg, 0)} mg` : "empty";
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
      <button class="btn" data-add-dish="${slot}">Add set</button>
    </div>
  </section>`;
}

function suggestList(slot, i, partIdx, q) {
  const open = state.suggest.slot === slot && state.suggest.i === i
    && (state.suggest.p ?? -1) === (partIdx == null ? -1 : partIdx);
  const query = (q || "").trim();
  if (!open || query.length < 1) return "";
  const hits = catalogFoodSuggestions(state.foods, query, 8);
  if (!hits.length) return "";
  const partAttr = partIdx == null ? "" : ` data-part="${partIdx}"`;
  return `<div class="suggest-list">${hits.map((f) =>
    `<button type="button" data-pick-food="${esc(f.id)}" data-slot="${esc(slot)}" data-idx="${i}"${partAttr}>${esc(f.name)} <span class="muted">${fmt(f.kcal_per_100g, 0)} kcal/100g</span></button>`
  ).join("")}</div>`;
}

function itemKcalCell(it) {
  const stored = Number(it.kcal);
  if (!Number.isNaN(stored) && stored > 0) return fmt(stored, 0);
  const { kcal } = itemNutrientPreview(it);
  return kcal != null && kcal > 0 ? fmt(kcal, 0) : "-";
}

function itemNaCell(it) {
  const stored = Number(it.sodium_mg);
  if (!Number.isNaN(stored) && stored > 0) return fmt(stored, 0);
  const { na } = itemNutrientPreview(it);
  return na != null && na > 0 ? fmt(na, 0) : "-";
}

function needsPer100gRow(it) {
  if (!(it?.name || "").trim() || itemHasParts(it)) return false;
  if (foodItemLinked(it)) return false;
  if (foodPickOptions(it).length >= 1) return false;
  if (stockLotForItem(it)) return false;
  const k = Number(it.kcal);
  const n = Number(it.sodium_mg);
  if ((!Number.isNaN(k) && k > 0) || (!Number.isNaN(n) && n > 0)) return false;
  return true;
}

function rowIconBtn(attrs, title, glyph) {
  return `<button type="button" class="btn icon-btn" title="${esc(title)}" aria-label="${esc(title)}" ${attrs}>${glyph}</button>`;
}

function mealItemsGridHead() {
  return `<div class="item-row item-row-head muted" aria-hidden="true">
    <span class="icon-spacer"></span>
    <span>Name</span>
    <span class="num">g</span>
    <span class="num">kcal</span>
    <span class="num">Na mg</span>
    <span class="icon-spacer"></span>
  </div>`;
}

function nameGramsRow(slot, i, it, partIdx) {
  const isPart = partIdx != null;
  const needPer100g = needsPer100gRow(it);
  const partAttr = isPart ? ` data-part="${partIdx}"` : "";
  const isSetHeader = !isPart && itemHasParts(it);
  const setStats = isSetHeader ? setPartStats(it) : null;
  const gramsPh = isPart ? "g" : "g";
  const namePh = isPart ? "Ingredient" : (isSetHeader ? (slot === "stock" ? "Batch name" : "Set name") : "Food");
  const gramsCell = isSetHeader
    ? `<span class="cell-num mono cell-sum" title="${setStats.count} ingredients, total grams">${setStats.count} ${fmt(setStats.grams, 0)}</span>`
    : `<input class="cell-grams" data-field="grams" data-slot="${esc(slot)}" data-idx="${i}"${partAttr} value="${esc(it.grams)}" placeholder="${gramsPh}" inputmode="decimal" title="grams">`;
  let lead;
  if (isPart) {
    lead = rowIconBtn(`data-eject-part="${esc(slot)}:${i}:${partIdx}"`, "Remove from set", "↩");
  } else if (!itemHasParts(it)) {
    lead = rowIconBtn(`data-make-dish="${esc(slot)}:${i}"`, "Convert to set", "⊞");
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
    ? `<button type="button" class="per100-tag row-vert-label pick-tag" data-food-pick="${esc(slot)}:${i}${partIdx == null ? "" : `:${partIdx}`}" title="${esc(pickOpts.length ? `${pickOpts.length} catalog match${pickOpts.length === 1 ? "" : "es"} — pick one` : "Pick food from catalog")}" aria-label="Pick food from catalog">pick</button>`
    : "";
  const stockLot = stockLotForItem(it);
  const stockHint = stockLot && !needPer100g
    ? `<span class="per100-tag row-vert-label stock-tag" title="${esc(stockLot.name)}">stock</span>`
    : "";
  const nameTitle = needPer100g ? "Not in Foods or Stock — set per 100 g in the kcal and Na columns" : "";
  const nameInput = `<div class="suggest name-field">
        <span class="name-input-row">${stockHint}${pickBtn}<input data-field="name" data-slot="${esc(slot)}" data-idx="${i}"${partAttr} value="${esc(it.name)}" placeholder="${namePh}" autocomplete="off"${nameTitle ? ` title="${esc(nameTitle)}"` : ""}></span>
        ${foodPickList(slot, i, partIdx, it, pickOpts)}
        ${suggestList(slot, i, partIdx, it.name)}
      </div>`;
  const kcalCell = needPer100g
    ? `<input class="cell-grams cell-per100g" data-field="kcal_per_100g" data-slot="${esc(slot)}" data-idx="${i}"${partAttr} value="${esc(it.kcal_per_100g)}" placeholder="kcal/100g" inputmode="decimal" title="kcal per 100 g (not in catalog)">`
    : "";
  const naCell = needPer100g
    ? `<input class="cell-grams cell-per100g" data-field="sodium_mg_per_100g" data-slot="${esc(slot)}" data-idx="${i}"${partAttr} value="${esc(it.sodium_mg_per_100g)}" placeholder="Na/100g" inputmode="decimal" title="sodium mg per 100 g (not in catalog)">`
    : `<span class="cell-num mono">${itemNaCell(it)}</span>`;
  const kcalCellOut = needPer100g
    ? kcalCell
    : `<span class="cell-num mono">${itemKcalCell(it)}</span>`;
  const nameCell = addPartBtn
    ? `<span class="name-with-actions">${nameInput}${addPartBtn}</span>`
    : nameInput;
  return `<div class="item-row ${isPart ? "part-row" : ""}">
    ${lead}
    <div class="cell-name cell-name-split">
      ${nameCell}
      ${addToSet}
    </div>
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
  const tot = m && (m.kcal || m.sodium_mg)
    ? ` <span class="mono muted">${fmt(m.kcal, 0)} kcal, ${fmt(m.sodium_mg, 0)} mg</span>`
    : "";
  return `<div class="meal-line"><span class="slot-lab">${slot}</span><span>${esc(text)}${tot}</span></div>`;
}

function daysView() {
  const cards = (state.days || []).map((d) => {
    const meals = ["lunch", "dinner", "snack"].map((s) => mealLine(s, d)).join("");
    const meta = [d.bp ? `BP ${esc(d.bp)}` : "", d.walking ? esc(d.walking) : ""].filter(Boolean).join(", ");
    return `<article class="day-card clickable" data-open="${esc(d.date)}">
      <div class="day-card-head">
        <span class="mono date-iso">${esc(d.date)}</span>
        <span class="mono">${fmt(d.totals?.kcal, 0)} kcal, ${fmt(d.totals?.sodium_mg, 0)} mg Na</span>
      </div>
      ${meals || `<p class="muted">No meal text</p>`}
      ${meta ? `<p class="muted">${meta}</p>` : ""}
    </article>`;
  }).join("");
  return el(`<div>
    <div class="row searchbar">
      <label class="field grow"><span>Search meals</span><input id="days-q" value="${esc(state.daysQ)}" placeholder="moussaka, pasta…"></label>
      <button class="btn primary" id="days-go">Search</button>
    </div>
    <div class="day-list">${cards || `<p class="muted">No days</p>`}</div>
  </div>`);
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
    return `<p class="muted" style="font-size:0.78rem;margin:0.35rem 0 0">No usage logged yet — <strong>left</strong> = batch minus lines below.</p>`;
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
            ? "household"
            : m.kind === "discarded"
              ? "discarded"
              : esc(m.kind || "adjustment");
    const note = m.note ? ` <span class="muted">(${esc(m.note)})</span>` : "";
    const g = m.grams < 0 ? `put back ${fmt(-m.grams, 0)} g` : `−${fmt(m.grams, 0)} g`;
    return `<tr><td class="mono">${what}${note}</td><td class="mono">${g}</td><td class="muted">${when}</td></tr>`;
  });
  return `<div class="stock-usage" style="margin-top:0.45rem">
    <p class="muted" style="font-size:0.78rem;margin:0 0 0.25rem"><strong>Usage</strong> (batch − these = left)</p>
    <table class="stock-usage-table"><tbody>${lines.join("")}</tbody></table>
  </div>`;
}

function stockDensity(lot) {
  const k = lot.kcal_per_100g != null ? `${fmt(lot.kcal_per_100g, 0)} kcal` : "";
  const n = lot.sodium_mg_per_100g != null ? `${fmt(lot.sodium_mg_per_100g, 0)} mg` : "";
  if (!k && !n) return "density unknown";
  return `${k}${k && n ? ", " : ""}${n} / 100g`;
}

function stockView() {
  ensureStockMealDraft();
  const f = state.stockForm;
  const set = state.mealDraft.stock.items[0];
  const lotCard = (lot) => {
    const active = stockLotActive(lot);
    const left = Math.max(0, lot.remaining_grams || 0);
    const batch = lot.batch_grams || 0;
    const hasLeft = left > 0.01;
    const qtyLabel = batch > 0 ? `${fmt(left, 0)}/${fmt(batch, 0)} g` : `${fmt(left, 0)} g`;
    return `<article class="card stock-card${active ? "" : " stock-inactive"}" style="margin-bottom:0.65rem">
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
      ${!active && hasLeft ? `<p class="muted" style="font-size:0.78rem;margin-top:0.25rem">Imported from an old “still in stock” note — toggle <strong>On</strong> only if it is still in the fridge.</p>` : ""}
      <div class="row" style="margin-top:0.5rem;flex-wrap:wrap;gap:0.35rem">
        ${active && hasLeft ? `<button type="button" class="btn primary" data-draw-stock="${esc(lot.id)}">Log my meal</button>` : ""}
        ${hasLeft ? `<button type="button" class="btn" data-stock-household="${esc(lot.id)}">Household ate</button>` : ""}
        ${hasLeft ? `<button type="button" class="btn" data-stock-discard="${esc(lot.id)}">Discard</button>` : ""}
        <button type="button" class="btn" data-edit-stock="${esc(lot.id)}">Edit</button>
        <button type="button" class="btn danger" data-del-stock="${esc(lot.id)}">Delete</button>
      </div>
    </article>`;
  };
  const lots = state.stock || [];
  const activeLots = lots.filter((lot) => stockLotActive(lot));
  const usedLots = lots.filter((lot) => !stockLotActive(lot));
  const activeRows = activeLots.map(lotCard).join("");
  const usedRows = usedLots.map(lotCard).join("");
  const editing = !!f.id;
  return el(`<div>
    <section class="card" style="margin-bottom:1rem">
      <h2>${editing ? "Edit batch" : "Prep batch"}</h2>
      <p class="muted">${editing ? `Editing <strong>${esc(set.name || f.id)}</strong> — cancel to discard changes.` : "One set in the fridge: name it like a meal set, list ingredients for the <strong>full batch</strong>, then batch weight (after cooking) and what is left."}</p>
      ${mealItemsGridHead()}
      ${itemBlock("stock", 0, set)}
      <div class="row" style="margin-top:0.55rem;flex-wrap:wrap">
        <label class="field date-field"><span>Prepared</span><input type="text" class="iso-date mono" id="stock-date" value="${esc(f.prepared_date)}" placeholder="YYYY-MM-DD" maxlength="10" inputmode="numeric" autocomplete="off" lang="en"></label>
        <label class="field tight"><span>Batch g</span><input id="stock-batch" value="${esc(f.batch_grams)}" inputmode="decimal" title="Weighed batch total (may differ from ingredient sum)"></label>
        <label class="field tight"><span>Left g</span><input id="stock-left" value="${esc(f.remaining_grams)}" inputmode="decimal" placeholder="= batch"></label>
        <label class="field tight stock-form-active"><span>Active</span><input type="checkbox" id="stock-active" ${f.active !== false ? "checked" : ""}></label>
      </div>
      <label class="field" style="margin-top:0.45rem"><span>Notes (optional per-100g in text)</span><textarea id="stock-notes" rows="2">${esc(f.notes)}</textarea></label>
      <div class="row" style="margin-top:0.35rem">
        <button type="button" class="btn primary" id="stock-save">${editing ? "Save changes" : "Save batch"}</button>
        ${editing
          ? `<button type="button" class="btn" id="stock-cancel-edit">Cancel</button>`
          : `<button type="button" class="btn" id="stock-clear">Clear form</button>`}
      </div>
    </section>
    <h2>Active (${activeLots.length})</h2>
    <div class="stock-list">${activeRows || `<p class="muted">Nothing left in the fridge</p>`}</div>
    <h2 style="margin-top:1.25rem">Used up (${usedLots.length})</h2>
    <div class="stock-list stock-used">${usedRows || `<p class="muted">No history yet</p>`}</div>
  </div>`);
}

function foodsView() {
  const q = (state.foodsQ || "").toLowerCase();
  const list = q
    ? state.foods.filter((f) => f.name.toLowerCase().includes(q) || (f.id || "").includes(q))
    : state.foods;
  const rows = list.slice(0, 200).map((f) => `<tr>
    <td>${esc(f.name)}</td>
    <td class="mono">${fmt(f.kcal_per_100g, 0)}</td>
    <td class="mono">${fmt(f.sodium_mg_per_100g, 1)}</td>
    <td class="muted">${esc(f.notes || "")}</td>
    <td><button class="btn" data-edit-food="${esc(f.id)}">Edit</button>
        <button class="btn danger" data-del-food="${esc(f.id)}">Delete</button></td>
  </tr>`).join("");
  return el(`<div>
    <div class="row searchbar">
      <label class="field grow"><span>Search catalog</span><input id="foods-q" value="${esc(state.foodsQ || "")}" placeholder="pasta, chicken…"></label>
    </div>
    <section class="card" style="margin-bottom:1rem">
      <h2>Add food</h2>
      <div class="row">
        <label class="field grow"><span>Name</span><input id="new-food-name"></label>
        <label class="field tight"><span>kcal / 100g</span><input id="new-food-kcal" inputmode="decimal"></label>
        <label class="field tight"><span>Na mg / 100g</span><input id="new-food-na" inputmode="decimal"></label>
        <label class="field"><span>Notes</span><input id="new-food-notes"></label>
        <button class="btn primary" id="add-food">Add</button>
      </div>
    </section>
    <div class="card">
      <p class="muted">${list.length} foods</p>
      <table>
        <thead><tr><th>Name</th><th>kcal</th><th>Na</th><th>Notes</th><th></th></tr></thead>
        <tbody>${rows || `<tr><td colspan="5" class="muted">No foods</td></tr>`}</tbody>
      </table>
    </div>
  </div>`);
}

function weightView() {
  const latest = state.overview?.latest_weight;
  const rows = (state.weights || []).slice().reverse().map((w) => `<tr>
    <td class="mono">${esc(w.date)}</td>
    <td>${fmt(w.kg, 1)}</td>
    <td>${w.rhr_bpm ?? ""}</td>
    <td>${w.protocol ? "yes" : ""}</td>
    <td>${esc(w.note || "")}</td>
    <td><button class="btn" data-edit-w="${esc(w.id)}">Edit</button>
        <button class="btn danger" data-del-w="${esc(w.id)}">Delete</button></td>
  </tr>`).join("");
  return el(`<div>
    <p class="muted">Latest: ${latest ? `${fmt(latest.kg, 1)} kg on ${esc(latest.date)}` : "none"}</p>
    <section class="card" style="margin-bottom:1rem">
      <h2>Add weigh-in</h2>
      <div class="row">
        ${dateField("w-date", state.date)}
        <label class="field tight"><span>kg</span><input id="w-kg" inputmode="decimal"></label>
        <label class="field tight"><span>RHR</span><input id="w-rhr" inputmode="numeric"></label>
        <label class="field grow"><span>Note</span><input id="w-note"></label>
        <label class="check"><input type="checkbox" id="w-protocol" checked> protocol</label>
        <button class="btn primary" id="add-weight">Add</button>
      </div>
    </section>
    <div class="card">
      <table>
        <thead><tr><th>Date</th><th>kg</th><th>RHR</th><th>Protocol</th><th>Note</th><th></th></tr></thead>
        <tbody>${rows || `<tr><td colspan="6" class="muted">No weigh-ins</td></tr>`}</tbody>
      </table>
    </div>
  </div>`);
}

function bind() {
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
  document.querySelectorAll("[data-tab]").forEach((b) => {
    b.addEventListener("click", async () => {
      state.tab = b.dataset.tab;
      if (state.tab === "days") await loadDays();
      if (state.tab === "weight") await loadWeights();
      if (state.tab === "foods") await loadFoods();
      if (state.tab === "stock") await loadStock();
      render();
    });
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
        await loadDay();
        render();
      }
    });
  });

  document.querySelectorAll("[data-field]").forEach((inp) => {
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
        const target2 = draftTarget(inp.dataset.slot, inp.dataset.idx, inp.dataset.part);
        if (target2) {
          linkDraftItemStock(target2);
          linkDraftItemFood(target2);
        }
        render();
        const again = document.querySelector(
          `[data-field="name"][data-slot="${inp.dataset.slot}"][data-idx="${inp.dataset.idx}"]${inp.dataset.part == null ? ":not([data-part])" : `[data-part="${inp.dataset.part}"]`}`
        );
        if (again) {
          again.focus();
          const v = again.value;
          again.setSelectionRange(v.length, v.length);
        }
      }
    });
  });
  document.querySelectorAll("[data-food-pick]").forEach((b) => {
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
  document.querySelectorAll("[data-resolve-food]").forEach((b) => {
    b.addEventListener("click", () => {
      const [foodID, slot, idx, part] = b.dataset.resolveFood.split(":");
      resolveFoodPick(foodID, slot, idx, part === "" ? null : part);
    });
  });
  document.querySelectorAll("[data-pick-food]").forEach((b) => {
    b.addEventListener("click", () => {
      const food = state.foods.find((f) => f.id === b.dataset.pickFood);
      const target = draftTarget(b.dataset.slot, b.dataset.idx, b.dataset.part);
      if (food && target) {
        target.name = food.name;
        target.food_id = food.id;
        target.kcal_per_100g = "";
        target.sodium_mg_per_100g = "";
      }
      state.suggest = { slot: "", i: -1, p: -1, q: "" };
      render();
    });
  });
  document.querySelectorAll("[data-add]").forEach((b) => {
    b.addEventListener("click", () => {
      state.mealDraft[b.dataset.add].items.push(emptyItem());
      scheduleSaveDay();
      render();
    });
  });
  document.querySelectorAll("[data-add-dish]").forEach((b) => {
    b.addEventListener("click", () => {
      const slot = b.dataset.addDish;
      const it = emptyItem();
      it.name = defaultSetName(slot);
      it.parts = [];
      it.is_set = true;
      state.mealDraft[slot].items.push(it);
      scheduleSaveDay();
      render();
    });
  });
  document.querySelectorAll("[data-make-dish]").forEach((b) => {
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
  document.querySelectorAll("[data-eject-part]").forEach((b) => {
    b.addEventListener("click", () => {
      const [slot, i, j] = b.dataset.ejectPart.split(":");
      ejectPartFromSet(slot, Number(i), Number(j));
      scheduleSaveDay();
      render();
    });
  });
  document.querySelectorAll("[data-add-part]").forEach((b) => {
    b.addEventListener("click", () => {
      const [slot, i] = b.dataset.addPart.split(":");
      const it = state.mealDraft[slot].items[Number(i)];
      if (!it.parts) it.parts = [];
      it.parts.push(emptyItem());
      scheduleSaveDay();
      render();
    });
  });
  document.querySelectorAll("[data-into-set]").forEach((sel) => {
    sel.addEventListener("change", () => {
      const v = sel.value;
      if (!v) return;
      const [slot, i] = sel.dataset.intoSet.split(":");
      absorbIntoSet(slot, Number(v), Number(i));
      scheduleSaveDay();
      render();
    });
  });
  document.querySelectorAll("[data-del]").forEach((b) => {
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
  document.querySelectorAll("[data-del-part]").forEach((b) => {
    b.addEventListener("click", () => {
      const [slot, i, j] = b.dataset.delPart.split(":");
      const it = state.mealDraft[slot].items[Number(i)];
      it.parts.splice(Number(j), 1);
      recomputeSetGrams(it);
      scheduleSaveDay();
      render();
    });
  });
  document.querySelectorAll("[data-note]").forEach((inp) => {
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
  document.querySelectorAll("[data-open]").forEach((tr) => {
    tr.addEventListener("click", async () => {
      state.date = tr.dataset.open;
      state.tab = "today";
      await loadDay();
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
  document.getElementById("stock-save")?.addEventListener("click", () => { saveStock().catch((e) => toast(e.message)); });
  function cancelStockForm() {
    state.stockForm = emptyStockForm();
    delete state.mealDraft?.stock;
    render();
  }
  document.getElementById("stock-clear")?.addEventListener("click", cancelStockForm);
  document.getElementById("stock-cancel-edit")?.addEventListener("click", cancelStockForm);
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
  document.querySelectorAll("[data-stock-household]").forEach((b) => {
    b.addEventListener("click", () => {
      adjustStock(b.dataset.stockHousehold, "household").catch((e) => toast(e.message));
    });
  });
  document.querySelectorAll("[data-stock-discard]").forEach((b) => {
    b.addEventListener("click", () => {
      adjustStock(b.dataset.stockDiscard, "discarded").catch((e) => toast(e.message));
    });
  });
  document.querySelectorAll("[data-edit-stock]").forEach((b) => {
    b.addEventListener("click", () => {
      const lot = state.stock.find((x) => x.id === b.dataset.editStock);
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
      render();
      window.scrollTo(0, 0);
    });
  });
  document.querySelectorAll("[data-del-stock]").forEach((b) => {
    b.addEventListener("click", () => {
      deleteStock(b.dataset.delStock).catch((e) => toast(e.message));
    });
  });
  document.getElementById("add-food")?.addEventListener("click", addFood);
  document.querySelectorAll("[data-del-food]").forEach((b) => {
    b.addEventListener("click", () => delFood(b.dataset.delFood));
  });
  document.querySelectorAll("[data-edit-food]").forEach((b) => {
    b.addEventListener("click", () => editFood(b.dataset.editFood));
  });

  document.getElementById("add-weight")?.addEventListener("click", addWeight);
  document.querySelectorAll("[data-del-w]").forEach((b) => {
    b.addEventListener("click", () => delWeight(b.dataset.delW));
  });
  document.querySelectorAll("[data-edit-w]").forEach((b) => {
    b.addEventListener("click", () => editWeight(b.dataset.editW));
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

function serializeItem(it) {
  const name = (it.name || "").trim();
  const parts = (it.parts || []).map(serializeItem).filter(Boolean);
  if (!name && !parts.length) return null;
  const food = knownFood(name);
  const row = { name: name || "Dish", grams: numOrUndef(it.grams) || 0 };
  if (it.is_set) {
    const { grams } = setPartStats(it);
    row.grams = grams;
  }
  if (food) row.food_id = food.id || it.food_id;
  else if (it.food_id) row.food_id = it.food_id;
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

async function addFood() {
  const name = document.getElementById("new-food-name").value.trim();
  const kcal = numOrUndef(document.getElementById("new-food-kcal").value);
  const na = numOrUndef(document.getElementById("new-food-na").value);
  const notes = document.getElementById("new-food-notes").value.trim();
  try {
    await api("/foods/batch", {
      method: "POST",
      body: JSON.stringify({ upsert: [{ name, kcal_per_100g: kcal, sodium_mg_per_100g: na, notes }] }),
    });
    await loadFoods();
    toast("Food saved");
    render();
  } catch (e) {
    toast(e.message);
  }
}

async function delFood(id) {
  if (!confirm("Delete this food?")) return;
  try {
    await api(`/foods/${id}`, { method: "DELETE" });
    await loadFoods();
    render();
  } catch (e) {
    toast(e.message);
  }
}

async function editFood(id) {
  const f = state.foods.find((x) => x.id === id);
  if (!f) return;
  const name = prompt("Name", f.name);
  if (name == null) return;
  const kcal = prompt("kcal / 100g", f.kcal_per_100g ?? "");
  if (kcal == null) return;
  const na = prompt("sodium mg / 100g", f.sodium_mg_per_100g ?? "");
  if (na == null) return;
  const notes = prompt("Notes", f.notes || "");
  if (notes == null) return;
  try {
    await api(`/foods/${id}`, {
      method: "PUT",
      body: JSON.stringify({
        name,
        kcal_per_100g: numOrUndef(kcal),
        sodium_mg_per_100g: numOrUndef(na),
        notes,
      }),
    });
    await loadFoods();
    render();
  } catch (e) {
    toast(e.message);
  }
}

async function addWeight() {
  const kg = numOrUndef(document.getElementById("w-kg").value);
  const rhr = numOrUndef(document.getElementById("w-rhr").value);
  const date = parseISODate(document.getElementById("w-date").value);
  if (!date) {
    toast("Date must be YYYY-MM-DD");
    return;
  }
  try {
    await api("/log", {
      method: "POST",
      body: JSON.stringify({
        date,
        weight: {
          kg,
          note: document.getElementById("w-note").value,
          protocol: document.getElementById("w-protocol").checked,
          rhr_bpm: rhr != null ? Math.round(rhr) : undefined,
        },
      }),
    });
    await loadWeights();
    toast("Weigh-in saved");
    render();
  } catch (e) {
    toast(e.message);
  }
}

async function delWeight(id) {
  if (!confirm("Delete this weigh-in?")) return;
  try {
    await api(`/weight/${id}`, { method: "DELETE" });
    await loadWeights();
    render();
  } catch (e) {
    toast(e.message);
  }
}

async function editWeight(id) {
  const w = state.weights.find((x) => x.id === id);
  if (!w) return;
  const kg = prompt("kg", w.kg);
  if (kg == null) return;
  const note = prompt("Note", w.note || "");
  if (note == null) return;
  try {
    await api(`/weight/${id}`, {
      method: "PUT",
      body: JSON.stringify({ date: w.date, kg: numOrUndef(kg), note, protocol: w.protocol, rhr_bpm: w.rhr_bpm }),
    });
    await loadWeights();
    render();
  } catch (e) {
    toast(e.message);
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
  } catch (e) {
    if (e.status === 404) state.day = null;
    else throw e;
  }
  fillDraftFromDay(state.day);
}

function syncStockFormFromDOM() {
  ensureStockMealDraft();
  const f = state.stockForm;
  f.set = state.mealDraft.stock.items[0];
  f.prepared_date = document.getElementById("stock-date")?.value?.trim() || f.prepared_date;
  f.batch_grams = document.getElementById("stock-batch")?.value ?? f.batch_grams;
  f.remaining_grams = document.getElementById("stock-left")?.value ?? f.remaining_grams;
  f.active = document.getElementById("stock-active")?.checked !== false;
  f.notes = document.getElementById("stock-notes")?.value ?? f.notes;
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
    return;
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
  render();
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
      household: "Household ate",
      discarded: "Discard from stock",
    };
    const overlay = el(`<div class="stock-dialog-overlay" role="dialog" aria-modal="true">
      <div class="stock-dialog">
        <h3>${esc(titles[kind] || "Stock")}</h3>
        <p class="stock-dialog-sub">${esc(lot.name)} — <span class="mono">${fmt(left, 0)} g</span> left</p>
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
  return new Promise((resolve) => {
    if (!lot) {
      resolve(false);
      return;
    }
    const left = Number(lot.remaining_grams) || 0;
    const batch = Number(lot.batch_grams) || 0;
    const qty = batch > 0 ? `${fmt(left, 0)}/${fmt(batch, 0)} g` : `${fmt(left, 0)} g`;
    const overlay = el(`<div class="stock-dialog-overlay" role="dialog" aria-modal="true">
      <div class="stock-dialog">
        <h3>Delete batch?</h3>
        <p class="stock-dialog-sub">${esc(lot.name)} <span class="mono">(${qty})</span></p>
        <p class="muted" style="font-size:0.82rem;margin:0 0 0.75rem">Removes the lot and its usage history. This cannot be undone.</p>
        <div class="row">
          <button type="button" class="btn" data-sd-cancel>Cancel</button>
          <button type="button" class="btn danger" data-sd-delete>Delete</button>
        </div>
      </div>
    </div>`);
    const close = (ok) => {
      overlay.remove();
      resolve(ok);
    };
    overlay.querySelector("[data-sd-cancel]")?.addEventListener("click", () => close(false));
    overlay.querySelector("[data-sd-delete]")?.addEventListener("click", () => close(true));
    overlay.addEventListener("keydown", (e) => {
      if (e.key === "Escape") close(false);
    });
    document.body.appendChild(overlay);
    overlay.querySelector("[data-sd-cancel]")?.focus();
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
  if (!ok) return;
  await api(`/stock/${id}`, { method: "DELETE" });
  if (state.stockForm?.id === id) {
    state.stockForm = emptyStockForm();
    delete state.mealDraft?.stock;
  }
  await loadStock();
  toast("Batch deleted");
  render();
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
  toast(mode === "discarded" ? `Discarded ${ans.grams} g` : `Household ${ans.grams} g (no meal log)`);
  render();
}

async function loadDays() {
  const q = state.daysQ ? `&q=${encodeURIComponent(state.daysQ)}` : "";
  const data = await api(`/days?limit=200${q}`);
  state.days = data.days || [];
}

async function loadWeights() {
  const data = await api("/weight");
  state.weights = data.weights || [];
  state.overview = await api("/overview?days=1");
}

async function boot() {
  state.overview = await api("/overview");
  if (state.overview?.today) state.date = parseISODate(state.overview.today) || state.overview.today;
  await loadFoods();
  await loadStock();
  await loadWeights();
  await loadDay();
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
