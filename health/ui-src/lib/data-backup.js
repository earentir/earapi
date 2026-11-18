import { api, toast } from "./api.js";
import { state } from "./state.js";

/** API scope for the active tab (import only applies this slice). */
export function backupScopeForTab(tab) {
  switch (tab) {
    case "foods":
      return "foods";
    case "days":
    case "today":
      return "days";
    case "stock":
      return "stock";
    case "weight":
      return "weight";
    default:
      return "all";
  }
}

function scopeLabel(scope) {
  switch (scope) {
    case "foods":
      return "Foods";
    case "days":
      return "Days";
    case "stock":
      return "Stock";
    case "weight":
      return "Weight";
    default:
      return "All data";
  }
}

export function dataBackupBarHtml(scope, { fullBackup = false, extra = "" } = {}) {
  const lab = fullBackup ? "All data" : scopeLabel(scope);
  const s = fullBackup ? "all" : scope;
  return `<div class="flex flex-wrap gap-2 items-center mb-3">
    <button type="button" class="btn btn-outline btn-sm" data-export-json="${s}">Export ${lab}</button>
    <label class="btn btn-outline btn-sm">Import ${lab}
      <input type="file" accept="application/json,.json" hidden data-import-json="${s}">
    </label>
    ${extra}
  </div>`;
}

export async function downloadHealthExport(scope) {
  const bundle = await api(`/export/json?scope=${encodeURIComponent(scope)}`);
  const name = scope === "all"
    ? `health-backup-${new Date().toISOString().slice(0, 10)}.json`
    : `health-${scope}-${new Date().toISOString().slice(0, 10)}.json`;
  const blob = new Blob([JSON.stringify(bundle, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  a.click();
  URL.revokeObjectURL(url);
  toast(`Exported ${scopeLabel(scope)}`);
}

export async function importHealthJson(scope, file, { onDone, replaceAll = false } = {}) {
  if (!file) return;
  const text = await file.text();
  let body;
  try {
    body = JSON.parse(text);
  } catch {
    toast("Invalid JSON file");
    return;
  }
  const merge = scope !== "all" || !replaceAll;
  const q = `scope=${encodeURIComponent(scope)}&merge=${merge ? "true" : "false"}`;
  try {
    const out = await api(`/import/json?${q}`, {
      method: "POST",
      body: JSON.stringify(body),
    });
    const parts = Object.entries(out.imported || {})
      .filter(([, n]) => n > 0)
      .map(([k, n]) => `${k}: ${n}`)
      .join(", ");
    toast(parts ? `Imported (${parts})` : "Imported");
    await onDone?.(scope);
  } catch (e) {
    toast(e.message || "Import failed");
  }
}

export function bindDataBackupHandlers({ onImported }) {
  document.querySelectorAll("[data-export-json]").forEach((btn) => {
    btn.addEventListener("click", () => {
      downloadHealthExport(btn.dataset.exportJson).catch((e) => toast(e.message));
    });
  });
  document.querySelectorAll("[data-import-json]").forEach((inp) => {
    inp.addEventListener("change", async () => {
      const file = inp.files?.[0];
      inp.value = "";
      if (!file) return;
      const scope = inp.dataset.importJson || backupScopeForTab(state.tab);
      const replaceAll = scope === "all" && document.getElementById("import-all-replace")?.checked;
      await importHealthJson(scope, file, { onDone: onImported, replaceAll });
    });
  });
}