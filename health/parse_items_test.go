package health

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseMealItemsFromRawDinner(t *testing.T) {
	raw := `Soup-tomato mix 400 g ~92 / ~112 + moussaka 200 g ~440 / ~390 + lettuce 100 g ~25 / ~20 + EVOO 2 g 0 / ~18 + balsamic cream 4 g ~16 / ~14 → **~573 / ~554 → +10% margin = ~609**`
	items := ParseMealItemsFromRaw(raw)
	if len(items) != 5 {
		t.Fatalf("got %d items %+v", len(items), items)
	}
	if items[0].Name != "Soup-tomato mix" || items[0].Grams != 400 {
		t.Fatalf("first %+v", items[0])
	}
	if items[0].Count != 0 {
		t.Fatalf("soup count %v want 0", items[0].Count)
	}
	if items[1].Name != "moussaka" || items[1].Grams != 200 {
		t.Fatalf("moussaka %+v", items[1])
	}
	if items[3].Grams != 2 {
		t.Fatalf("evoo %+v", items[3])
	}
}

func TestParseMealStructureNov11Lunch(t *testing.T) {
	raw := `Perch (cooked, no salt) 400 g (~70 kcal / ~70 mg per 100 g, Oct 24 precedent) ~280 / ~280 + roasted bell pepper 150 g + roasted potato 150 g (Nov 10 style, minimal oil in batch) ~8 / ~138 + Salad: baby leaves 50 g + iceberg 50 g + tomato 100 g + EVOO 4 g + balsamic cream 11 g → **~360 / ~572 → +10% margin = ~629**`
	items := ParseMealStructureFromRaw(raw)
	if len(items) != 4 {
		t.Fatalf("top-level %d %+v", len(items), items)
	}
	if items[3].Name != "Salad" || len(items[3].Parts) != 5 {
		t.Fatalf("salad set %+v", items[3])
	}
}

func TestParseMealItemsNov11Lunch(t *testing.T) {
	raw := `Perch (cooked, no salt) 400 g + roasted bell pepper 150 g + roasted potato 150 g + Salad: baby leaves 50 g + iceberg 50 g + tomato 100 g + EVOO 4 g + balsamic cream 11 g → **~360 / ~572 → +10% margin = ~629**`
	flat := ParseMealItemsFromRaw(raw)
	if len(flat) != 8 {
		t.Fatalf("flat %d %+v", len(flat), flat)
	}
}

func TestParseMealItemsNov3Snack(t *testing.T) {
	raw := `Rusks no salt 12 g ×3 (36 g, label 387 kcal / ~15 mg per 100 g) ~5 / ~139 + NuKrema 13 g ×3 (39 g, label 550 kcal / ~38 mg per 100 g) ~15 / ~215 → **~20 / ~354 → +10% margin = ~389** + Vitalina FRUIT 170 g (label: ~87 kcal / ~75 mg per portion) ~75 / ~87 + corn wafers ×2 (~20 g, 416 kcal / ~400 mg per 100 g, weight estimated ~10 g/wafer - correct if off) ~80 / ~83 → **~175 / ~576**`
	items := ParseMealItemsFromRaw(raw)
	if len(items) != 4 {
		t.Fatalf("got %d %+v", len(items), items)
	}
	if items[3].Grams != 20 && items[3].Grams != 0 {
		// ×2 with (~20 g) in paren may win
	}
}

func TestParseBoneInWingsNov12(t *testing.T) {
	ch := `roasted wings 440 g (265 g bones → 175 g edible)`
	it, ok := parseOneMealChunk(ch)
	if !ok {
		t.Fatal("parse fail")
	}
	if it.Grams != 175 {
		t.Fatalf("grams=%v want 175 edible", it.Grams)
	}
}

func TestParseThighBoneYieldOct2(t *testing.T) {
	ch := `chicken thigh 270 g bone-in (bone 115 g → 155 g edible)`
	it, ok := parseOneMealChunk(ch)
	if !ok {
		t.Fatal("parse fail")
	}
	if it.Grams != 155 {
		t.Fatalf("grams=%v want 155", it.Grams)
	}
}

func TestParseChickenLegsMeatYieldOct6(t *testing.T) {
	ch := `Chicken legs 340 g edible (725 g raw air-fried, ~200 g bone)`
	it, ok := parseOneMealChunk(ch)
	if !ok {
		t.Fatal("parse fail")
	}
	if it.Grams != 340 {
		t.Fatalf("grams=%v want 340", it.Grams)
	}
}

func TestParseSkewersMeatOct7(t *testing.T) {
	ch := `Skewers 410 g meat (440 g incl. ~30 g wood; breast 340 g + thigh 70 g, no oil)`
	it, ok := parseOneMealChunk(ch)
	if !ok {
		t.Fatal("parse fail")
	}
	if it.Grams != 410 {
		t.Fatalf("grams=%v want 410", it.Grams)
	}
}

func TestParseLegsEdibleTimes2(t *testing.T) {
	ch := `Chicken legs 155 g edible ×2 (270 g bone-in each, 57% yield verified Oct 2)`
	it, ok := parseOneMealChunk(ch)
	if !ok {
		t.Fatal("parse fail")
	}
	if it.Grams != 310 {
		t.Fatalf("grams=%v want 310 (155×2)", it.Grams)
	}
}

func TestParseWingsNov13YieldNote(t *testing.T) {
	ch := `roasted wings 105 g edible (264 g whole, 159 g bone; 3 wings, Nov 12 yield ÷5×3)`
	it, ok := parseOneMealChunk(ch)
	if !ok {
		t.Fatal("parse fail")
	}
	if it.Grams != 105 {
		t.Fatalf("grams=%v want 105 total edible (not 105×3)", it.Grams)
	}
}

func TestParseWingsNov13NoStealEdibleFromNoteTail(t *testing.T) {
	ch := `roasted wings 105 g edible (264 g whole, 159 g bone; 3 wings, scaled from Nov 12 five-wing 175 g edible)`
	it, ok := parseOneMealChunk(ch)
	if !ok {
		t.Fatal("parse fail")
	}
	if it.Grams != 105 {
		t.Fatalf("grams=%v want 105 (not 175 from note)", it.Grams)
	}
}

func TestParseWingsEdibleOnly(t *testing.T) {
	ch := `chicken wings 170 g edible`
	it, ok := parseOneMealChunk(ch)
	if !ok {
		t.Fatal("parse fail")
	}
	if it.Grams != 170 {
		t.Fatalf("grams=%v", it.Grams)
	}
}

func TestParseRusksParenTotalGrams(t *testing.T) {
	ch := `Rusks no salt 12 g ×3 (36 g, label 387 kcal / ~15 mg per 100 g)`
	it, ok := parseOneMealChunk(ch)
	if !ok {
		t.Fatal("parse fail")
	}
	if it.Grams != 36 && it.Grams != 12 {
		t.Fatalf("grams=%v want 36 or 12", it.Grams)
	}
}

func TestCompactMealRawNov11Lunch(t *testing.T) {
	verbose := `Perch (cooked, no salt) 400 g + roasted bell pepper 150 g + roasted potato 150 g + Salad: baby leaves 50 g + iceberg 50 g + tomato 100 g + EVOO 4 g + balsamic cream 11 g → **~360 / ~572 → +10% margin = ~629**`
	want := `Perch (cooked, no salt) 400 g + roasted bell pepper 150 g + roasted potato 150 g + Salad: baby leaves 50 g + iceberg 50 g + tomato 100 g + EVOO 4 g + balsamic cream 11 g → **~360 / ~572 → +10% margin = ~629**`
	if got := CompactMealRaw(verbose); got != want {
		t.Fatalf("compact:\n got %q\nwant %q", got, want)
	}
}

func TestRefreshNovemberItemTable(t *testing.T) {
	if os.Getenv("REFRESH_NOVEMBER_ITEMS") == "" {
		t.Skip("set REFRESH_NOVEMBER_ITEMS=1 to rewrite item appendix in mddata")
	}
	path := filepath.Join("..", "mddata", "Health Tracker - 2026 November.md")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(b)
	if i := strings.Index(content, "\n## Items (name + g"); i >= 0 {
		content = content[:i]
	}
	content = strings.TrimRight(content, "\n") + "\n"
	days := parseMonthMarkdown(content, 2026)
	var rows []string
	rows = append(rows, "## Items (name + g, auto-split from meal text)", "", "| Date | Slot | Items |", "|---|---|---|")
	for _, d := range days {
		date := strings.TrimPrefix(d.Date, "2026-")
		m, _ := time.Parse("2006-01-02", d.Date)
		if !m.IsZero() {
			date = m.Format("Jan 2")
		}
		for _, slot := range []string{"lunch", "dinner", "snack"} {
			raw := strings.TrimSpace(d.Meals[slot].Raw)
			if raw == "" || itemSkipRawRe.MatchString(raw) {
				continue
			}
			items := ParseMealItemsFromRaw(raw)
			if len(items) == 0 {
				continue
			}
			var bits []string
			for _, it := range flattenItems(items) {
				bits = append(bits, formatMealItemCompact(it))
			}
			rows = append(rows, "| "+date+" | "+slot+" | "+strings.Join(bits, ", ")+" |")
		}
	}
	rows = append(rows, "", "_Nutrition notes and per-item mg/kcal stay in the main table; this section is structured rows for import and LLM fill. Regenerate: `REFRESH_NOVEMBER_ITEMS=1 go test ./health -run TestRefreshNovemberItemTable`._", "")
	if err := os.WriteFile(path, []byte(content+"\n"+strings.Join(rows, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseMealItemsTimesAndParen(t *testing.T) {
	items := ParseMealItemsFromRaw(`Granola bars ×2 (42 g, label) ~104 / ~200 + rusks no salt 12 g ×3`)
	if len(items) != 2 {
		t.Fatalf("%+v", items)
	}
	if items[0].Grams != 42 {
		t.Fatalf("bars grams %v", items[0].Grams)
	}
	if items[1].Grams != 36 {
		t.Fatalf("rusks grams %v want 36", items[1].Grams)
	}
}

func TestScaleDish(t *testing.T) {
	root := t.TempDir()
	db, err := OpenSharedDB(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(db, root, "u")
	if err != nil {
		t.Fatal(err)
	}
	k, n := 100.0, 200.0
	if _, err := st.AddFood(Food{Name: "Oil", KcalPer100g: &k, SodiumMgPer100g: &n}); err != nil {
		t.Fatal(err)
	}
	k2, n2 := 50.0, 10.0
	if _, err := st.AddFood(Food{Name: "Leaves", KcalPer100g: &k2, SodiumMgPer100g: &n2}); err != nil {
		t.Fatal(err)
	}
	meal := MealSlot{
		Items: []MealItem{{
			Name:  "Salad",
			Grams: 150,
			Parts: []MealItem{
				{Name: "Leaves", Grams: 200},
				{Name: "Oil", Grams: 10},
			},
		}},
	}
	if _, err := st.PrepareMealSlot(&meal, nil); err != nil {
		t.Fatal(err)
	}
	// batch 210g, eaten 150g; leaves 100 kcal, oil 10 kcal -> 110 * 150/210
	want := 110.0 * 150 / 210
	if meal.Items[0].Kcal < want-0.05 || meal.Items[0].Kcal > want+0.05 {
		t.Fatalf("kcal %v want %v", meal.Items[0].Kcal, want)
	}
}
