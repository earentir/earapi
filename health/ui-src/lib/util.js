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

/**
 * Simple form in a DaisyUI modal. Resolves field values or null if cancelled.
 * fields: { key, label, value?, type?: text|decimal|numeric|textarea|checkbox|row, grow?, tight?, date?: bool, rows?, fields? (for row) }
 * deleteLabel + onDelete: optional; Delete button on the left of the action row (edit flows).
 */
export function formModal({
  title,
  fields,
  submitLabel = "Save",
  cancelLabel = "Cancel",
  deleteLabel,
  onDelete,
}) {
  return new Promise((resolve) => {
    const fieldLabel = (f) => {
      const val = f.value ?? "";
      const cls = f.grow ? "field grow" : f.tight ? "field tight" : "field";
      if (f.type === "checkbox") {
        const checked = val === true || val === "true" || val === "1";
        return `<label class="check"><input type="checkbox" data-fm="${esc(f.key)}" ${checked ? "checked" : ""}> ${esc(f.label)}</label>`;
      }
      if (f.type === "textarea") {
        return `<label class="${cls}"><span>${esc(f.label)}</span><textarea data-fm="${esc(f.key)}" rows="${f.rows || 2}">${esc(String(val))}</textarea></label>`;
      }
      const mode = f.type === "decimal" ? ' inputmode="decimal"' : f.type === "numeric" ? ' inputmode="numeric"' : "";
      const dateAttrs = f.date ? ' class="iso-date mono" maxlength="10" placeholder="YYYY-MM-DD" autocomplete="off" lang="en"' : "";
      return `<label class="${cls}"><span>${esc(f.label)}</span><input type="text" data-fm="${esc(f.key)}" value="${esc(String(val))}"${mode}${dateAttrs}></label>`;
    };
    const fieldHtml = fields.map((f) => {
      if (f.type === "row" && f.fields?.length) {
        return `<div class="form-modal-row row">${f.fields.map(fieldLabel).join("")}</div>`;
      }
      return fieldLabel(f);
    }).join("");
    const deleteBtn = deleteLabel && onDelete
      ? `<button type="button" class="btn danger" data-fm-delete>${esc(deleteLabel)}</button>`
      : "";
    const overlay = el(`<div class="modal modal-open modal-middle entity-form-modal" role="dialog" aria-modal="true">
      <div class="modal-box modal-box-form">
        <h3 class="text-lg font-bold form-modal-title">${esc(title)}</h3>
        <div class="form-modal-fields stack-fields">${fieldHtml}</div>
        <div class="modal-action form-modal-actions">
          ${deleteBtn}
          <button type="button" class="btn" data-fm-cancel>${esc(cancelLabel)}</button>
          <button type="button" class="btn primary" data-fm-submit>${esc(submitLabel)}</button>
        </div>
      </div>
      <form method="dialog" class="modal-backdrop"><button type="button" data-fm-cancel aria-label="Close">close</button></form>
    </div>`);
    const close = (vals) => {
      overlay.remove();
      resolve(vals);
    };
    const leafFields = () => fields.flatMap((f) => (f.type === "row" && f.fields?.length ? f.fields : [f]));
    const read = () => {
      const out = {};
      for (const f of leafFields()) {
        if (!f.key) continue;
        const inp = overlay.querySelector(`[data-fm="${f.key}"]`);
        if (!inp) continue;
        if (f.type === "checkbox") out[f.key] = inp.checked;
        else out[f.key] = inp.value;
      }
      return out;
    };
    overlay.querySelectorAll("[data-fm-cancel]").forEach((b) => b.addEventListener("click", () => close(null)));
    overlay.querySelector("[data-fm-submit]")?.addEventListener("click", () => close(read()));
    overlay.querySelector("[data-fm-delete]")?.addEventListener("click", async () => {
      if (!onDelete) return;
      try {
        const ok = await onDelete();
        if (ok) close(null);
      } catch {
        /* onDelete shows toast */
      }
    });
    overlay.addEventListener("keydown", (e) => {
      if (e.key === "Escape") close(null);
      if (e.key === "Enter" && e.target?.matches?.("input:not([type=checkbox])")) {
        e.preventDefault();
        close(read());
      }
    });
    document.body.appendChild(overlay);
    bindIsoInputs(overlay);
    const first = overlay.querySelector("[data-fm]");
    first?.focus();
    if (first?.select) first.select();
  });
}

/** Modal shell with custom HTML; onSubmit(overlay) may return false to keep open. */
export function customModal({
  title,
  bodyHtml,
  submitLabel = "Save",
  cancelLabel = "Cancel",
  extraClass = "",
  onMount,
  onSubmit,
}) {
  return new Promise((resolve) => {
    const overlay = el(`<div class="modal modal-open modal-middle ${extraClass}" role="dialog" aria-modal="true">
      <div class="modal-box modal-box-form modal-box-wide">
        <h3 class="text-lg font-bold">${esc(title)}</h3>
        <div class="custom-modal-body">${bodyHtml}</div>
        <div class="modal-action">
          <button type="button" class="btn" data-cm-cancel>${esc(cancelLabel)}</button>
          <button type="button" class="btn primary" data-cm-submit>${esc(submitLabel)}</button>
        </div>
      </div>
      <form method="dialog" class="modal-backdrop"><button type="button" data-cm-cancel aria-label="Close">close</button></form>
    </div>`);
    const close = (ok) => {
      overlay.remove();
      resolve(ok);
    };
    overlay.querySelectorAll("[data-cm-cancel]").forEach((b) => b.addEventListener("click", () => close(false)));
    overlay.querySelector("[data-cm-submit]")?.addEventListener("click", async () => {
      if (onSubmit) {
        try {
          const keep = await onSubmit(overlay);
          if (keep === false) return;
        } catch {
          return;
        }
      }
      close(true);
    });
    overlay.addEventListener("keydown", (e) => {
      if (e.key === "Escape") close(false);
    });
    document.body.appendChild(overlay);
    onMount?.(overlay);
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
