import { esc } from "./strings.js";

const FM_INP = "input input-bordered input-sm w-full";
const FM_TEXTAREA = "textarea textarea-bordered textarea-sm w-full min-h-16";
const FM_LABEL = "label-text text-xs uppercase opacity-60";

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
  const labelHtml = label ? `<span class="label-text text-xs uppercase opacity-60">Date</span>` : "";
  return `<label class="form-control min-w-0 max-w-xs">${labelHtml}
    <div class="join w-full">
      <button type="button" class="btn btn-outline btn-sm join-item" data-shift="${esc(id)}" data-delta="-1" title="Previous day">‹</button>
      <input type="text" class="input input-bordered input-sm join-item w-full min-w-0 font-mono tabular-nums" id="${esc(id)}" value="${esc(value)}" placeholder="YYYY-MM-DD" maxlength="10" inputmode="numeric" spellcheck="false" autocomplete="off" lang="en" data-iso-date>
      <button type="button" class="btn btn-outline btn-sm join-item" data-shift="${esc(id)}" data-delta="1" title="Next day">›</button>
    </div>
  </label>`;
}

/** Wire text inputs to stay ISO date / 24h time (no native date/time pickers). */
export function bindIsoInputs(root = document) {
  root.querySelectorAll("input[data-iso-date]").forEach((inp) => {
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
      const cls = f.grow ? "form-control flex-[2] min-w-0" : f.tight ? "form-control min-w-0 max-w-[7rem]" : "form-control min-w-0 flex-1";
      if (f.type === "checkbox") {
        const checked = val === true || val === "true" || val === "1";
        return `<label class="label cursor-pointer justify-start gap-2 py-1 min-h-0"><input type="checkbox" class="checkbox checkbox-sm" data-fm="${esc(f.key)}" ${checked ? "checked" : ""}><span class="label-text">${esc(f.label)}</span></label>`;
      }
      if (f.type === "textarea") {
        return `<label class="${cls}"><span class="${FM_LABEL}">${esc(f.label)}</span><textarea class="${FM_TEXTAREA}" data-fm="${esc(f.key)}" rows="${f.rows || 2}">${esc(String(val))}</textarea></label>`;
      }
      const mode = f.type === "decimal" ? ' inputmode="decimal"' : f.type === "numeric" ? ' inputmode="numeric"' : "";
      const dateExtra = f.date ? ' maxlength="10" placeholder="YYYY-MM-DD" autocomplete="off" lang="en" data-iso-date' : "";
      const inpCls = f.date ? FM_INP.replace("input-sm", "input-sm font-mono tabular-nums") : FM_INP;
      return `<label class="${cls}"><span class="${FM_LABEL}">${esc(f.label)}</span><input type="text" class="${inpCls}" data-fm="${esc(f.key)}" value="${esc(String(val))}"${mode}${dateExtra}></label>`;
    };
    const fieldHtml = fields.map((f) => {
      if (f.type === "row" && f.fields?.length) {
        return `<div class="flex flex-wrap gap-2 items-end">${f.fields.map(fieldLabel).join("")}</div>`;
      }
      return fieldLabel(f);
    }).join("");
    const deleteBtn = deleteLabel && onDelete
      ? `<button type="button" class="btn btn-outline btn-error btn-sm" data-fm-delete>${esc(deleteLabel)}</button>`
      : "";
    const overlay = el(`<div class="modal modal-open modal-middle" role="dialog" aria-modal="true">
      <div class="modal-box max-w-lg">
        <h3 class="text-lg font-bold">${esc(title)}</h3>
        <div class="flex flex-col gap-2 py-2">${fieldHtml}</div>
        <div class="modal-action">
          ${deleteBtn}
          <button type="button" class="btn btn-outline btn-sm" data-fm-cancel>${esc(cancelLabel)}</button>
          <button type="button" class="btn btn-primary btn-sm" data-fm-submit>${esc(submitLabel)}</button>
        </div>
      </div>
      <div class="modal-backdrop modal-backdrop-inert" aria-hidden="true"></div>
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

/** Modal shell with custom HTML; onSubmit(overlay) may return false to keep open.
 * deleteLabel + onDelete: optional; Delete button on the left of the action row (edit flows).
 */
export function customModal({
  title,
  titleAside,
  bodyHtml,
  submitLabel = "Save",
  cancelLabel = "Cancel",
  deleteLabel,
  extraClass = "",
  boxClass = "modal-box w-11/12 max-w-5xl",
  onMount,
  onSubmit,
  onDelete,
}) {
  return new Promise((resolve) => {
    const deleteBtn = deleteLabel && onDelete
      ? `<button type="button" class="btn btn-outline btn-error btn-sm" data-cm-delete>${esc(deleteLabel)}</button>`
      : "";
    const head = titleAside
      ? `<div class="flex flex-wrap items-baseline justify-between gap-2"><h3 class="text-lg font-bold">${esc(title)}</h3><div class="text-sm text-base-content/70">${titleAside}</div></div>`
      : `<h3 class="text-lg font-bold">${esc(title)}</h3>`;
    const modalCls = extraClass ? ` ${extraClass}` : "";
    const overlay = el(`<div class="modal modal-open modal-middle${modalCls}" role="dialog" aria-modal="true">
      <div class="${boxClass}">
        ${head}
        <div class="py-2" data-modal-body>${bodyHtml}</div>
        <div class="modal-action">
          ${deleteBtn}
          <button type="button" class="btn btn-outline btn-sm" data-cm-cancel>${esc(cancelLabel)}</button>
          <button type="button" class="btn btn-primary btn-sm" data-cm-submit>${esc(submitLabel)}</button>
        </div>
      </div>
      <div class="modal-backdrop modal-backdrop-inert" aria-hidden="true"></div>
    </div>`);
    const close = (ok) => {
      overlay.remove();
      resolve(ok);
    };
    overlay.querySelectorAll("[data-cm-cancel]").forEach((b) => b.addEventListener("click", () => close(false)));
    overlay.querySelector("[data-cm-delete]")?.addEventListener("click", async () => {
      if (!onDelete) return;
      try {
        const ok = await onDelete();
        if (ok) close(false);
      } catch {
        /* onDelete shows toast */
      }
    });
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
          <button type="button" class="btn btn-outline btn-sm" data-confirm-cancel>${esc(cancelLabel)}</button>
          <button type="button" class="btn btn-sm ${danger ? "btn-outline btn-error" : "btn-primary"}" data-confirm-ok>${esc(confirmLabel)}</button>
        </div>
      </div>
      <div class="modal-backdrop modal-backdrop-inert" aria-hidden="true"></div>
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
