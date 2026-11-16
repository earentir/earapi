/** Mirrors health/fuzzy.go for consistent catalog matching in the UI. */

export function cleanFoodDisplayName(name) {
  return String(name ?? "").trim().replace(/^\*+|\*+$/g, "").trim();
}

export function foodMatchQuery(name) {
  let n = String(name ?? "").trim();
  const i = n.indexOf("(");
  if (i > 0) n = n.slice(0, i).trim();
  return n.toLowerCase();
}

const FOOD_PREP_PHRASES = [
  "sunny side up", "sunny side", "sunny-side",
  "air fryer", "air fried", "air-fried", "airfried",
  "deep fried", "deep-fried", "pan fried", "pan-fried", "stir fried", "stir-fried",
  "no salt", "no-salt", "low salt", "low-salt", "low sodium", "low-sodium",
  "bone in", "bone-in", "skin on", "skin-on",
  "cooked no salt", "cooked, no salt",
  "ready meal", "ready-meal",
];

const foodPrepStopwords = new Set([
  "boiled", "roasted", "fried", "grilled", "baked", "steamed", "smoked", "braised",
  "poached", "scrambled", "sauteed", "sautéed", "saute", "microwaved", "charred", "broiled",
  "stewed", "blanched", "seared", "toasted", "air", "deep", "pan", "stir",
  "raw", "cooked", "uncooked", "frozen", "dried", "canned", "fresh", "unsalted", "salted",
  "seasoned", "pickled", "marinated", "breaded", "caramelized",
  "shredded", "sliced", "diced", "chopped", "minced", "crispy", "whole", "boneless",
  "bone", "bones", "skinless", "skin", "edible", "homemade", "ready",
  "sunny", "side", "up", "omelet", "omelette", "style", "minimal", "leftover",
  "large", "small", "medium",
]);

function stripPrepFromMealLine(s) {
  let t = String(s ?? "").trim().toLowerCase().replace(/-/g, " ");
  for (const p of FOOD_PREP_PHRASES) {
    t = t.replaceAll(p.replace(/-/g, " "), " ");
  }
  return t.replace(/\s+/g, " ").trim();
}

const foodPluralSingular = {
  potatoes: "potato", eggs: "egg", tomatoes: "tomato", grapes: "grape",
  leaves: "leaf", beans: "bean", onions: "onion", peppers: "pepper",
};

function tokens(s) {
  return s
    .toLowerCase()
    .replace(/[^\p{L}\p{N}]+/gu, " ")
    .split(/\s+/)
    .filter(Boolean);
}

function foldFoodToken(t) {
  if (foodPluralSingular[t]) return foodPluralSingular[t];
  if (t.endsWith("ies") && t.length > 4) return `${t.slice(0, -3)}y`;
  if (t.endsWith("es") && t.length > 3) return t.slice(0, -2);
  if (t.endsWith("s") && !t.endsWith("ss") && t.length > 3) return t.slice(0, -1);
  return t;
}

function matchQueryTokens(q) {
  const qt = tokens(stripPrepFromMealLine(q));
  const meaningful = qt.filter((t) => !/^\d/.test(t));
  if (!meaningful.length) return [];
  return meaningful.filter((t) => !foodPrepStopwords.has(t));
}

function tokenInFoodName(token, full) {
  if (!token) return true;
  if (full.includes(token)) return true;
  if (foodPluralSingular[token] && full.includes(foodPluralSingular[token])) return true;
  const folded = foldFoodToken(token);
  if (folded !== token && full.includes(folded)) return true;
  return false;
}

function tokensEqualForFood(a, b) {
  if (a === b) return true;
  return foldFoodToken(a) === foldFoodToken(b);
}

function levenshtein(a, b) {
  if (!a.length) return b.length;
  if (!b.length) return a.length;
  const prev = new Array(b.length + 1);
  const cur = new Array(b.length + 1);
  for (let j = 0; j <= b.length; j++) prev[j] = j;
  for (let i = 1; i <= a.length; i++) {
    cur[0] = i;
    for (let j = 1; j <= b.length; j++) {
      const cost = a[i - 1] === b[j - 1] ? 0 : 1;
      cur[j] = Math.min(cur[j - 1] + 1, prev[j] + 1, prev[j - 1] + cost);
    }
    for (let j = 0; j <= b.length; j++) prev[j] = cur[j];
  }
  return prev[b.length];
}

export function scoreFoodName(catalogName, q) {
  const low = cleanFoodDisplayName(catalogName).toLowerCase();
  const full = cleanFoodDisplayName(catalogName).toLowerCase();
  if (low === q || full === q) return 1000;
  if (low.includes(q) || full.includes(q)) {
    return 800 + Math.floor((200 * q.length) / Math.max(1, Math.min(low.length, full.length)));
  }
  const qt = matchQueryTokens(q);
  const nt = tokens(low);
  let overlap = 0;
  for (const tq of qt) {
    for (const tn of nt) {
      if (tokensEqualForFood(tq, tn)) overlap += 100;
      else if (tn.includes(tq) || tq.includes(tn)) overlap += 50;
      else {
        const d = levenshtein(tq, tn);
        if (d <= 2 && tq.length > 2) overlap += 40 - d * 10;
      }
    }
  }
  // Meal shorthand: every query token appears somewhere in the full catalog name (incl. parentheses).
  if (qt.length > 0 && qt.every((t) => tokenInFoodName(t, full))) {
    overlap = Math.max(overlap, 80 + qt.length * 40);
  }
  return overlap;
}

function tokenBoost(catalogName, q) {
  const low = cleanFoodDisplayName(catalogName).toLowerCase();
  return (
    q.includes("marata") && low.includes("marata")
    && q.includes("1.7") && low.includes("1.7")
  );
}

function mealNounTokens(mealLine) {
  return matchQueryTokens(mealLine);
}

function mealPrepHints(mealLine) {
  return tokens(String(mealLine ?? "").trim()).filter((t) => foodPrepStopwords.has(t));
}

export function scoreFoodNameForMealLine(catalogName, mealLine) {
  const q = foodMatchQuery(mealLine);
  let sc = scoreFoodName(catalogName, q);
  const full = cleanFoodDisplayName(catalogName).toLowerCase();
  for (const h of mealPrepHints(mealLine)) {
    if (full.includes(h)) sc += 55;
  }
  return sc;
}

/** Meal shorthand must match food nouns, not only a shared prep word (boiled). */
export function catalogMatchesMealLine(catalogName, mealLine) {
  const q = String(mealLine ?? "").trim().toLowerCase();
  if (!q) return false;
  const full = cleanFoodDisplayName(catalogName).toLowerCase();
  const qt = matchQueryTokens(q);
  if (!qt.length) return false;
  if (!qt.every((t) => tokenInFoodName(t, full))) return false;
  const nouns = mealNounTokens(mealLine);
  if (!nouns.length) return true;
  return nouns.every((n) => tokenInFoodName(n, full));
}

function filterFoodsForMealLine(foods, mealLine) {
  return foods.filter((f) => catalogMatchesMealLine(f.name, mealLine));
}

function allQueryTokensInName(catalogName, q) {
  return catalogMatchesMealLine(catalogName, q);
}

export function foodByAlias(aliases, foods, name) {
  const norm = String(name ?? "").trim().toLowerCase();
  if (!norm) return null;
  const row = (aliases || []).find((a) => String(a.alias ?? "").trim().toLowerCase() === norm);
  if (!row) return null;
  return foods.find((f) => f.id === row.food_id) || null;
}

/** Fuzzy alias match (e.g. meal line "hot peppers" → alias "hot horn pepper"). */
export function bestFoodByAlias(aliases, foods, mealLine) {
  const q = foodMatchQuery(mealLine);
  if (!q || !aliases?.length || !foods?.length) return null;
  const exact = foodByAlias(aliases, foods, mealLine);
  if (exact) return exact;
  const scored = [];
  for (const row of aliases) {
    const alias = cleanFoodDisplayName(row.alias);
    if (!alias) continue;
    if (!catalogMatchesMealLine(alias, mealLine)) continue;
    const sc = scoreFoodNameForMealLine(alias, mealLine);
    if (sc < 80) continue;
    const f = foods.find((x) => x.id === row.food_id);
    if (f) scored.push({ f, sc });
  }
  if (!scored.length) return null;
  scored.sort((a, b) => b.sc - a.sc);
  const bestSc = scored[0].sc;
  const tied = scored.filter((x) => x.sc >= bestSc - 15);
  const foodsSeen = new Set();
  for (const x of tied) foodsSeen.add(x.f.id);
  if (foodsSeen.size > 1) return null;
  return scored[0].f;
}

/** Typeahead via aliases → catalog food (with which alias matched). */
export function aliasFoodSuggestions(aliases, foods, itemName, limit = 8) {
  const q = String(itemName ?? "").trim();
  if (!q || !aliases?.length || !foods?.length) return [];
  const parts = q.toLowerCase().split(/\s+/).filter(Boolean);
  const partial = /\s$/.test(q) ? "" : parts[parts.length - 1] || "";
  const scored = [];
  for (const row of aliases) {
    const alias = cleanFoodDisplayName(row.alias);
    if (!alias) continue;
    const low = alias.toLowerCase();
    if (partial) {
      const wordHit = tokens(low).some((w) => w.startsWith(partial));
      if (!low.includes(partial) && !wordHit) continue;
    }
    const sc = scoreFoodName(alias, foodMatchQuery(itemName));
    if (partial && sc < 30 && !low.includes(partial)) continue;
    const f = foods.find((x) => x.id === row.food_id);
    if (f) scored.push({ f, alias, sc });
  }
  scored.sort((a, b) => b.sc - a.sc || a.alias.localeCompare(b.alias));
  const out = [];
  const seen = new Set();
  for (const row of scored) {
    if (seen.has(row.f.id)) continue;
    seen.add(row.f.id);
    out.push(row);
    if (out.length >= limit) break;
  }
  return out;
}

/** Live typeahead while typing a meal line (prefix-friendly, not as strict as auto-link). */
export function typeaheadFoodSuggestions(foods, itemName, limit = 8) {
  const raw = String(itemName ?? "").trim().toLowerCase();
  if (!raw || !foods?.length) return [];
  const pool = filterFoodsForMealLine(foods, itemName);
  const parts = raw.split(/\s+/).filter(Boolean);
  const partial = /\s$/.test(String(itemName ?? "")) ? "" : parts[parts.length - 1];

  const ranked = [];
  for (const f of pool) {
    const full = cleanFoodDisplayName(f.name).toLowerCase();
    if (partial) {
      const wordHit = tokens(full).some((w) => w.startsWith(partial));
      if (!full.includes(partial) && !wordHit) continue;
    }
    ranked.push({ f, sc: scoreFoodName(f.name, raw) });
  }
  ranked.sort((a, b) => b.sc - a.sc || a.f.name.localeCompare(b.f.name));
  let out = ranked.slice(0, limit).map((x) => x.f);
  if (out.length) return out;
  return foodPickCandidates(foods, itemName, limit);
}

/** Typeahead list for meal name fields (catalog lookup while typing). */
export function catalogFoodSuggestions(foods, itemName, limit = 8, aliases = null) {
  return mealLineFoodSuggestions(foods, aliases, itemName, limit).map((row) => row.f);
}

/**
 * Meal-line typeahead: when an alias matches what you type, the suggestion label is that alias
 * (not the catalog name + “aka …”).
 */
export function mealLineFoodSuggestions(foods, aliases, itemName, limit = 8) {
  const q = String(itemName ?? "").trim();
  if (!q || !foods?.length) return [];
  const byId = new Map();

  if (aliases?.length) {
    for (const row of aliasFoodSuggestions(aliases, foods, itemName, Math.max(limit, 8) * 2)) {
      const prev = byId.get(row.f.id);
      if (!prev || row.sc > prev.sc) {
        byId.set(row.f.id, {
          f: row.f,
          label: cleanFoodDisplayName(row.alias),
          via: "alias",
          sc: row.sc,
        });
      }
    }
  }

  for (const f of typeaheadFoodSuggestions(foods, q, limit)) {
    if (byId.has(f.id)) continue;
    const sc = scoreFoodName(f.name, foodMatchQuery(itemName));
    byId.set(f.id, {
      f,
      label: cleanFoodDisplayName(f.name),
      via: "catalog",
      sc,
    });
  }

  const out = [...byId.values()];
  out.sort((a, b) => {
    if (a.via === "alias" && b.via !== "alias") return -1;
    if (b.via === "alias" && a.via !== "alias") return 1;
    return b.sc - a.sc || a.label.localeCompare(b.label);
  });
  return out.slice(0, limit);
}

/** Foods that might match this line name (for user pick when several are close). */
/** Candidates for manual catalog pick (strict first, then any token overlap). */
export function foodPickCandidates(foods, itemName, limit = 12) {
  const strict = foodMatchCandidates(foods, itemName, limit);
  if (strict.length) return strict;
  const q = foodMatchQuery(itemName);
  const qt = matchQueryTokens(q);
  if (!qt.length || !foods?.length) return [];
  const ranked = [];
  for (const f of filterFoodsForMealLine(foods, itemName)) {
    const full = cleanFoodDisplayName(f.name).toLowerCase();
    let hit = 0;
    for (const t of qt) {
      if (tokenInFoodName(t, full)) hit += 1;
    }
    if (hit > 0) ranked.push({ f, hit, sc: scoreFoodName(f.name, q) });
  }
  ranked.sort((a, b) => b.hit - a.hit || b.sc - a.sc || a.f.name.localeCompare(b.f.name));
  let out = ranked.slice(0, limit).map((x) => x.f);
  if (out.length) return out;
  const qq = q.toLowerCase();
  return foods
    .filter((f) => cleanFoodDisplayName(f.name).toLowerCase().includes(qq))
    .slice(0, limit);
}

export function foodMatchCandidates(foods, itemName, limit = 12) {
  const q = foodMatchQuery(itemName);
  if (!q || !foods?.length) return [];
  const pool = filterFoodsForMealLine(foods, itemName);
  if (!pool.length) return [];
  const scored = [];
  for (const f of pool) {
    const sc = scoreFoodNameForMealLine(f.name, itemName);
    if (sc >= 80) scored.push({ f, sc });
  }
  if (!scored.length) return [];
  scored.sort((a, b) => b.sc - a.sc);
  const best = scored[0].sc;
  const minSc = Math.max(80, best - 15);
  return scored.filter((x) => x.sc >= minSc).slice(0, limit).map((x) => x.f);
}

export function bestFoodMatch(foods, itemName) {
  const q = foodMatchQuery(itemName);
  if (!q || !foods?.length) return null;
  const pool = filterFoodsForMealLine(foods, itemName);
  if (!pool.length) return null;
  const scored = [];
  for (const f of pool) {
    const sc = scoreFoodNameForMealLine(f.name, itemName);
    if (sc >= 80) scored.push({ f, sc });
  }
  scored.sort((a, b) => b.sc - a.sc);
  if (!scored.length) {
    const hit = foods.find((f) => tokenBoost(f.name, q));
    return hit || null;
  }
  const bestSc = scored[0].sc;
  if (scored.length > 1 && scored[1].sc >= bestSc - 15) return null;
  return scored[0].f;
}
