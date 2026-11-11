const AUTH_KEY = "earapi-health-auth";
const SLOTS = ["lunch", "dinner", "snack"];

const state = {
  user: "",
  token: "",
  tab: "today",
  date: "",
  foods: [],
  day: null,
  days: [],
  daysQ: "",
  weights: [],
  overview: null,
  mealDraft: emptyDraft(),
  toast: "",
  busy: false,
  suggest: { slot: "", i: -1, q: "" },
};

function pad2(n) {
  return String(n).padStart(2, "0");
}

function isoDate(y, m, d) {
  return `${y}-${pad2(m)}-${pad2(d)}`;
}

function parseISODate(s) {
  const m = String(s || "").trim().match(/^(\d{4})-(\d{2})-(\d{2})$/);
  if (!m) return null;
  const y = Number(m[1]);
  const mo = Number(m[2]);
  const d = Number(m[3]);
  const dt = new Date(Date.UTC(y, mo - 1, d));
  if (dt.getUTCFullYear() !== y || dt.getUTCMonth() + 1 !== mo || dt.getUTCDate() !== d) return null;
  return isoDate(y, mo, d);
}

function shiftISODate(s, days) {
  const ok = parseISODate(s);
  if (!ok) return s;
  const [y, m, d] = ok.split("-").map(Number);
  const dt = new Date(Date.UTC(y, m - 1, d + days));
  return isoDate(dt.getUTCFullYear(), dt.getUTCMonth() + 1, dt.getUTCDate());
}

function parseTime24(s) {
  const t = String(s || "").trim();
  if (!t) return "";
  const m = t.match(/^(\d{1,2}):([0-5]\d)$/);
  if (!m) return null;
  const h = Number(m[1]);
  if (h > 23) return null;
  return `${pad2(h)}:${m[2]}`;
}

function dateField(id, value) {
  return `<label class="field tight"><span>Date</span>
    <div class="date-nav">
      <button type="button" class="btn" data-shift="${esc(id)}" data-delta="-1" title="Previous day">‹</button>
      <input class="iso-date mono" id="${esc(id)}" value="${esc(value)}" placeholder="YYYY-MM-DD" maxlength="10" spellcheck="false" autocomplete="off">
      <button type="button" class="btn" data-shift="${esc(id)}" data-delta="1" title="Next day">›</button>
    </div>
  </label>`;
}

function emptyItem() {
  return { name: "", grams: "", food_id: "", parts: [], kcal_per_100g: "", sodium_mg_per_100g: "", note: "" };
}

function emptySlot() {
  return { items: [emptyItem()], kcal_margin: "0.10", note: "" };
}

function emptyDraft() {
  return { lunch: emptySlot(), dinner: emptySlot(), snack: emptySlot(), bp: "", walking: "" };
}

function loadAuth() {
  try {
    const a = JSON.parse(localStorage.getItem(AUTH_KEY) || "null");
    if (a?.user && a?.token) {
      state.user = a.user;
      state.token = a.token;
    }
  } catch { /* ignore */ }
}

function saveAuth() {
  localStorage.setItem(AUTH_KEY, JSON.stringify({ user: state.user, token: state.token }));
}

async function api(path, opts = {}) {
  const headers = {
    Accept: "application/json",
    "X-User": state.user,
    "X-API-Key": state.token,
    ...(opts.body ? { "Content-Type": "application/json" } : {}),
    ...(opts.headers || {}),
  };
  const res = await fetch(`/health/v1${path}`, { ...opts, headers });
  const text = await res.text();
  let data = null;
  try { data = text ? JSON.parse(text) : null; } catch { data = { error: text }; }
  if (!res.ok) {
    const err = new Error(data?.error || res.statusText);
    err.status = res.status;
    err.data = data;
    throw err;
  }
  return data;
}

function toast(msg) {
  state.toast = msg;
  render();
  setTimeout(() => {
    if (state.toast === msg) {
      state.toast = "";
      render();
    }
  }, 2400);
}

function fmt(n, d = 0) {
  if (n == null || n === "") return "—";
  const x = Number(n);
  if (Number.isNaN(x)) return "—";
  return x.toFixed(d);
}

function el(html) {
  const t = document.createElement("template");
  t.innerHTML = html.trim();
  return t.content.firstElementChild;
}

function render() {
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

function esc(s) {
  return String(s ?? "").replace(/[&<>"']/g, (c) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
  }[c]));
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

function topbar() {
  return el(`<header class="topbar"><div class="topbar-inner">
    <div><span class="brand">health</span><span class="brand-sub">${esc(state.user)}</span></div>
    <nav class="nav">
      ${["today", "days", "foods", "weight"].map((t) =>
        `<button data-tab="${t}" class="${state.tab === t ? "active" : ""}">${t}</button>`
      ).join("")}
    </nav>
    <button class="btn ghost" id="logout">Log out</button>
  </div></header>`);
}

function content() {
  const wrap = el(`<main class="wrap"></main>`);
  if (state.tab === "today") wrap.append(todayView());
  if (state.tab === "days") wrap.append(daysView());
  if (state.tab === "foods") wrap.append(foodsView());
  if (state.tab === "weight") wrap.append(weightView());
  return wrap;
}

function todayView() {
  const d = state.mealDraft;
  const totals = state.day?.totals || { kcal: 0, sodium_mg: 0 };
  const box = el(`<div>
    <div class="row" style="margin-bottom:1rem">
      ${dateField("day-date", state.date)}
      <div class="totals" style="padding-bottom:0.2rem">
        <span>${fmt(totals.kcal, 0)} kcal</span>
        <span>${fmt(totals.sodium_mg, 0)} mg Na</span>
      </div>
      <button class="btn primary" id="save-day" ${state.busy ? "disabled" : ""}>Save day</button>
      <button class="btn danger" id="delete-day">Delete day</button>
    </div>
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

function knownFood(name) {
  const q = (name || "").trim().toLowerCase();
  if (!q) return null;
  return state.foods.find((f) => f.name.toLowerCase() === q) || null;
}

function slotCard(slot, draft) {
  const meal = state.day?.meals?.[slot];
  const summary = meal ? `${fmt(meal.kcal, 0)} kcal · ${fmt(meal.sodium_mg, 0)} mg` : "empty";
  const rows = (draft.items || []).map((it, i) => itemBlock(slot, i, it)).join("");
  return `<section class="card" data-slot="${slot}">
    <div class="slot-head">
      <h2 style="text-transform:capitalize;margin:0">${slot}</h2>
      <span class="muted mono">${summary}</span>
    </div>
    ${rows}
    <div class="row">
      <button class="btn" data-add="${slot}">Add item</button>
      <button class="btn" data-add-dish="${slot}">Add dish</button>
      <label class="field tight"><span>kcal margin</span><input data-margin="${slot}" value="${esc(draft.kcal_margin)}" placeholder="0.10"></label>
    </div>
  </section>`;
}

function suggestList(slot, i, partIdx, q) {
  const open = state.suggest.slot === slot && state.suggest.i === i
    && (state.suggest.p ?? -1) === (partIdx == null ? -1 : partIdx);
  const query = (q || "").trim().toLowerCase();
  if (!open || query.length < 1) return "";
  const hits = state.foods.filter((f) => f.name.toLowerCase().includes(query)).slice(0, 8);
  if (!hits.length) return "";
  const partAttr = partIdx == null ? "" : ` data-part="${partIdx}"`;
  return `<div class="suggest-list">${hits.map((f) =>
    `<button type="button" data-pick-food="${esc(f.id)}" data-slot="${esc(slot)}" data-idx="${i}"${partAttr}>${esc(f.name)} <span class="muted">${fmt(f.kcal_per_100g, 0)} kcal/100g</span></button>`
  ).join("")}</div>`;
}

function nameGramsRow(slot, i, it, partIdx) {
  const isPart = partIdx != null;
  const unknown = (it.name || "").trim() && !knownFood(it.name) && !it.food_id && !(it.parts && it.parts.length);
  const partAttr = isPart ? ` data-part="${partIdx}"` : "";
  const gramsLabel = isPart ? "batch g" : (it.parts && it.parts.length ? "eaten g" : "grams");
  return `<div class="item-row ${isPart ? "part-row" : ""}">
    <div class="suggest">
      <input data-field="name" data-slot="${esc(slot)}" data-idx="${i}"${partAttr} value="${esc(it.name)}" placeholder="${isPart ? "Ingredient" : "Food or dish"}" autocomplete="off">
      ${suggestList(slot, i, partIdx, it.name)}
    </div>
    <input data-field="grams" data-slot="${esc(slot)}" data-idx="${i}"${partAttr} value="${esc(it.grams)}" placeholder="${gramsLabel}" inputmode="decimal">
    ${isPart ? `<button class="btn ghost" data-del-part="${esc(slot)}:${i}:${partIdx}">×</button>` : `<button class="btn ghost" data-del="${esc(slot)}:${i}">×</button>`}
  </div>
  ${unknown ? `<div class="item-row new-food">
    <input data-field="kcal_per_100g" data-slot="${esc(slot)}" data-idx="${i}"${partAttr} value="${esc(it.kcal_per_100g)}" placeholder="kcal/100g (new food)">
    <input data-field="sodium_mg_per_100g" data-slot="${esc(slot)}" data-idx="${i}"${partAttr} value="${esc(it.sodium_mg_per_100g)}" placeholder="Na mg/100g">
    <span class="muted">not in catalog</span>
  </div>` : ""}`;
}

function itemBlock(slot, i, it) {
  const dish = Array.isArray(it.parts);
  const parts = (it.parts || []).map((p, j) => nameGramsRow(slot, i, p, j)).join("");
  return `<div class="item-block">
    ${nameGramsRow(slot, i, it, null)}
    ${dish ? `<div class="parts">${parts}
      <button class="btn ghost" data-add-part="${esc(slot)}:${i}">Add ingredient</button>
    </div>` : `<button class="btn ghost" data-make-dish="${esc(slot)}:${i}">Make dish</button>`}
  </div>`;
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
  if (items.length) return items.join(" · ");
  const raw = (m.raw || "").replace(/\s+/g, " ").trim();
  if (!raw || raw === "—" || /^[—\-]+$/.test(raw) || /^—\s*\(no /i.test(raw)) return "";
  return raw;
}

function mealLine(slot, d) {
  const m = d.meals?.[slot];
  const text = mealText(m);
  if (!text) return "";
  const tot = m && (m.kcal || m.sodium_mg)
    ? ` <span class="mono muted">${fmt(m.kcal, 0)} kcal · ${fmt(m.sodium_mg, 0)} mg</span>`
    : "";
  return `<div class="meal-line"><span class="slot-lab">${slot}</span><span>${esc(text)}${tot}</span></div>`;
}

function daysView() {
  const cards = (state.days || []).map((d) => {
    const meals = ["lunch", "dinner", "snack"].map((s) => mealLine(s, d)).join("");
    const meta = [d.bp ? `BP ${esc(d.bp)}` : "", d.walking ? esc(d.walking) : ""].filter(Boolean).join(" · ");
    return `<article class="day-card clickable" data-open="${esc(d.date)}">
      <div class="day-card-head">
        <span class="mono date-iso">${esc(d.date)}</span>
        <span class="mono">${fmt(d.totals?.kcal, 0)} kcal · ${fmt(d.totals?.sodium_mg, 0)} mg Na</span>
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
      if (inp.dataset.field === "name") {
        state.suggest = {
          slot: inp.dataset.slot,
          i: Number(inp.dataset.idx),
          p: inp.dataset.part == null ? -1 : Number(inp.dataset.part),
          q: inp.value,
        };
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
      render();
    });
  });
  document.querySelectorAll("[data-add-dish]").forEach((b) => {
    b.addEventListener("click", () => {
      const it = emptyItem();
      it.parts = [emptyItem()];
      state.mealDraft[b.dataset.addDish].items.push(it);
      render();
    });
  });
  document.querySelectorAll("[data-make-dish]").forEach((b) => {
    b.addEventListener("click", () => {
      const [slot, i] = b.dataset.makeDish.split(":");
      const it = state.mealDraft[slot].items[Number(i)];
      if (it && !Array.isArray(it.parts)) it.parts = [emptyItem()];
      render();
    });
  });
  document.querySelectorAll("[data-add-part]").forEach((b) => {
    b.addEventListener("click", () => {
      const [slot, i] = b.dataset.addPart.split(":");
      const it = state.mealDraft[slot].items[Number(i)];
      if (!it.parts) it.parts = [];
      it.parts.push(emptyItem());
      render();
    });
  });
  document.querySelectorAll("[data-del]").forEach((b) => {
    b.addEventListener("click", () => {
      const [slot, i] = b.dataset.del.split(":");
      state.mealDraft[slot].items.splice(Number(i), 1);
      if (!state.mealDraft[slot].items.length) state.mealDraft[slot].items.push(emptyItem());
      render();
    });
  });
  document.querySelectorAll("[data-del-part]").forEach((b) => {
    b.addEventListener("click", () => {
      const [slot, i, j] = b.dataset.delPart.split(":");
      const it = state.mealDraft[slot].items[Number(i)];
      it.parts.splice(Number(j), 1);
      render();
    });
  });
  document.querySelectorAll("[data-margin]").forEach((inp) => {
    inp.addEventListener("change", () => { state.mealDraft[inp.dataset.margin].kcal_margin = inp.value; });
  });
  document.querySelectorAll("[data-note]").forEach((inp) => {
    inp.addEventListener("change", () => { state.mealDraft[inp.dataset.note].note = inp.value; });
  });
  document.getElementById("day-bp")?.addEventListener("change", (e) => { state.mealDraft.bp = e.target.value; });
  document.getElementById("day-walking")?.addEventListener("change", (e) => { state.mealDraft.walking = e.target.value; });
  document.getElementById("save-day")?.addEventListener("click", saveDay);
  document.getElementById("delete-day")?.addEventListener("click", deleteDay);

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
  if (food) row.food_id = food.id || it.food_id;
  else if (it.food_id) row.food_id = it.food_id;
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

async function saveDay() {
  state.busy = true;
  render();
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
        kcal_margin: numOrUndef(state.mealDraft[slot].kcal_margin) || 0,
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
    fillDraftFromDay(state.day);
    if (out.foods_ensured?.length) await loadFoods();
    toast("Saved");
  } catch (e) {
    toast(e.message);
  } finally {
    state.busy = false;
    render();
  }
}

async function deleteDay() {
  if (!confirm(`Delete ${state.date}?`)) return;
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

function cloneDraftItem(it) {
  return {
    name: it.name || "",
    grams: it.grams || "",
    food_id: it.food_id || "",
    kcal_per_100g: "",
    sodium_mg_per_100g: "",
    note: it.note || "",
    parts: Array.isArray(it.parts) ? it.parts.map(cloneDraftItem) : undefined,
  };
}

function fillDraftFromDay(day) {
  const d = emptyDraft();
  if (day) {
    d.bp = day.bp || "";
    d.walking = day.walking || "";
    for (const slot of SLOTS) {
      const m = day.meals?.[slot];
      if (!m) continue;
      d[slot].kcal_margin = m.kcal_margin ? String(m.kcal_margin) : "0";
      d[slot].note = "";
      d[slot].items = (m.items && m.items.length)
        ? m.items.map(cloneDraftItem)
        : emptySlot().items;
    }
  }
  state.mealDraft = d;
}

async function loadFoods() {
  const data = await api("/foods");
  state.foods = data.foods || [];
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
  await loadDay();
  render();
}

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
