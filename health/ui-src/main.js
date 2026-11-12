import { loadTheme } from "./lib/theme.js";
import { setRender, toast } from "./lib/api.js";
import { render, bootIfAuthed, onHistoryNav, openDayFromDaysList } from "./lib/render.js";

loadTheme();
setRender(render);
bootIfAuthed();

const appEl = document.getElementById("app");
appEl?.addEventListener("click", (e) => {
  if (e.target.closest?.(".day-card-status--hoverable")) return;
  if (e.target.closest?.(".day-legacy-hover")) return;
  const card = e.target.closest?.("[data-open-day]");
  if (!card?.dataset.openDay) return;
  e.preventDefault();
  openDayFromDaysList(card.dataset.openDay).catch((err) => toast(err.message));
});
appEl?.addEventListener("keydown", (e) => {
  if (e.key !== "Enter" && e.key !== " ") return;
  if (e.target.closest?.(".day-legacy-hover")) return;
  const card = e.target.closest?.("[data-open-day]");
  if (!card?.dataset.openDay) return;
  e.preventDefault();
  openDayFromDaysList(card.dataset.openDay).catch((err) => toast(err.message));
});

window.addEventListener("popstate", () => {
  onHistoryNav().catch((e) => console.error(e));
});
