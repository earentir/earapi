# Health Tracker API

## Base

```
https://api.earentir.dev/health/v1
```

Web UI: `https://api.earentir.dev/health/ui/`  
`GET /health` redirects to the UI.

All `/health/v1` routes except **`GET /help`** require auth. JSON bodies use `Content-Type: application/json`. Dates are **`YYYY-MM-DD`**. Meal slots: **`lunch`**, **`dinner`**, **`snack`**.

Live route list: **`GET /help`** (no auth).

## Auth

From server `config/earapi.json` → `health.users[]` (`username` + `token`).

| Header          | Value              |
| --------------- | ------------------ |
| `X-User`        | username           |
| `X-API-Key`     | token              |
| `Accept`        | `application/json` |

GET-only alternative: `?user=&token=`  
Or `Authorization: Bearer <token>` **with** `X-User`.

- **401** `{"error":"unauthorized"}`
- **503** `{"error":"health auth not configured"}`

## Nutrition model

| Concept | Meaning |
| ------- | ------- |
| Per-100 g fields | Nutrition for **100 g edible** (label math). |
| `label_grams` | Pack/serving size on the label (usually 100). Not meal grams. |
| Meal line `grams` | What was weighed (may include bone/peel). |
| `edible_fraction` | Share of meal `grams` that scale per-100 g → line kcal/Na. |
| `legacy_totals` | Markdown day column (kcal / sodium). UI comparison only; not synthesized from current totals. |

```text
nutrition_grams = meal_grams × edible_fraction
line_kcal ≈ kcal_per_100g × nutrition_grams / 100
```

Values **> 1 and ≤ 100** for `edible_fraction` are treated as **percent**. Invalid → **1**. **Eggs** may be kcal-per-egg; server uses count/`×` heuristics.

Do not invent a second catalog row whose only difference is “(edible)” in the name — set yield on the same food.

Yield for a meal line: `MealItem.edible_fraction` → **1** if `kind` is **`packed`** → variant default cut → food `edible_fraction` → **1**.

---

## Core JSON shapes

### `Food`

`kind` is **`packed`** (shop pack label) or **`raw`** (skinless / with skin × raw / cooked). Empty stored kind is inferred on read: dual prep, `with_skin` data, or cut-level yield → **raw**; otherwise **packed**. New rows default **packed**. Top-level macros mirror the **default variant** after a **raw** save. Omitted fields are unchanged on `PUT` / `EnsureFood` merge.

```json
{
  "id": "yogurt-10pct",
  "name": "10% yogurt",
  "kind": "packed",
  "kcal_per_100g": 130,
  "sodium_mg_per_100g": 40,
  "protein_g_per_100g": 9.5,
  "energy_kj_per_100g": null,
  "fat_g_per_100g": null,
  "saturates_g_per_100g": null,
  "carbs_g_per_100g": null,
  "sugars_g_per_100g": null,
  "salt_g_per_100g": null,
  "label_grams": 150,
  "notes": "",
  "edible_fraction": null
}
```

Raw example keeps `variants` (cut × prep). Packed `PUT` should **omit** `variants` so stored cut/prep JSON is kept when switching back to raw. Packed macros update only the **default** variant slot.

- `kind`: `packed` \| `raw`.
- `default_cut`: `skinless` \| `with_skin`. `default_prep`: `raw` \| `cooked`.
- Sending **`variants`** on `PUT` **replaces** the whole object.
- New rows need **`kcal_per_100g` and/or `sodium_mg_per_100g`** unless updating by `id` or exact `name`.
- **`EnsureFood`** fuzzy-matches **`name`** before insert — prefer **`id`**.

### `MealItem`

```json
{
  "name": "Chicken leg",
  "food_id": "chicken-leg",
  "stock_lot_id": "",
  "grams": 270,
  "count": 0,
  "edible_fraction": 0.57,
  "kcal_per_100g": null,
  "sodium_mg_per_100g": null,
  "note": "",
  "parts": []
}
```

`parts[]` = a **set** (dish/batch). `grams`/`count` on the parent are the amount eaten of the set; parts are ingredients. Line `kcal_per_100g` / `sodium_mg_per_100g` can create/update catalog on log.

### `Day` / `MealSlot`

```json
{
  "date": "2026-11-18",
  "bp": "128/82",
  "walking": "park 40min",
  "meals": {
    "lunch": { "raw": "", "items": [], "kcal": 0, "sodium_mg": 0 },
    "dinner": { "items": [] },
    "snack": { "items": [] }
  },
  "totals": { "kcal": 0, "sodium_mg": 0 },
  "legacy_totals": { "kcal": 0, "sodium_mg": 0 },
  "display": null
}
```

`display` is server math (intake bands, item gaps). `kcal_margin` on slots is ignored.

### `WeighIn`

```json
{
  "id": "w_2026-11-18_1",
  "date": "2026-11-18",
  "kg": 142.5,
  "rhr_bpm": 58,
  "note": "morning",
  "protocol": true
}
```

`rhr_bpm` is resting heart rate (beats per minute). `protocol` = protocol weigh-in.

### `StockLot`

Prepped batch: `items[]` are full-batch ingredient lines. `batch_grams` / `remaining_grams` in grams.

---

## Quick reference

| Method | Path | Purpose |
| ------ | ---- | ------- |
| `GET` | `/help` | Routes + examples (no auth) |
| `GET` | `/overview?days=14` | Recent days + latest weight + settings |
| `GET` | `/gaps?from=&to=&limit=` | Missing catalog / unscorable meal lines |
| `POST` | `/fill` | Upsert foods + recompute meals |
| `POST` | `/nutrition/compute/day` | Scale a day JSON (no persist) |
| `POST` | `/nutrition/compute/meal` | Scale one slot (no persist) |
| `POST` | `/log` | One-shot day/meals/foods/weight write |
| `GET` | `/weight/latest` | Latest weigh-in |
| `GET` | `/weight?from=&to=` | List weigh-ins |
| `POST` | `/weight` | Add weigh-in |
| `PUT` | `/weight/:id` | Update |
| `DELETE` | `/weight/:id` | Delete |
| `GET` | `/foods?q=&limit=` | Catalog + aliases |
| `GET` | `/foods/:id` | One food + aliases |
| `POST` | `/foods` | Create |
| `PUT` | `/foods/:id` | Update |
| `DELETE` | `/foods/:id` | Delete (`replace_with` optional) |
| `POST` | `/foods/batch` | Upsert + delete arrays |
| `POST` | `/foods/:id/aliases` | Add alias |
| `DELETE` | `/foods/:id/aliases` | Remove alias |
| `GET` | `/days?from=&to=&date=&q=&slot=&limit=` | List days |
| `GET` | `/days/:date` | One day |
| `POST` | `/days` | Create or merge meals onto a date |
| `PUT` | `/days/:date` | Replace day |
| `PATCH` | `/days/:date` | Patch bp/walking/meals |
| `DELETE` | `/days/:date` | Delete day |
| `POST` | `/days/:date/meals/log` | Append/log a slot |
| `POST` | `/days/:date/meals` | Write a slot (`ensure_foods` optional) |
| `PUT` | `/days/:date/meals/:slot` | Replace slot |
| `DELETE` | `/days/:date/meals/:slot` | Clear slot |
| `GET` | `/stock?active=1` | List lots |
| `GET` | `/stock/:id` | One lot |
| `POST` | `/stock` | Create lot |
| `PUT` | `/stock/:id` | Update lot |
| `DELETE` | `/stock/:id` | Delete lot |
| `POST` | `/stock/:id/draw` | Log grams onto a meal slot |
| `POST` | `/stock/:id/adjust` | Remove grams, no meal (`household` \| `discarded`) |
| `PATCH` | `/stock/:id/active` | `{ "active": true\|false }` |
| `GET` | `/settings` | Bands + body profile; adds `bmr_kcal` / `sedentary_kcal` from latest kg |
| `PUT` | `/settings` | Save settings |
| `GET` | `/export/json?scope=` | `all` \| `foods` \| `days` \| `stock` \| `weight` |
| `GET` | `/import/json/status` | Import flags |
| `POST` | `/import/json?scope=&merge=` | Restore JSON |
| `POST` | `/import/markdown?replace=` | Import markdown days |
| `POST` | `/import/repair-days?from_md=1` | Re-parse raw, rescale, optional mddata legacy |

---

## Overview, gaps, fill, compute

### `GET /overview?days=14`

Dashboard blob: days in range, latest weight, settings.

### `GET /gaps`

| Query | Default |
| ----- | ------- |
| `from`, `to` | ISO dates |
| `limit` | `100` |

**200** — food gaps list **missing field names only** (no kcal numbers):

```json
{
  "foods": [{ "id": "...", "name": "...", "missing": ["kcal_per_100g"] }],
  "items": [{ "ref": "2026-11-11/dinner/2", "date": "...", "slot": "dinner", "name": "...", "grams": 150, "note": "missing_food" }],
  "food_count": 1,
  "item_count": 3
}
```

`note: "missing_food"` → unresolved/deleted `food_id`.

### `POST /fill`

```json
{
  "foods": [{ "name": "New spice", "kcal_per_100g": 250, "sodium_mg_per_100g": 800 }],
  "recompute_slots": [{ "date": "2026-11-11", "slot": "dinner" }],
  "recompute_days": ["2026-11-11"]
}
```

**200** `{ "foods_ensured", "foods_created", "reply", "recomputed", "gaps_remaining" }`

Typical: `GET /gaps` → `POST /fill` → check `gaps_remaining`.

### `POST /nutrition/compute/day`

Body: `Day`. Does not persist. If `legacy_totals` empty and `date` exists, stored legacy is used. Returns scaled meals + `display`.

### `POST /nutrition/compute/meal`

Body: `{ "date", "slot", "meal", "legacy_totals"? }`. Does not persist.

---

## Log (one-shot write)

### `POST /log`

```json
{
  "date": "2026-11-11",
  "bp": "128/82",
  "walking": "park 40min",
  "weight": { "kg": 142.5, "protocol": true, "note": "morning", "rhr_bpm": 58 },
  "ensure_foods": [],
  "meals": [
    { "slot": "lunch", "replace": true, "items": [{ "name": "Pasta", "grams": 120 }] },
    { "slot": "dinner", "items": [{ "name": "Salad", "grams": 250, "parts": [{ "name": "Tomato", "grams": 120 }] }] }
  ]
}
```

Also: `slot` + `items` (single meal), `raw`, `replace`, `delete_meals`, `delete_foods`, `delete_weights`, `delete_day`, `weight_update`.

Empty `date` → today (Athens). Server computes kcal/sodium.

---

## Weight

- **`GET /weight/latest`** → `{ "weight": WeighIn \| null }`
- **`GET /weight?from=&to=`** → `{ "weights", "count" }`
- **`POST /weight`** body `{ date, kg, note, rhr_bpm, protocol }` → **201** `{ "weight" }`
- **`PUT /weight/:id`** body `WeighIn`
- **`DELETE /weight/:id`** → `{ "deleted": "<id>" }`

---

## Foods

- **`GET /foods?q=&limit=`** → `{ "foods", "aliases", "count" }`
- **`GET /foods/:id`** → `{ "food", "aliases" }` — **404** if missing
- **`POST /foods`** — always insert (slug `id` if omitted) → **201** `{ "food" }`
- **`PUT /foods/:id`** — patch; URL `id` wins
- **`DELETE /foods/:id`** optional `{ "replace_with": "<keeper-id>" }` (or query). With replacement, aliases and old ids stay resolvable. Empty replacement leaves `missing_food` gaps. **`POST /foods/batch` `delete[]` never replaces.**
- **`POST /foods/batch`** `{ "upsert": [Food], "delete": ["id"] }` — each upsert is `EnsureFood`
- **`POST /foods/:id/aliases`** `{ "alias": "peppers" }` — stored lowercase
- **`DELETE /foods/:id/aliases`** `{ "alias": "peppers" }`

---

## Days and meals

- **`GET /days`** query `from`, `to`, `date`, `q`, `slot` (or `type`), `limit`
- **`GET /days/:date`** → `{ "day" }`
- **`POST /days`** create; if date exists, **appends** meal items
- **`PUT /days/:date`** full replace
- **`PATCH /days/:date`** partial
- **`DELETE /days/:date`**
- **`POST /days/:date/meals`** / **`.../meals/log`** — write/append a slot; optional `ensure_foods`
- **`PUT /days/:date/meals/:slot`** replace one slot
- **`DELETE /days/:date/meals/:slot`**

---

## Stock

- **`GET /stock?active=1`** — `active=1` filters fridge-on lots
- **`GET /stock/:id`**
- **`POST /stock`** / **`PUT /stock/:id`** — lot + `items[]` (batch recipe)
- **`DELETE /stock/:id`**
- **`POST /stock/:id/draw`** `{ "date", "slot", "grams" }` — deduct + log meal line
- **`POST /stock/:id/adjust`** `{ "grams", "mode": "household"|"discarded" }` — `grams` 0 = all remaining; no meal
- **`PATCH /stock/:id/active`** `{ "active": true }`

---

## Settings

**`GET /settings`** / **`PUT /settings`** body `KcalBudgetSettings`:

| Field | Role |
| ----- | ---- |
| `soft_low_kcal` / `soft_high_kcal` / `gain_min_kcal` | Intake bar bands |
| `soft_low_sodium_mg` / `soft_high_sodium_mg` / `ceiling_sodium_mg` | Sodium bands |
| `meal_margin_percent` | +% on catalog **day** total once |
| `legacy_diff_green_lt` / `_yellow_lt` / `_orange_lt` | \|catalog − markdown\| colors |
| `sex` / `age_years` / `height_cm` | Mifflin–St Jeor |
| `sedentary_discount_percent` | 0–50, lowers sedentary TDEE |

GET adds computed `bmr_kcal`, `sedentary_kcal`, `energy_weight_kg` from latest kg.

---

## Export / import

- **`GET /export/json?scope=all\|foods\|days\|stock\|weight`**
- **`GET /import/json/status`**
- **`POST /import/json?scope=&merge=true`** — `merge=false` replaces (dangerous)
- **`POST /import/json?merge=false&source=legacy`**
- **`POST /import/markdown?replace=true`**
- **`POST /import/repair-days?from_md=1`** — re-parse meal raw, rescale; optional mddata legacy backfill

Never `merge=false` or bulk delete without explicit user confirmation.

---

## LLM / client rules

1. Read before write (`GET /foods/:id`, `GET /days/:date`).
2. Prefer `id` + `PUT` for known foods; `POST /fill` or batch `upsert` for gaps.
3. Do not bulk-delete or `import?merge=false` unless the user asked.
4. Deduping foods: aliases on the keeper, then `DELETE` duplicate with `replace_with`.
5. Meal `grams` are as-weighed; yield is `edible_fraction`, not a name suffix.

```bash
curl -sS 'https://api.earentir.dev/health/v1/foods?q=pepper&limit=10' \
  -H 'X-User: USER' -H 'X-API-Key: TOKEN'
```
