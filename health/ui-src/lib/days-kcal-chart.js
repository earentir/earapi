import { fmt } from "./util.js";
import { kcalBudgetBand, kcalBudgetBandLabel, mergedKcalBudget } from "./kcal-budget.js";

const BAND_FILL = {
  green: "var(--ok, #3ecf8e)",
  yellow: "#d4b429",
  orange: "#e6940a",
  red: "var(--warn, #f87171)",
  none: "color-mix(in oklab, var(--color-base-content) 35%, transparent)",
};

function bandForDay(d, budget) {
  const b = budget || mergedKcalBudget(null);
  const fromServer = d.display?.intake_bar_band;
  if (fromServer && fromServer !== "none") return fromServer;
  return kcalBudgetBand(d.totals?.kcal, b);
}

export function daysKcalChartSectionHtml() {
  return `<section class="card days-kcal-chart-card">
    <h2 style="margin:0 0 0.35rem;font-size:1rem">Daily kcal</h2>
    <p class="muted days-kcal-chart-legend" style="margin:0 0 0.5rem;font-size:0.78rem">
      Scale to gain line · bands match day cards (green below soft low, yellow in band, orange above soft, red at gain+)
    </p>
    <div class="days-kcal-chart-host" data-days-kcal-chart></div>
  </section>`;
}

export function mountDaysKcalChart(host, days, budgetRaw) {
  if (!host) return;
  const budget = mergedKcalBudget(budgetRaw);
  const series = (days || [])
    .map((d) => ({
      date: d.date,
      kcal: Number(d.totals?.kcal) || 0,
      band: bandForDay(d, budget),
    }))
    .filter((d) => d.date && d.kcal > 0)
    .sort((a, b) => a.date.localeCompare(b.date));

  host.replaceChildren();
  if (!series.length) {
    host.innerHTML = `<p class="muted" style="margin:0">No day totals to chart yet.</p>`;
    return;
  }

  const yMax = Math.max(
    budget.gain_min_kcal,
    ...series.map((s) => s.kcal),
  );
  const tip = document.createElement("div");
  tip.className = "days-kcal-chart-tip";
  tip.setAttribute("role", "tooltip");
  tip.hidden = true;

  const pad = { l: 44, r: 52, t: 12, b: 28 };
  const barW = 10;
  const gap = 3;
  const innerW = series.length * (barW + gap) - gap;
  const W = Math.max(640, pad.l + pad.r + innerW);
  const H = 200;
  const innerH = H - pad.t - pad.b;

  const svgNS = "http://www.w3.org/2000/svg";
  const svg = document.createElementNS(svgNS, "svg");
  svg.setAttribute("class", "days-kcal-chart-svg");
  svg.setAttribute("viewBox", `0 0 ${W} ${H}`);
  svg.setAttribute("preserveAspectRatio", "xMinYMid meet");
  svg.setAttribute("aria-label", "Daily kcal bar chart");

  const yAt = (kcal) => pad.t + (1 - kcal / yMax) * innerH;

  const limits = [
    { k: budget.soft_low_kcal, label: `${budget.soft_low_kcal}`, cls: "soft-low" },
    { k: budget.soft_high_kcal, label: `${budget.soft_high_kcal}`, cls: "soft-high" },
    { k: budget.gain_min_kcal, label: `${budget.gain_min_kcal} gain`, cls: "gain" },
  ];

  const guides = document.createElementNS(svgNS, "g");
  guides.setAttribute("class", "days-kcal-chart-guides");
  limits.forEach((lim) => {
    if (!lim.k || lim.k > yMax) return;
    const y = yAt(lim.k);
    const line = document.createElementNS(svgNS, "line");
    line.setAttribute("x1", String(pad.l));
    line.setAttribute("x2", String(W - pad.r));
    line.setAttribute("y1", String(y));
    line.setAttribute("y2", String(y));
    line.setAttribute("class", `days-kcal-guide days-kcal-guide--${lim.cls}`);
    guides.appendChild(line);
    const lab = document.createElementNS(svgNS, "text");
    lab.setAttribute("x", String(W - pad.r + 4));
    lab.setAttribute("y", String(y + 3));
    lab.setAttribute("class", "days-kcal-guide-label");
    lab.textContent = lim.label;
    guides.appendChild(lab);
  });
  svg.appendChild(guides);

  const yAxis = document.createElementNS(svgNS, "g");
  yAxis.setAttribute("class", "days-kcal-chart-axis-group");
  [0, yMax].forEach((k) => {
    const y = yAt(k);
    const t = document.createElementNS(svgNS, "text");
    t.setAttribute("x", String(pad.l - 6));
    t.setAttribute("y", String(y + 3));
    t.setAttribute("class", "days-kcal-chart-axis");
    t.setAttribute("text-anchor", "end");
    t.textContent = fmt(k, 0);
    yAxis.appendChild(t);
  });
  svg.appendChild(yAxis);

  const bars = document.createElementNS(svgNS, "g");
  bars.setAttribute("class", "days-kcal-bars");
  series.forEach((s, i) => {
    const x = pad.l + i * (barW + gap);
    const y0 = yAt(0);
    const y1 = yAt(s.kcal);
    const h = Math.max(2, y0 - y1);
    const g = document.createElementNS(svgNS, "g");
    g.setAttribute("class", "days-kcal-bar-group");
    g.setAttribute("tabindex", "0");
    const rect = document.createElementNS(svgNS, "rect");
    rect.setAttribute("x", String(x));
    rect.setAttribute("y", String(y1));
    rect.setAttribute("width", String(barW));
    rect.setAttribute("height", String(h));
    rect.setAttribute("rx", "2");
    rect.setAttribute("class", `days-kcal-bar-fill days-kcal-bar-fill--${s.band}`);
    rect.setAttribute("fill", BAND_FILL[s.band] || BAND_FILL.none);
    g.appendChild(rect);

    const showTip = (evt) => {
      const bandLabel = kcalBudgetBandLabel(s.band) || s.band;
      tip.hidden = false;
      tip.textContent = `${s.date} · ${fmt(s.kcal, 0)} kcal — ${bandLabel}`;
      const hostRect = host.getBoundingClientRect();
      const xPos = evt?.clientX != null ? evt.clientX - hostRect.left : x;
      const yPos = evt?.clientY != null ? evt.clientY - hostRect.top : y1;
      tip.style.left = `${Math.min(Math.max(xPos, 12), hostRect.width - 12)}px`;
      tip.style.top = `${Math.max(yPos - 40, 4)}px`;
    };
    const hideTip = () => {
      tip.hidden = true;
    };
    g.addEventListener("mouseenter", showTip);
    g.addEventListener("mousemove", showTip);
    g.addEventListener("mouseleave", hideTip);
    g.addEventListener("focus", () => {
      const r = rect.getBoundingClientRect();
      showTip({ clientX: r.left + r.width / 2, clientY: r.top });
    });
    g.addEventListener("blur", hideTip);
    bars.appendChild(g);
  });
  svg.appendChild(bars);

  host.appendChild(svg);
  host.appendChild(tip);
}