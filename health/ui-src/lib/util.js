import { esc } from "./strings.js";

export { esc };

export function fmt(n, d = 0) {
  if (n == null || n === "") return "-";
  const x = Number(n);
  if (Number.isNaN(x)) return "-";
  return x.toFixed(d);
}

export function el(html) {
  const t = document.createElement("template");
  t.innerHTML = html.trim();
  return t.content.firstElementChild;
}

export function pad2(n) {
  return String(n).padStart(2, "0");
}

export function isoDate(y, m, d) {
  return `${y}-${pad2(m)}-${pad2(d)}`;
}

export function parseISODate(s) {
  const m = String(s || "").trim().match(/^(\d{4})-(\d{2})-(\d{2})$/);
  if (!m) return null;
  const y = Number(m[1]);
  const mo = Number(m[2]);
  const d = Number(m[3]);
  const dt = new Date(Date.UTC(y, mo - 1, d));
  if (dt.getUTCFullYear() !== y || dt.getUTCMonth() + 1 !== mo || dt.getUTCDate() !== d) return null;
  return isoDate(y, mo, d);
}

export function weekdayNameForISODate(s) {
  const ok = parseISODate(s);
  if (!ok) return "";
  const [y, m, d] = ok.split("-").map(Number);
  const dt = new Date(y, m - 1, d);
  return dt.toLocaleDateString("en-US", { weekday: "long" });
}

export function shiftISODate(s, days) {
  const ok = parseISODate(s);
  if (!ok) return s;
  const [y, m, d] = ok.split("-").map(Number);
  const dt = new Date(Date.UTC(y, m - 1, d + days));
  return isoDate(dt.getUTCFullYear(), dt.getUTCMonth() + 1, dt.getUTCDate());
}

export function parseTime24(s) {
  const t = String(s || "").trim();
  if (!t) return "";
  const m = t.match(/^(\d{1,2}):([0-5]\d)$/);
  if (!m) return null;
  const h = Number(m[1]);
  if (h > 23) return null;
  return `${pad2(h)}:${m[2]}`;
}

export function formatDateTyping(raw) {
  const digits = String(raw || "").replace(/\D/g, "").slice(0, 8);
  if (digits.length <= 4) return digits;
  if (digits.length <= 6) return `${digits.slice(0, 4)}-${digits.slice(4)}`;
  return `${digits.slice(0, 4)}-${digits.slice(4, 6)}-${digits.slice(6)}`;
}

export function formatTimeTyping(raw) {
  const cleaned = String(raw || "").replace(/[^\d:]/g, "");
  if (cleaned.includes(":")) {
    const [h, m = ""] = cleaned.split(":");
    const hh = h.replace(/\D/g, "").slice(0, 2);
    const mm = m.replace(/\D/g, "").slice(0, 2);
    if (mm.length === 0 && !cleaned.endsWith(":")) return hh;
    return `${hh}:${mm}`;
  }
  const digits = cleaned.replace(/\D/g, "").slice(0, 4);
  if (digits.length <= 2) return digits;
  return `${digits.slice(0, 2)}:${digits.slice(2)}`;
}

export function dateField(id, value, opts = {}) {
  const label = opts.label !== false;
  const labelHtml = label ? `<span>Date</span>` : "";
  return `<label class="field date-field${label ? "" : " date-field-plain"}">${labelHtml}
    <div class="date-nav">
      <button type="button" class="btn" data-shift="${esc(id)}" data-delta="-1" title="Previous day">‹</button>
      <input type="text" class="iso-date mono" id="${esc(id)}" value="${esc(value)}" placeholder="YYYY-MM-DD" maxlength="10" inputmode="numeric" spellcheck="false" autocomplete="off" lang="en">
      <button type="button" class="btn" data-shift="${esc(id)}" data-delta="1" title="Next day">›</button>
    </div>
  </label>`;
}

/** Wire text inputs to stay ISO date / 24h time (no native date/time pickers). */
export function bindIsoInputs(root = document) {
  root.querySelectorAll("input.iso-date").forEach((inp) => {
    inp.addEventListener("input", () => {
      const next = formatDateTyping(inp.value);
      if (next !== inp.value) {
        const pos = inp.selectionStart;
        inp.value = next;
        if (pos != null) inp.setSelectionRange(next.length, next.length);
      }
    });
    inp.addEventListener("blur", () => {
      const ok = parseISODate(inp.value);
      if (ok) inp.value = ok;
      else if (inp.value.trim()) inp.value = formatDateTyping(inp.value);
    });
  });
  root.querySelectorAll("input.iso-time").forEach((inp) => {
    inp.addEventListener("input", () => {
      const next = formatTimeTyping(inp.value);
      if (next !== inp.value) {
        inp.value = next;
        inp.setSelectionRange(next.length, next.length);
      }
    });
    inp.addEventListener("blur", () => {
      const ok = parseTime24(inp.value);
      inp.value = ok || "";
    });
  });
}

export function confirmModal({ title, message, confirmLabel = "Confirm", cancelLabel = "Cancel", danger = false }) {
  return new Promise((resolve) => {
    const overlay = el(`<div class="modal modal-open modal-middle" role="dialog" aria-modal="true" aria-labelledby="confirm-modal-title">
      <div class="modal-box">
        <h3 class="text-lg font-bold" id="confirm-modal-title">${esc(title)}</h3>
        <div class="py-3 text-sm">${message}</div>
        <div class="modal-action">
          <button type="button" class="btn" data-confirm-cancel>${esc(cancelLabel)}</button>
          <button type="button" class="btn ${danger ? "danger" : "primary"}" data-confirm-ok>${esc(confirmLabel)}</button>
        </div>
      </div>
      <form method="dialog" class="modal-backdrop"><button type="button" data-confirm-cancel aria-label="Close">close</button></form>
    </div>`);
    const done = (ok) => {
      overlay.remove();
      resolve(ok);
    };
    overlay.querySelectorAll("[data-confirm-cancel]").forEach((b) => {
      b.addEventListener("click", () => done(false));
    });
    overlay.querySelector("[data-confirm-ok]")?.addEventListener("click", () => done(true));
    overlay.addEventListener("keydown", (e) => {
      if (e.key === "Escape") done(false);
    });
    document.body.appendChild(overlay);
    overlay.querySelector("[data-confirm-ok]")?.focus();
  });
}
