import { fmt } from "./util.js";
const CHART_HOST = "w-full h-48 min-h-40 relative";

function cssColor(varName) {
  return getComputedStyle(document.documentElement).getPropertyValue(varName).trim() || "currentColor";
}

function weightSeries(weights) {
  return (weights || [])
    .filter((w) => w?.date && Number(w.kg) > 0)
    .slice()
    .sort((a, b) => a.date.localeCompare(b.date));
}

/** DaisyUI card + SVG line chart; hover points show date + kg. */
export function mountWeightChart(host, weights, opts = {}) {
  if (!host) return;
  let lastKey = "";
  const paint = () => {
    const key = `${host.clientWidth}x${host.clientHeight}`;
    if (key === lastKey && host.querySelector("svg, p")) return;
    lastKey = key;
    paintWeightChart(host, weights, opts);
  };
  paint();
  if (typeof ResizeObserver === "undefined") return;
  host._chartRO?.disconnect();
  const ro = new ResizeObserver(() => paint());
  host._chartRO = ro;
  ro.observe(host);
}

function paintWeightChart(host, weights, opts = {}) {
  const series = weightSeries(weights);
  host.replaceChildren();
  host.className = `${CHART_HOST}`;
  if (series.length < 2) {
    host.innerHTML = `<p class="text-base-content/60 m-0 py-2">Add at least two weigh-ins to see the trend line.</p>`;
    return;
  }

  const stroke = cssColor("--color-primary");
  const gridStroke = cssColor("--color-base-content");
  const gridOpacity = "0.15";

  const svgNS = "http://www.w3.org/2000/svg";
  const svg = document.createElementNS(svgNS, "svg");
  svg.setAttribute("class", "w-full h-full block");
  const pad = { l: 48, r: 20, t: 20, b: 36 };
  const W = Math.max(host.clientWidth || 640, 280);
  const H = Math.max(host.clientHeight || 192, 160);
  svg.setAttribute("viewBox", `0 0 ${W} ${H}`);
  svg.setAttribute("preserveAspectRatio", "none");
  svg.setAttribute("aria-label", "Weight trend");

  const innerW = Math.max(1, W - pad.l - pad.r);
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
  for (let i = 0; i <= 4; i++) {
    const y = pad.t + (i / 4) * innerH;
    const line = document.createElementNS(svgNS, "line");
    line.setAttribute("x1", String(pad.l));
    line.setAttribute("x2", String(W - pad.r));
    line.setAttribute("y1", String(y));
    line.setAttribute("y2", String(y));
    line.setAttribute("stroke", gridStroke);
    line.setAttribute("stroke-opacity", gridOpacity);
    grid.appendChild(line);
    const val = yMax - (i / 4) * yRange;
    const label = document.createElementNS(svgNS, "text");
    label.setAttribute("x", String(pad.l - 6));
    label.setAttribute("y", String(y + 4));
    label.setAttribute("fill", "currentColor");
    label.setAttribute("opacity", "0.55");
    label.setAttribute("font-size", "10");
    label.setAttribute("text-anchor", "end");
    label.textContent = fmt(val, 1);
    grid.appendChild(label);
  }
  svg.appendChild(grid);

  const poly = document.createElementNS(svgNS, "polyline");
  poly.setAttribute("fill", "none");
  poly.setAttribute("stroke", stroke);
  poly.setAttribute("stroke-width", "2");
  poly.setAttribute(
    "points",
    series.map((s, i) => `${xAt(i)},${yAt(Number(s.kg))}`).join(" "),
  );
  svg.appendChild(poly);

  const dots = document.createElementNS(svgNS, "g");
  series.forEach((s, i) => {
    const g = document.createElementNS(svgNS, "g");
    g.setAttribute("tabindex", "0");
    const cx = xAt(i);
    const cy = yAt(Number(s.kg));
    const hit = document.createElementNS(svgNS, "circle");
    hit.setAttribute("cx", String(cx));
    hit.setAttribute("cy", String(cy));
    hit.setAttribute("r", "12");
    hit.setAttribute("fill", "transparent");
    const dot = document.createElementNS(svgNS, "circle");
    dot.setAttribute("cx", String(cx));
    dot.setAttribute("cy", String(cy));
    dot.setAttribute("r", "4");
    dot.setAttribute("fill", stroke);
    g.appendChild(hit);
    g.appendChild(dot);
    const title = document.createElementNS(svgNS, "title");
    const bmr = opts.bmrForKg?.(s.kg);
    title.textContent = Number.isFinite(bmr)
      ? `${s.date}, ${fmt(s.kg, 1)} kg, BMR ${fmt(bmr, 0)}`
      : `${s.date}, ${fmt(s.kg, 1)} kg`;
    g.appendChild(title);
    dots.appendChild(g);
  });
  svg.appendChild(dots);

  const xLabels = document.createElementNS(svgNS, "g");
  const labelIdx = new Set([0, series.length - 1]);
  if (series.length > 2) labelIdx.add(Math.floor(series.length / 2));
  labelIdx.forEach((i) => {
    const t = document.createElementNS(svgNS, "text");
    t.setAttribute("x", String(xAt(i)));
    t.setAttribute("y", String(H - 8));
    t.setAttribute("fill", "currentColor");
    t.setAttribute("opacity", "0.55");
    t.setAttribute("font-size", "10");
    t.setAttribute("text-anchor", "middle");
    t.textContent = series[i].date.slice(5);
    xLabels.appendChild(t);
  });
  svg.appendChild(xLabels);

  host.appendChild(svg);
}

export function weightChartSectionHtml() {
  return `<section class="card card-bordered bg-base-100 shadow-sm border-base-300">
    <div class="card-body p-4 gap-2">
    <h2 class="card-title text-base">Weight trend</h2>
    <div class="${CHART_HOST}" data-weight-chart></div>
    </div>
  </section>`;
}