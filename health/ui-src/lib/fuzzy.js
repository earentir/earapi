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

function tokens(s) {
  return s
    .toLowerCase()
    .replace(/[^\p{L}\p{N}]+/gu, " ")
    .split(/\s+/)
    .filter(Boolean);
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
  const qt = tokens(q);
  const nt = tokens(low);
  let overlap = 0;
  for (const tq of qt) {
    for (const tn of nt) {
      if (tq === tn) overlap += 100;
      else if (tn.includes(tq) || tq.includes(tn)) overlap += 50;
      else {
        const d = levenshtein(tq, tn);
        if (d <= 2 && tq.length > 2) overlap += 40 - d * 10;
      }
    }
  }
  // Meal shorthand: every query token appears somewhere in the full catalog name (incl. parentheses).
  if (qt.length > 0 && qt.every((t) => full.includes(t))) {
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

function allQueryTokensInName(catalogName, q) {
  const full = cleanFoodDisplayName(catalogName).toLowerCase();
  const qt = tokens(q);
  if (!qt.length) return false;
  return qt.every((t) => full.includes(t));
}

export function foodByAlias(aliases, foods, name) {
  const norm = String(name ?? "").trim().toLowerCase();
  if (!norm) return null;
  const row = (aliases || []).find((a) => String(a.alias ?? "").trim().toLowerCase() === norm);
  if (!row) return null;
  return foods.find((f) => f.id === row.food_id) || null;
}

/** Typeahead: strict for multi-word queries (avoids soup≈sauce / lone "base" noise). */
export function catalogFoodSuggestions(foods, itemName, limit = 8) {
  const q = String(itemName ?? "").trim().toLowerCase();
  if (!q || !foods?.length) return [];
  const qt = tokens(q);
  if (qt.length >= 2) return foodMatchCandidates(foods, itemName, limit);
  return foods.filter((f) => cleanFoodDisplayName(f.name).toLowerCase().includes(q)).slice(0, limit);
}

/** Foods that might match this line name (for user pick when several are close). */
/** Candidates for manual catalog pick (strict first, then any token overlap). */
export function foodPickCandidates(foods, itemName, limit = 12) {
  const strict = foodMatchCandidates(foods, itemName, limit);
  if (strict.length) return strict;
  const q = foodMatchQuery(itemName);
  const qt = tokens(q);
  if (!qt.length || !foods?.length) return [];
  const ranked = [];
  for (const f of foods) {
    const full = cleanFoodDisplayName(f.name).toLowerCase();
    let hit = 0;
    for (const t of qt) {
      if (full.includes(t)) hit += 1;
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
  const qt = tokens(q);
  let pool = foods;
  if (qt.length >= 2) {
    const strict = foods.filter((f) => allQueryTokensInName(f.name, q));
    if (!strict.length) return [];
    pool = strict;
  }
  const scored = [];
  for (const f of pool) {
    const sc = scoreFoodName(f.name, q);
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
  let pool = foods;
  const qt = tokens(q);
  if (qt.length > 1) {
    const strict = foods.filter((f) => allQueryTokensInName(f.name, q));
    if (!strict.length) return null;
    pool = strict;
  }
  const scored = [];
  for (const f of pool) {
    const sc = scoreFoodName(f.name, q);
    if (sc > 0) scored.push({ f, sc });
  }
  scored.sort((a, b) => b.sc - a.sc);
  if (!scored.length) return null;
  const bestSc = scored[0].sc;
  if (bestSc < 80) {
    const hit = foods.find((f) => tokenBoost(f.name, q));
    return hit || null;
  }
  if (scored.length > 1 && scored[1].sc >= bestSc - 15) return null;
  return scored[0].f;
}
