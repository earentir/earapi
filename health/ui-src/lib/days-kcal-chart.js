import { fmt } from "./util.js";
import { dayIntakeBand, mergedKcalBudget } from "./kcal-budget.js";
const CHART_HOST = "w-full h-48 min-h-40 relative";

function cssColor(varName) {
  return getComputedStyle(document.documentElement).getPropertyValue(varName).trim() || "currentColor";
}

const BAND_FILL = {
  green: () => cssColor("--color-success"),
  yellow: () => cssColor("--color-warning"),
  orange: () => cssColor("--color-warning"),
  red: () => cssColor("--color-error"),
  none: () => cssColor("--color-base-content"),
};

export function daysKcalChartSectionHtml() {
  return `<section class="card card-bordered bg-base-100 shadow-sm border-base-300">
    <div class="card-body p-4 gap-2">
    <h2 class="card-title text-base">Daily kcal</h2>
    <div class="${CHART_HOST}" data-days-kcal-chart></div>
    </div>
  </section>`;
}

export function mountDaysKcalChart(host, days, budgetRaw, sedentaryKcal) {
  if (!host) return;
  let lastKey = "";
  const paint = () => {
    const sed = Number(sedentaryKcal) || 0;
    const key = `${host.clientWidth}x${host.clientHeight}x${sed}`;
    if (key === lastKey && host.querySelector("svg, p")) return;
    lastKey = key;
    paintDaysKcalChart(host, days, budgetRaw, sed);
  };
  paint();
  watchChartHost(host, paint);
}

function paintDaysKcalChart(host, days, budgetRaw, sedentaryKcal = 0) {
  const budget = mergedKcalBudget(budgetRaw);
  const series = (days || [])
    .map((d) => ({
      date: d.date,
      kcal: Number(d.totals?.kcal) || 0,
      band: dayIntakeBand(d, budget),
    }))
    .filter((d) => d.date && d.kcal > 0)
    .sort((a, b) => a.date.localeCompare(b.date));

  host.replaceChildren();
  if (!series.length) {
    host.innerHTML = `<p class="text-sm text-base-content/60 m-0">No day totals to chart yet.</p>`;
    return;
  }

  const yMax = Math.max(
    budget.gain_min_kcal,
    sedentaryKcal > 0 ? sedentaryKcal : 0,
    ...series.map((s) => s.kcal),
  );
  const pad = { l: 44, r: 52, t: 12, b: 28 };
  const W = Math.max(host.clientWidth || 640, 280);
  const H = Math.max(host.clientHeight || 192, 160);
  const innerW = Math.max(1, W - pad.l - pad.r);
  const innerH = H - pad.t - pad.b;
  const slot = innerW / series.length;
  const gap = Math.max(1, Math.min(10, slot * 0.18));
  const barW = Math.max(1, slot - gap);

  const svgNS = "http://www.w3.org/2000/svg";
  const svg = document.createElementNS(svgNS, "svg");
  svg.setAttribute("class", "w-full h-full block");
  host.className = CHART_HOST;
  svg.setAttribute("viewBox", `0 0 ${W} ${H}`);
  svg.setAttribute("preserveAspectRatio", "none");
  svg.setAttribute("aria-label", "Daily kcal bar chart");

  const yAt = (kcal) => pad.t + (1 - kcal / yMax) * innerH;

  const limits = [
    { k: budget.soft_low_kcal, label: `${budget.soft_low_kcal}`, cls: "soft-low" },
    { k: budget.soft_high_kcal, label: `${budget.soft_high_kcal}`, cls: "soft-high" },
    ...(sedentaryKcal > 0 ? [{ k: sedentaryKcal, label: `${fmt(sedentaryKcal, 0)} sed`, cls: "sedentary" }] : []),
    { k: budget.gain_min_kcal, label: `${budget.gain_min_kcal} gain`, cls: "gain" },
  ];

  const guides = document.createElementNS(svgNS, "g");
  limits.forEach((lim) => {
    if (!lim.k || lim.k > yMax) return;
    const y = yAt(lim.k);
    const line = document.createElementNS(svgNS, "line");
    line.setAttribute("x1", String(pad.l));
    line.setAttribute("x2", String(W - pad.r));
    line.setAttribute("y1", String(y));
    line.setAttribute("y2", String(y));
    line.setAttribute("stroke", cssColor("--color-base-content"));
    line.setAttribute("stroke-opacity", lim.cls === "gain" ? "0.45" : "0.22");
    line.setAttribute("stroke-dasharray", lim.cls === "sedentary" ? "4 3" : "6 4");
    guides.appendChild(line);
    const lab = document.createElementNS(svgNS, "text");
    lab.setAttribute("x", String(W - pad.r + 4));
    lab.setAttribute("y", String(y + 3));
    lab.setAttribute("fill", "currentColor");
    lab.setAttribute("opacity", "0.55");
    lab.setAttribute("font-size", "9");
    lab.textContent = lim.label;
    guides.appendChild(lab);
  });
  svg.appendChild(guides);

  const yAxis = document.createElementNS(svgNS, "g");
  [0, yMax].forEach((k) => {
    const y = yAt(k);
    const t = document.createElementNS(svgNS, "text");
    t.setAttribute("x", String(pad.l - 6));
    t.setAttribute("y", String(y + 3));
    t.setAttribute("fill", "currentColor");
    t.setAttribute("opacity", "0.55");
    t.setAttribute("font-size", "10");
    t.setAttribute("text-anchor", "end");
    t.textContent = fmt(k, 0);
    yAxis.appendChild(t);
  });
  svg.appendChild(yAxis);

  const bars = document.createElementNS(svgNS, "g");
  series.forEach((s, i) => {
    const x = pad.l + i * slot + (slot - barW) / 2;
    const y0 = yAt(0);
    const y1 = yAt(s.kcal);
    const h = Math.max(2, y0 - y1);
    const g = document.createElementNS(svgNS, "g");
    g.setAttribute("tabindex", "0");
    const rect = document.createElementNS(svgNS, "rect");
    rect.setAttribute("x", String(x));
    rect.setAttribute("y", String(y1));
    rect.setAttribute("width", String(barW));
    rect.setAttribute("height", String(h));
    rect.setAttribute("rx", String(barW > 8 ? 2 : 1));
    const fillFn = BAND_FILL[s.band] || BAND_FILL.none;
    rect.setAttribute("fill", fillFn());
    g.appendChild(rect);
    const title = document.createElementNS(svgNS, "title");
    title.textContent = `${s.date}, ${fmt(s.kcal, 0)} kcal`;
    g.appendChild(title);
    bars.appendChild(g);
  });
  svg.appendChild(bars);

  host.appendChild(svg);
}

function watchChartHost(host, paint) {
  if (typeof ResizeObserver === "undefined") return;
  host._chartRO?.disconnect();
  const ro = new ResizeObserver(() => paint());
  host._chartRO = ro;
  ro.observe(host);
}
