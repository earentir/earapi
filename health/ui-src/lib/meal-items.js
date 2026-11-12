import { state, emptyItem } from "./state.js";

export function itemHasParts(it) {
  return !!(it?.is_set || (it?.parts && it.parts.length > 0));
}

export function itemIsLoose(it) {
  return (it?.name || "").trim() && !itemHasParts(it);
}

export function setsInSlot(slot, excludeIdx) {
  return (state.mealDraft[slot]?.items || [])
    .map((it, j) => ({ it, j }))
    .filter(({ it, j }) => j !== excludeIdx && itemHasParts(it));
}

export function loosePeersInSlot(slot, excludeIdx) {
  return (state.mealDraft[slot]?.items || [])
    .map((it, j) => ({ it, j }))
    .filter(({ it, j }) => j !== excludeIdx && itemIsLoose(it));
}

export function partRowIsBlank(p) {
  return !(p?.name || "").trim() && !(Number(p?.grams) > 0);
}

export function pruneBlankSetParts(it) {
  if (!it?.parts?.length) return;
  it.parts = it.parts.filter((p) => !partRowIsBlank(p));
}

export function setPartStats(set) {
  const parts = (set?.parts || []).filter((p) => !partRowIsBlank(p));
  const grams = parts.reduce((s, p) => s + (Number(p.grams) || 0), 0);
  return { count: parts.length, grams };
}

export function recomputeSetGrams(set, keepPartial) {
  if (!set?.parts?.length) return;
  const { grams: g } = setPartStats(set);
  if (g <= 0) return;
  const eaten = Number(set.grams) || 0;
  if (!keepPartial || eaten <= 0 || eaten >= g - 0.01) {
    set.grams = g;
  }
}

export function cloneDraftItem(it) {
  const parts = Array.isArray(it.parts) ? it.parts.map(cloneDraftItem) : undefined;
  const isSet = !!it.is_set || (parts && parts.length > 0 && !it.stock_lot_id);
  return {
    name: it.name || "",
    grams: it.grams != null && it.grams !== "" ? it.grams : "",
    food_id: it.food_id || "",
    stock_lot_id: it.stock_lot_id || "",
    kcal: it.kcal != null ? it.kcal : "",
    sodium_mg: it.sodium_mg != null ? it.sodium_mg : "",
    kcal_per_100g: "",
    sodium_mg_per_100g: "",
    note: it.note || "",
    is_set: isSet,
    parts,
  };
}

export function absorbIntoSet(slot, setIdx, fromIdx) {
  const items = state.mealDraft[slot]?.items;
  if (!items || setIdx === fromIdx) return;
  const set = items[setIdx];
  const src = items[fromIdx];
  if (!itemHasParts(set) || !itemIsLoose(src)) return;
  const part = cloneDraftItem(src);
  part.parts = undefined;
  const onlyBlank = set.parts.length === 1 && !(set.parts[0].name || "").trim();
  if (onlyBlank) set.parts = [part];
  else set.parts.push(part);
  recomputeSetGrams(set);
  items.splice(fromIdx, 1);
}

export function ejectPartFromSet(slot, setIdx, partIdx) {
  const items = state.mealDraft[slot]?.items;
  const set = items?.[setIdx];
  if (!set?.parts?.[partIdx]) return;
  if (slot === "stock") {
    set.parts.splice(partIdx, 1);
    if (!set.parts.length) set.parts.push(emptyItem());
    recomputeSetGrams(set);
    return;
  }
  const loose = cloneDraftItem(set.parts[partIdx]);
  loose.parts = undefined;
  set.parts.splice(partIdx, 1);
  recomputeSetGrams(set);
  items.splice(setIdx + 1, 0, loose);
}
