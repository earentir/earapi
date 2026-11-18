import { esc } from "./strings.js";

const FM_INP = "input input-bordered input-sm w-full max-w-full min-w-0";
const FM_TEXTAREA = "textarea textarea-bordered textarea-sm w-full min-h-12";
const FM_LABEL = "label-text text-xs uppercase opacity-60";

export { esc };

export function daisyToggle({ checked = false, extraAttrs = "" } = {}) {
  return `<input type="checkbox" class="toggle"${checked ? " checked" : ""} ${extraAttrs}>`;
}

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

const ISO_CAL_ICON = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" class="w-4 h-4 pointer-events-none" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="5" width="18" height="16" rx="2"/><path d="M16 3v4M8 3v4M3 10h18"/></svg>`;

/** Visible value is always YYYY-MM-DD. Calendar button calls the native picker via showPicker(). */
export function isoDateInputHtml({
  id,
  value = "",
  extraClass = "input input-bordered input-sm iso-date-text",
  extraAttrs = "",
  join = false,
} = {}) {
  const v = esc(String(value || ""));
  const visClass = join
    ? "input input-bordered input-sm font-mono tabular-nums"
    : `${extraClass} font-mono tabular-nums`;
  const idAttr = id ? ` id="${esc(id)}"` : "";
  const vis = `<span class="iso-date-field join-item"><input type="text"${idAttr} class="${visClass}" value="${v}" placeholder="YYYY-MM-DD" maxlength="10" size="10" inputmode="numeric" spellcheck="false" autocomplete="off" lang="en" data-iso-date ${extraAttrs}></span>`;
  const pick = `<button type="button" class="btn btn-square btn-sm btn-outline ${join ? "join-item" : ""} shrink-0" data-iso-date-pick title="Pick date" aria-label="Pick date">${ISO_CAL_ICON}</button>
      <input type="date" class="iso-date-picker-host" tabindex="-1" value="${v}" data-iso-date-picker aria-hidden="true">`;
  if (join) return `${vis}${pick}`;
  return `<div class="join w-auto relative">${vis}${pick}</div>`;
}

export function dateField(id, value, opts = {}) {
  const label = opts.label !== false;
  const labelHtml = label ? `<span class="label-text text-xs uppercase opacity-60">Date</span>` : "";
  return `<label class="form-control w-auto shrink-0">${labelHtml}
    <div class="join w-auto relative">
      <button type="button" class="btn btn-square btn-sm btn-outline join-item" data-shift="${esc(id)}" data-delta="-1" title="Previous day">‹</button>
      ${isoDateInputHtml({ id, value, join: true })}
      <button type="button" class="btn btn-square btn-sm btn-outline join-item" data-shift="${esc(id)}" data-delta="1" title="Next day">›</button>
    </div>
  </label>`;
}

function isoDateWrap(inp) {
  return inp.closest(".join, label") || inp.parentElement;
}

/** Wire ISO date text (always YYYY-MM-DD) plus hidden native picker, and 24h time. */
export function bindIsoInputs(root = document) {
  root.querySelectorAll("input[data-iso-date]").forEach((inp) => {
    if (inp.dataset.isoBound) return;
    inp.dataset.isoBound = "1";
    const wrap = isoDateWrap(inp);
    const picker = wrap?.querySelector("input[data-iso-date-picker]") || null;
    const pickBtn = wrap?.querySelector("[data-iso-date-pick]") || null;
    const syncPicker = () => {
      const ok = parseISODate(inp.value);
      if (ok && picker) picker.value = ok;
    };
    const openPicker = (e) => {
      e?.preventDefault();
      e?.stopPropagation();
      syncPicker();
      if (!picker) return;
      try {
        if (typeof picker.showPicker === "function") picker.showPicker();
        else picker.click();
      } catch {
        picker.focus();
        picker.click();
      }
    };
    inp.addEventListener("input", () => {
      const next = formatDateTyping(inp.value);
      if (next !== inp.value) {
        const pos = inp.selectionStart;
        inp.value = next;
        if (pos != null) inp.setSelectionRange(next.length, next.length);
      }
      const ok = parseISODate(inp.value);
      if (ok && picker) picker.value = ok;
    });
    inp.addEventListener("blur", () => {
      const ok = parseISODate(inp.value);
      if (ok) inp.value = ok;
      else if (inp.value.trim()) inp.value = formatDateTyping(inp.value);
      syncPicker();
    });
    pickBtn?.addEventListener("click", openPicker);
    picker?.addEventListener("change", () => {
      if (!picker.value) return;
      inp.value = picker.value;
      inp.dispatchEvent(new Event("input", { bubbles: true }));
      inp.dispatchEvent(new Event("change", { bubbles: true }));
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
      const cls = f.date
        ? "flex flex-col w-auto shrink-0"
        : f.grow
          ? "flex flex-col flex-[2] min-w-0"
          : f.tight
            ? "flex flex-col min-w-0 w-[6.5rem]"
            : "flex flex-col min-w-0 flex-1";
      const hint = f.tooltip
        ? `<span class="tooltip tooltip-bottom before:max-w-56" data-tip="${esc(f.tooltip)}"><button type="button" class="btn btn-ghost btn-xs btn-circle min-h-0 h-5 w-5 p-0 text-base-content/50" tabindex="-1" title="${esc(f.tooltip)}" aria-label="${esc(f.tooltip)}">?</button></span>`
        : "";
      const cap = `<span class="${FM_LABEL} inline-flex items-center gap-1">${esc(f.label)}${hint}</span>`;
      if (f.type === "checkbox") {
        const checked = val === true || val === "true" || val === "1";
        return `<label class="flex items-center gap-3 cursor-pointer py-1 select-none">${daisyToggle({ checked, extraAttrs: `data-fm="${esc(f.key)}"` })}<span class="text-sm">${esc(f.label)}</span></label>`;
      }
      if (f.type === "textarea") {
        return `<label class="${cls}">${cap}<textarea class="${FM_TEXTAREA}" data-fm="${esc(f.key)}" rows="${f.rows || 2}">${esc(String(val))}</textarea></label>`;
      }
      if (f.date) {
        return `<label class="${cls}">${cap}${isoDateInputHtml({ value: String(val), extraClass: FM_INP, extraAttrs: `data-fm="${esc(f.key)}"` })}</label>`;
      }
      const mode = f.type === "decimal" ? ' inputmode="decimal"' : f.type === "numeric" ? ' inputmode="numeric"' : "";
      return `<label class="${cls}">${cap}<input type="text" class="${FM_INP}" data-fm="${esc(f.key)}" value="${esc(String(val))}"${mode}></label>`;
    };
    const fieldHtml = fields.map((f) => {
      if (f.type === "row" && f.fields?.length) {
        return `<div class="flex flex-wrap gap-2 items-end w-full min-w-0">${f.fields.map(fieldLabel).join("")}</div>`;
      }
      return fieldLabel(f);
    }).join("");
    const deleteBtn = deleteLabel && onDelete
      ? `<button type="button" class="btn btn-outline btn-error btn-sm" data-fm-delete>${esc(deleteLabel)}</button>`
      : "";
    const overlay = el(`<div class="modal modal-open modal-middle" role="dialog" aria-modal="true">
      <div class="modal-box max-w-lg overflow-x-hidden">
        <h3 class="text-lg font-bold">${esc(title)}</h3>
        <div class="flex flex-col gap-3 mt-3">${fieldHtml}</div>
        <div class="modal-action mt-2">
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
    if (first?.select && first.type !== "date" && first.type !== "checkbox") first.select();
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
