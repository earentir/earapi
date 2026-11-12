package health

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// go test ./health -run TestFoodCoverageReport -v
func TestFoodCoverageReport(t *testing.T) {
	dir := filepath.Join("..", "mddata")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("mddata not present")
	}
	_, foods, days, stock, err := ImportFromMDDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	type issue struct {
		kind   string
		name   string
		detail string
		count  int
	}
	issues := map[string]*issue{}

	add := func(kind, name, detail string) {
		key := kind + "\x00" + name + "\x00" + detail
		if issues[key] == nil {
			issues[key] = &issue{kind: kind, name: name, detail: detail}
		}
		issues[key].count++
	}

	var walk func(items []MealItem)
	walk = func(items []MealItem) {
		for _, it := range items {
			if len(it.Parts) > 0 {
				walk(it.Parts)
				continue
			}
			name := strings.TrimSpace(it.Name)
			if name == "" {
				continue
			}
			if it.Grams <= 0 && !strings.Contains(name, " ") {
				continue
			}
			if itemNeedsNutritionLeaf(it, foods) {
				f, ok := BestFoodMatch(foods, name)
				if !ok {
					add("unmatched", name, "no catalog match")
				} else if f.KcalPer100g == nil && f.SodiumMgPer100g == nil {
					add("no data", name, "matched "+f.Name+" but both kcal/Na missing")
				} else if f.KcalPer100g == nil {
					add("no kcal", name, "matched "+f.Name)
				} else if f.SodiumMgPer100g == nil {
					add("no sodium", name, "matched "+f.Name)
				} else {
					add("gap", name, "matched "+f.Name+" — partial/score gap")
				}
			}
		}
	}
	for _, d := range days {
		for _, m := range d.Meals {
			walk(m.Items)
		}
	}
	for _, lot := range stock {
		walk(lot.Items)
		if lot.KcalPer100g == nil || lot.SodiumMgPer100g == nil {
			add("stock lot", lot.Name, fmt.Sprintf("kcal=%v sod=%v", lot.KcalPer100g, lot.SodiumMgPer100g))
		}
	}

	var catalog []FoodGap
	for _, g := range foodGaps(foods) {
		catalog = append(catalog, g)
	}
	sort.Slice(catalog, func(i, j int) bool {
		return catalog[i].Name < catalog[j].Name
	})

	list := make([]*issue, 0, len(issues))
	for _, v := range issues {
		list = append(list, v)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].kind != list[j].kind {
			return list[i].kind < list[j].kind
		}
		if list[i].count != list[j].count {
			return list[i].count > list[j].count
		}
		return list[i].name < list[j].name
	})

	t.Logf("=== Catalog foods missing per 100 g (%d) ===", len(catalog))
	for _, g := range catalog {
		t.Logf("  %s — missing %v", g.Name, g.Missing)
	}

	byKind := map[string][]*issue{}
	for _, v := range list {
		byKind[v.kind] = append(byKind[v.kind], v)
	}
	for _, kind := range []string{"unmatched", "no data", "no kcal", "no sodium", "gap", "stock lot"} {
		rows := byKind[kind]
		if len(rows) == 0 {
			continue
		}
		t.Logf("=== Meal/stock lines: %s (%d distinct) ===", kind, len(rows))
		limit := 40
		for i, v := range rows {
			if i >= limit {
				t.Logf("  … and %d more", len(rows)-limit)
				break
			}
			t.Logf("  %dx %s — %s", v.count, v.name, v.detail)
		}
	}
}

func TestTopUnmatchedMealNames(t *testing.T) {
	dir := filepath.Join("..", "mddata")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("mddata not present")
	}
	_, foods, days, _, _ := ImportFromMDDir(dir)
	counts := map[string]int{}
	var walk func(items []MealItem)
	walk = func(items []MealItem) {
		for _, it := range items {
			if len(it.Parts) > 0 {
				walk(it.Parts)
				continue
			}
			name := strings.TrimSpace(it.Name)
			if name == "" || it.Grams <= 0 {
				continue
			}
			if strings.Contains(strings.ToLower(name), "kcal") || strings.Contains(name, "/") {
				continue
			}
			if len(name) > 55 {
				continue
			}
			f, ok := BestFoodMatch(foods, name)
			if ok && f.KcalPer100g != nil && f.SodiumMgPer100g != nil {
				continue
			}
			if ok {
				counts["[incomplete] "+name+" → "+f.Name]++
				continue
			}
			counts[name]++
		}
	}
	for _, d := range days {
		for _, m := range d.Meals {
			walk(m.Items)
		}
	}
	type kv struct {
		k string
		v int
	}
	var list []kv
	for k, v := range counts {
		list = append(list, kv{k, v})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].v > list[j].v })
	t.Log("Top lines still missing catalog or per 100 g (grams>0, import structure):")
	for i, x := range list {
		if i >= 45 {
			t.Logf("… %d more distinct names", len(list)-45)
			break
		}
		t.Logf("  %4d  %s", x.v, x.k)
	}
}
