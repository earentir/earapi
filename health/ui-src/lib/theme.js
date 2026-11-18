import { THEME_KEY, THEMES } from "./constants.js";

export function loadTheme() {
  const saved = localStorage.getItem(THEME_KEY) || "dark";
  const theme = THEMES.includes(saved) ? saved : "dark";
  document.documentElement.dataset.theme = theme;
  return theme;
}

export function saveTheme(theme) {
  if (!THEMES.includes(theme)) return;
  localStorage.setItem(THEME_KEY, theme);
  document.documentElement.dataset.theme = theme;
}

export function themeSelectHtml(current) {
  const opts = THEMES.map(
    (t) => `<option value="${t}"${t === current ? " selected" : ""}>${t}</option>`
  ).join("");
  return `<select class="select select-bordered select-sm w-28 max-w-full" id="theme-pick" title="Theme" aria-label="Theme">${opts}</select>`;
}
