/** Mirrors health/stock_sync.go matchMealItemToStockLot */

export function matchStockLotByName(name, lots) {
  const n = String(name ?? "").trim().toLowerCase();
  if (!n || !lots?.length) return null;
  if (n.includes("soup-tomato") || n.includes("soup tomato")) {
    for (const lot of lots) {
      if (String(lot.name ?? "").toLowerCase().includes("soup-tomato")) return lot;
    }
  }
  if (n.includes("non-tomato")) {
    for (const lot of lots) {
      if (String(lot.name ?? "").toLowerCase().includes("non-tomato")) return lot;
    }
  }
  for (const lot of lots) {
    const ln = String(lot.name ?? "").trim().toLowerCase();
    if (!ln) continue;
    if (n === ln) return lot;
    if (ln.length >= 10 && n.includes(ln)) return lot;
    if (n.length >= 10 && ln.includes(n)) return lot;
  }
  return null;
}
