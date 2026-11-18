# Health API — food catalog (LLM guide)

## Base URL

```
https://api.earentir.dev/health/v1
```

Web UI: `https://api.earentir.dev/health/ui/`

All routes below except **`GET /help`** require auth.

## Auth

Credentials come from server `config/earapi.json` → `health.users[]` (`username` + `token`).

Send on **every** request (JSON bodies use `Content-Type: application/json`):

| Header       | Value            |
| ------------ | ---------------- |
| `X-User`     | your username    |
| `X-API-Key`  | your token       |
| `Accept`     | `application/json` |

Alternatives (GET only): query `?user=&token=`  
Or: `Authorization: Bearer <token>` **with** `X-User`.

- **401** `{"error":"unauthorized"}` — wrong/missing credentials.
- **503** `{"error":"health auth not configured"}` — server not set up.

## Nutrition model (catalog vs meal lines)

| Concept | Meaning |
| ------- | ------- |
| **Per-100 g fields** (`kcal_per_100g`, macros, sodium, …) | Nutrition for **100 g edible** food (label math), not “as weighed with bone”. |
| **`label_grams`** | Pack/serving reference on the label (usually **100**). Not meal grams. |
| **Meal line `grams`** | What the user weighed for that eating event (may still include bone, seeds, peel). |
| **`edible_fraction`** | Share of **meal `grams`** that count when scaling per-100 g → line kcal/sodium. |

Effective grams for scaling:

```text
nutrition_grams = meal_grams × edible_fraction
line_kcal ≈ kcal_per_100g × nutrition_grams / 100
```

(Same pattern for sodium and other per-100 g fields. **Eggs** are a special case: catalog kcal may be per egg, not per 100 g — server uses count/`×` heuristics.)

**Do not** create a second catalog row whose only difference is “(edible)” in the name. Set yield on the same food via **`edible_fraction`** (per cut when using **`variants`**).

---

## `Food` JSON shape

Top-level fields mirror the **default variant** profile after save (see **`variants.default_cut`** + **`default_prep`**). Omitted fields are unchanged on patch-style updates (`PUT`, **`EnsureFood`** merge).

### Simple food (no variants)

```json
{
  "id": "yogurt-10pct",
  "name": "10% yogurt",
  "kcal_per_100g": 130,
  "sodium_mg_per_100g": 40,
  "protein_g_per_100g": 9.5,
  "label_grams": 100,
  "notes": "",
  "edible_fraction": null
}
```

Omit **`edible_fraction`** or use **`1`** when meal **`grams`** are already edible-only.

### Variant food (cut × prep)

```json
{
  "id": "chicken-breast",
  "name": "Chicken breast",
  "kcal_per_100g": 165,
  "sodium_mg_per_100g": 74,
  "protein_g_per_100g": 31,
  "label_grams": 100,
  "notes": "",
  "variants": {
    "enabled": true,
    "default_cut": "skinless",
    "default_prep": "cooked",
    "skinless": {
      "edible_fraction": 1,
      "raw": {
        "kcal_per_100g": 120,
        "protein_g_per_100g": 22,
        "sodium_mg_per_100g": 60
      },
      "cooked": {
        "kcal_per_100g": 165,
        "protein_g_per_100g": 31,
        "sodium_mg_per_100g": 74
      }
    },
    "with_skin": {
      "edible_fraction": 0.85,
      "cooked": { "kcal_per_100g": 197 }
    }
  }
}
```

### Field notes

| Field | Notes |
| ----- | ----- |
| **`variants`** | When **`enabled`: true**, full object is stored. **`skinless`** / **`with_skin`** each hold optional **`edible_fraction`**, **`raw`**, **`cooked`** blocks (`FoodNutritionPer100`: same keys as top-level per-100 g). |
| **`default_cut`** | `skinless` \| `with_skin` — which cut’s profile is copied to top-level on save and which cut’s **`edible_fraction`** applies to meals unless overridden. |
| **`default_prep`** | `raw` \| `cooked` — which prep block supplies top-level macros. |
| **`edible_fraction`** | **0–1** fraction. Values **> 1 and ≤ 100** are treated as **percent** (e.g. `57` → `0.57`). Invalid/out-of-range → **1**. |
| **Yield precedence (meals)** | `MealItem.edible_fraction` → if variants: `variants.{default_cut}.edible_fraction` → top-level **`edible_fraction`** → **1**. |
| **Write behavior** | On insert/update, server runs **`SyncFoodFromVariantDefault`**: top-level per-100 g + **`label_grams`** (100) + label kcal/sodium come from the default cut/prep block. Sending **`variants`** in a patch **replaces** the whole variants object (not a deep merge per nested key). |
| **Create rules** | **`EnsureFood`** / **`POST /foods`** for a **new** row need at least **`kcal_per_100g` and/or `sodium_mg_per_100g`** unless updating by **`id`** or exact **`name`**. |
| **Fuzzy merge** | **`EnsureFood`** tries **`BestFoodMatch`** on **`name`** before insert — prefer **`id`** when you mean one specific row. |

### `MealItem` yield override

```json
{
  "name": "Chicken leg",
  "food_id": "chicken-leg",
  "grams": 270,
  "edible_fraction": 0.57
}
```

Use when this line’s yield differs from the catalog default (one-off weigh-in). Omit to use the food’s default yield.

## Aliases

```json
{ "alias": "peppers", "food_id": "bell-pepper" }
```

Aliases are **lowercased** on store. Meal lines can resolve old names / deleted `food_id`s via alias redirects.

---

## Endpoints — catalog CRUD

### `GET /foods`

List catalog + all aliases.

| Query   | Meaning                                      |
| ------- | -------------------------------------------- |
| `q`     | Rank/filter by name (fuzzy)                    |
| `limit` | Max rows when using `q` (or cap when no `q`) |

**200**

```json
{
  "foods": [{ "...Food..." }],
  "aliases": [{ "alias": "...", "food_id": "..." }],
  "count": 42
}
```

---

### `GET /foods/:id`

**200** `{ "food": { ... }, "aliases": [ ... ] }`  
**404** `{ "error": "not found" }`

---

### `POST /foods`

Create one food (always insert path, not fuzzy merge).

**Body:** `Food` (no `id` required; server assigns slug id).

**201** `{ "food": { ... } }`

---

### `PUT /foods/:id`

Replace/patch fields on one row.

**Body:** `Food` (fields to update; `id` in URL wins).

**200** `{ "food": { ... } }`

---

### `DELETE /foods/:id`

Remove catalog row.

**Body (optional JSON)** or query:

```json
{ "replace_with": "<other-food-id>" }
```

| `replace_with` | Behavior                                                                      |
| -------------- | ----------------------------------------------------------------------------- |
| set            | Re-point aliases for deleted id/name to replacement; old ids stay resolvable |
| empty          | Row deleted; meal lines with that `food_id` may show as **missing_food** gaps |

**200** `{ "deleted": "<id>", "replace_with": "..." }`

**Do not** bulk-delete without explicit user approval. **`POST /foods/batch`** `delete[]` uses **no** `replace_with` — leaves broken refs.

---

### `POST /foods/batch`

Bulk upsert/delete.

**Body**

```json
{
  "upsert": [{ "name": "...", "kcal_per_100g": 100 }],
  "delete": ["food-id-to-remove"]
}
```

- Each **`upsert`** → **`EnsureFood`** (fuzzy name match, update by `id`, or create).
- Each **`delete`** → **`DeleteFood(id, "")`** — no replacement.

**200** `{ "foods": [ ... ], "deleted": [ "id", ... ] }`

---

## Endpoints — aliases

### `POST /foods/:id/aliases`

**Body** `{ "alias": "hot horn pepper" }`  
**200** `{ "alias": { "alias": "...", "food_id": "..." } }`

One alias per request; repeat for multiple.

---

### `DELETE /foods/:id/aliases`

**Body** `{ "alias": "peppers" }`  
**200** `{ "ok": true }`

---

## Endpoints — gaps & fill (catalog + meals)

Use these when fixing nutrition data for logged days, not only the static catalog.

### `GET /gaps`

Find catalog rows and meal lines that still need data.

| Query          | Default                          |
| -------------- | -------------------------------- |
| `from`, `to`   | date range (ISO `YYYY-MM-DD`)    |
| `limit`        | `100`                            |

**200** — no kcal numbers on food gaps (only `missing` field names):

```json
{
  "foods": [{ "id": "...", "name": "...", "missing": ["kcal_per_100g"] }],
  "items": [{
    "ref": "2026-11-11/dinner/2",
    "date": "...",
    "slot": "dinner",
    "name": "...",
    "grams": 150,
    "note": "missing_food"
  }],
  "food_count": 1,
  "item_count": 3
}
```

`note: "missing_food"` → `food_id` points at a deleted/unresolved catalog row.

---

### `POST /fill`

Upsert foods and recompute affected meals.

**Body**

```json
{
  "foods": [
    { "name": "New spice", "kcal_per_100g": 250, "sodium_mg_per_100g": 800 },
    { "id": "pasta", "kcal_per_100g": 155 }
  ],
  "recompute_slots": [{ "date": "2026-11-11", "slot": "dinner" }],
  "recompute_days": ["2026-11-11"]
}
```

**200**

```json
{
  "foods_ensured": [],
  "foods_created": [],
  "reply": "human-readable summary if new rows were created",
  "recomputed": [{ "date": "...", "slot": "..." }],
  "gaps_remaining": {}
}
```

**Typical LLM workflow:** `GET /gaps` → `POST /fill` with `foods` + `recompute_days` → check `gaps_remaining`.

---

## Endpoints — export / import (foods only)

### `GET /export/json?scope=foods`

Downloads JSON bundle (includes `foods` and `food_aliases`). Other scopes: `days`, `stock`, `weight`, `all`.

### `POST /import/json?scope=foods&merge=true`

Import from bundle; `merge=true` merges, `merge=false` replaces per server rules. **Dangerous on prod** — only when the user explicitly asks.

---

## Related (meals touching catalog)

| Method | Path                      | Use                                        |
| ------ | ------------------------- | ------------------------------------------ |
| `POST` | `/log`                    | Day log; optional `ensure_foods`, `delete_foods` |
| `POST` | `/days/:date/meals`       | Meal slot; optional `ensure_foods`         |

`ensure_foods` uses the same `Food` shape as **`EnsureFood`**.

Structured meal **`items[]`** may include **`edible_fraction`** per line (see **Nutrition model**). **`grams`** is always as-weighed; do not encode yield only in the food name.

---

## Safety rules for LLMs

1. **Read before write:** `GET /foods` or `GET /foods/:id` before edits.
2. **Prefer `id` + `PUT`** for known rows; use **`POST /fill`** or batch **`upsert`** for gap fixes.
3. **Never** bulk `delete` / `import?merge=false` without explicit user confirmation.
4. **Deleting duplicates:** add aliases on the keeper → `DELETE` duplicate with **`replace_with`** set to keeper `id`.
5. **Do not invent** destructive merge/delete scripts against production.

---

## Example: list and update

```bash
curl -sS 'https://api.earentir.dev/health/v1/foods?q=pepper&limit=10' \
  -H 'X-User: USER' -H 'X-API-Key: TOKEN'

curl -sS -X PUT 'https://api.earentir.dev/health/v1/foods/bell-pepper' \
  -H 'X-User: USER' -H 'X-API-Key: TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{"name":"Bell pepper","kcal_per_100g":31,"sodium_mg_per_100g":4}'
```

### Example: variants + edible yield (bone-in weigh)

After a weigh-in: total **440 g**, waste **265 g** → edible share **175/440 ≈ 0.398**. Store on the **default cut** (and send full **`variants`** on `PUT` so nothing is dropped):

```bash
curl -sS -X PUT 'https://api.earentir.dev/health/v1/foods/chicken-wings' \
  -H 'X-User: USER' -H 'X-API-Key: TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Chicken wings",
    "variants": {
      "enabled": true,
      "default_cut": "skinless",
      "default_prep": "cooked",
      "skinless": {
        "edible_fraction": 0.398,
        "cooked": { "kcal_per_100g": 240, "sodium_mg_per_100g": 82 }
      },
      "with_skin": {}
    }
  }'
```

Meal lines still log **`grams`: 440**; kcal scales with **`440 × 0.398`** against per-100 g cooked numbers.

---

## Quick reference

| Method   | Path                              | Purpose                          |
| -------- | --------------------------------- | -------------------------------- |
| `GET`    | `/help`                           | Route list + examples (no auth)  |
| `GET`    | `/foods`                          | List + aliases                   |
| `GET`    | `/foods/:id`                      | One food + aliases               |
| `POST`   | `/foods`                          | Create                           |
| `PUT`    | `/foods/:id`                      | Update                           |
| `DELETE` | `/foods/:id`                      | Delete (`replace_with` optional) |
| `POST`   | `/foods/batch`                    | Upsert + delete array            |
| `POST`   | `/foods/:id/aliases`              | Add alias                        |
| `DELETE` | `/foods/:id/aliases`              | Remove alias                     |
| `GET`    | `/gaps`                           | What still needs data            |
| `POST`   | `/fill`                           | Upsert foods + recompute meals   |
| `GET`    | `/export/json?scope=foods`        | Backup foods                     |
| `POST`   | `/import/json?scope=foods&merge=true` | Restore/merge                |