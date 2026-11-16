/** EU label salt (g NaCl per 100 g) ↔ sodium (mg per 100 g), Reg. 1169/2011. */
export const SODIUM_MG_PER_GRAM_SALT = 393.34;

export function sodiumMgFromSaltG(saltG) {
  return saltG * SODIUM_MG_PER_GRAM_SALT;
}

export function saltGFromSodiumMg(sodiumMg) {
  if (!Number.isFinite(sodiumMg) || sodiumMg <= 0) return null;
  return sodiumMg / SODIUM_MG_PER_GRAM_SALT;
}