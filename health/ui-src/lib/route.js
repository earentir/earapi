import { isoDate, parseISODate } from "./util.js";

export const NAV_TABS = [
  { id: "today", label: "Today" },
  { id: "days", label: "Days" },
  { id: "foods", label: "Foods" },
  { id: "stock", label: "Stock" },
  { id: "weight", label: "Weight" },
  { id: "settings", label: "Settings" },
];

export const TAB_IDS = NAV_TABS.map((t) => t.id);

/** Base path of the tracker shell, e.g. /health/ui */
export function appRootPath() {
  const path = window.location.pathname;
  const marker = "/health/ui";
  const i = path.indexOf(marker);
  if (i === -1) return marker;
  return path.slice(0, i + marker.length);
}

export function pathForTab(tab, date) {
  const root = appRootPath();
  const id = TAB_IDS.includes(tab) ? tab : "today";
  if (id === "today") {
    const d = date && parseISODate(date);
    return d ? `${root}/today/${d}` : `${root}/today`;
  }
  return `${root}/${id}`;
}

export function parseAppRoute() {
  const root = appRootPath();
  let tail = window.location.pathname.slice(root.length).replace(/^\//, "");
  if (!tail || tail === "index.html") {
    return { tab: "today", date: null };
  }
  const [seg, extra] = tail.split("/");
  if (seg === "today") {
    const d = extra ? parseISODate(extra) : null;
    return { tab: "today", date: d };
  }
  if (TAB_IDS.includes(seg)) {
    return { tab: seg, date: null };
  }
  return { tab: "today", date: null };
}

/** Server calendar day from overview, else local date (YYYY-MM-DD). */
export function resolveTodayDate(state) {
  const o = state?.overview?.today;
  if (o) {
    const d = parseISODate(o);
    if (d) return d;
    if (/^\d{4}-\d{2}-\d{2}$/.test(String(o).trim())) return String(o).trim();
  }
  const now = new Date();
  return isoDate(now.getFullYear(), now.getMonth() + 1, now.getDate());
}

export function applyRouteToState(state) {
  const { tab, date } = parseAppRoute();
  state.tab = tab;
  if (tab === "today") {
    state.date = date || resolveTodayDate(state);
  } else if (date) {
    state.date = date;
  }
}

export function pushAppRoute(state, { replace = false } = {}) {
  const url = pathForTab(state.tab, state.tab === "today" ? state.date : null);
  const hist = { tab: state.tab, date: state.date };
  if (replace) history.replaceState(hist, "", url);
  else history.pushState(hist, "", url);
}
