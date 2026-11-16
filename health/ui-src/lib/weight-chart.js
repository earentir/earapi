import { fmt } from "./util.js";

function weightSeries(weights) {
  return (weights || [])
    .filter((w) => w?.date && Number(w.kg) > 0)
    .slice()
    .sort((a, b) => a.date.localeCompare(b.date));
}

/** DaisyUI card + SVG line chart; hover points show date + kg. */
export function mountWeightChart(host, weights) {
  if (!host) return;
  const series = weightSeries(weights);
  host.replaceChildren();
  if (series.length < 2) {
    host.innerHTML = `<p class="muted" style="margin:0;padding:0.5rem 0">Add at least two weigh-ins to see the trend line.</p>`;
    return;
  }

  const tip = document.createElement("div");
  tip.className = "weight-chart-tip";
  tip.setAttribute("role", "tooltip");
  tip.hidden = true;

  const svgNS = "http://www.w3.org/2000/svg";
  const svg = document.createElementNS(svgNS, "svg");
  svg.setAttribute("class", "weight-chart-svg");
  svg.setAttribute("viewBox", "0 0 640 220");
  svg.setAttribute("preserveAspectRatio", "none");
  svg.setAttribute("aria-label", "Weight trend");

  const pad = { l: 48, r: 20, t: 20, b: 36 };
  const W = 640;
  const H = 220;
  const innerW = W - pad.l - pad.r;
  const innerH = H - pad.t - pad.b;

  const kgs = series.map((s) => Number(s.kg));
  const minK = Math.min(...kgs);
  const maxK = Math.max(...kgs);
  const padK = Math.max((maxK - minK) * 0.08, 0.3);
  const yMin = minK - padK;
  const yMax = maxK + padK;
  const yRange = yMax - yMin || 1;

  const xAt = (i) => pad.l + (series.length === 1 ? innerW / 2 : (i / (series.length - 1)) * innerW);
  const yAt = (kg) => pad.t + (1 - (kg - yMin) / yRange) * innerH;

  const grid = document.createElementNS(svgNS, "g");
  grid.setAttribute("class", "weight-chart-grid");
  for (let i = 0; i <= 4; i++) {
    const y = pad.t + (i / 4) * innerH;
    const line = document.createElementNS(svgNS, "line");
    line.setAttribute("x1", String(pad.l));
    line.setAttribute("x2", String(W - pad.r));
    line.setAttribute("y1", String(y));
    line.setAttribute("y2", String(y));
    grid.appendChild(line);
    const val = yMax - (i / 4) * yRange;
    const label = document.createElementNS(svgNS, "text");
    label.setAttribute("x", String(pad.l - 6));
    label.setAttribute("y", String(y + 4));
    label.setAttribute("class", "weight-chart-axis");
    label.setAttribute("text-anchor", "end");
    label.textContent = fmt(val, 1);
    grid.appendChild(label);
  }
  svg.appendChild(grid);

  const poly = document.createElementNS(svgNS, "polyline");
  poly.setAttribute("class", "weight-chart-line");
  poly.setAttribute("fill", "none");
  poly.setAttribute(
    "points",
    series.map((s, i) => `${xAt(i)},${yAt(Number(s.kg))}`).join(" "),
  );
  svg.appendChild(poly);

  const dots = document.createElementNS(svgNS, "g");
  dots.setAttribute("class", "weight-chart-points");
  series.forEach((s, i) => {
    const g = document.createElementNS(svgNS, "g");
    g.setAttribute("class", "weight-chart-point");
    g.setAttribute("tabindex", "0");
    const cx = xAt(i);
    const cy = yAt(Number(s.kg));
    const hit = document.createElementNS(svgNS, "circle");
    hit.setAttribute("cx", String(cx));
    hit.setAttribute("cy", String(cy));
    hit.setAttribute("r", "12");
    hit.setAttribute("class", "weight-chart-hit");
    const dot = document.createElementNS(svgNS, "circle");
    dot.setAttribute("cx", String(cx));
    dot.setAttribute("cy", String(cy));
    dot.setAttribute("r", "4");
    dot.setAttribute("class", "weight-chart-dot");
    g.appendChild(hit);
    g.appendChild(dot);
    const showTip = (evt) => {
      tip.hidden = false;
      tip.textContent = `${s.date} — ${fmt(s.kg, 1)} kg`;
      const hostRect = host.getBoundingClientRect();
      const x = evt?.clientX != null ? evt.clientX - hostRect.left : cx;
      const y = evt?.clientY != null ? evt.clientY - hostRect.top : cy;
      tip.style.left = `${Math.min(Math.max(x, 8), hostRect.width - 8)}px`;
      tip.style.top = `${Math.max(y - 36, 4)}px`;
    };
    const hideTip = () => {
      tip.hidden = true;
    };
    g.addEventListener("mouseenter", showTip);
    g.addEventListener("mousemove", showTip);
    g.addEventListener("mouseleave", hideTip);
    g.addEventListener("focus", () => {
      const r = dot.getBoundingClientRect();
      showTip({ clientX: r.left + r.width / 2, clientY: r.top });
    });
    g.addEventListener("blur", hideTip);
    dots.appendChild(g);
  });
  svg.appendChild(dots);

  const xLabels = document.createElementNS(svgNS, "g");
  xLabels.setAttribute("class", "weight-chart-xlabels");
  const labelIdx = new Set([0, series.length - 1]);
  if (series.length > 2) labelIdx.add(Math.floor(series.length / 2));
  labelIdx.forEach((i) => {
    const t = document.createElementNS(svgNS, "text");
    t.setAttribute("x", String(xAt(i)));
    t.setAttribute("y", String(H - 8));
    t.setAttribute("class", "weight-chart-axis");
    t.setAttribute("text-anchor", "middle");
    t.textContent = series[i].date.slice(5);
    xLabels.appendChild(t);
  });
  svg.appendChild(xLabels);

  host.appendChild(svg);
  host.appendChild(tip);
}

export function weightChartSectionHtml() {
  return `<section class="card weight-chart-card">
    <h2 style="margin:0 0 0.5rem;font-size:1rem">Weight trend</h2>
    <div class="weight-chart-host" data-weight-chart></div>
  </section>`;
}