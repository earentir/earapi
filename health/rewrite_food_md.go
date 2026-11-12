package health

import (
	"fmt"
	"strings"
)

// RewriteFoodDataMarkdown rewrites catalog markdown to EU label columns and shorter notes.
func RewriteFoodDataMarkdown(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	inRefTable := false
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.Contains(line, "Reference kcal") || strings.Contains(line, "Reference per 100 g") {
			out = append(out, "## Reference per 100 g (EU nutrition declaration)")
			out = append(out, "")
			out = append(out, euTableHeader())
			inRefTable = true
			for i+1 < len(lines) && (strings.HasPrefix(lines[i+1], "|") || lines[i+1] == "") {
				i++
				if strings.Contains(lines[i], "---") {
					break
				}
			}
			continue
		}
		if inRefTable && strings.HasPrefix(line, "## ") {
			inRefTable = false
			out = append(out, line)
			continue
		}
		if inRefTable && strings.HasPrefix(line, "|") && !strings.Contains(line, "---") {
			cells := splitTableRow(line)
			if len(cells) >= 3 && cells[0] != "Food" {
				var f Food
				if isEUNutritionTableRow(cells) {
					f = foodFromEUTableCells(cells)
				} else {
					f = foodFromLegacyTableCells(cells)
				}
				out = append(out, euTableRow(f))
				continue
			}
		}
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "- ") {
			text := strings.TrimPrefix(trim, "- ")
			if len(text) < 8 || skipFoodBulletForCatalog(text) {
				out = append(out, line)
				continue
			}
			name, body := bulletNameBody(text)
			f := Food{Name: name, Notes: body}
			kcal, sod := parseNutrientsFromNotes(text)
			f.KcalPer100g, f.SodiumMgPer100g = kcal, sod
			parsed := ParseEULabelFromText(text)
			mergeFoodLabel(&f, parsed)
			ApplyLabelFromNotes(&f)
			if strings.TrimSpace(f.Notes) == "" {
				f.Notes = residualFoodNotes(text)
			}
			NormalizeFoodLabel(&f)
			out = append(out, bulletLine(f))
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func isEUNutritionTableRow(cells []string) bool {
	return len(cells) >= 9 || (len(cells) >= 2 && strings.Contains(cells[1], "kJ"))
}

func euTableHeader() string {
	return "| Food | Energy | Fat | of which saturates | Carbohydrate | of which sugars | Protein | Salt | Notes |"
}

func foodFromLegacyTableCells(cells []string) Food {
	name := strings.TrimSpace(cells[0])
	f := Food{Name: name}
	if len(cells) > 3 {
		f.Notes = strings.TrimSpace(cells[3])
	}
	f.KcalPer100g = parseApproxNumber(cells[1])
	f.SodiumMgPer100g = parseApproxNumber(cells[2])
	ApplyLabelFromNotes(&f)
	return f
}

func foodFromEUTableCells(cells []string) Food {
	name := strings.TrimSpace(cells[0])
	f := Food{Name: name}
	cell := func(i int) string {
		if i < len(cells) {
			return cells[i]
		}
		return ""
	}
	if len(cells) > 8 {
		f.Notes = strings.TrimSpace(cells[8])
	}
	parseEnergyCell(cell(1), &f)
	f.FatGPer100g = parseGCell(cell(2))
	f.SaturatesGPer100g = parseGCell(cell(3))
	f.CarbsGPer100g = parseGCell(cell(4))
	f.SugarsGPer100g = parseGCell(cell(5))
	f.ProteinGPer100g = parseGCell(cell(6))
	f.SaltGPer100g = parseGCell(cell(7))
	NormalizeFoodLabel(&f)
	return f
}

func parseEnergyCell(s string, f *Food) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	if m := reLabelKcal.FindStringSubmatch(s); len(m) > 1 {
		v := parseFloatLoose(m[1])
		f.KcalPer100g = &v
	}
	if i := strings.Index(s, "kJ"); i > 0 {
		j := parseFloatLoose(strings.TrimSpace(s[:i]))
		f.EnergyKjPer100g = &j
	}
}

func parseGCell(s string) *float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	v := parseFloatLoose(strings.TrimSuffix(strings.TrimSuffix(s, " g"), "g"))
	return &v
}

func euTableRow(f Food) string {
	return fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s | %s | %s |",
		escMDCell(f.Name),
		escMDCell(formatEnergyCell(f)),
		escMDCell(formatGCell(f.FatGPer100g)),
		escMDCell(formatGCell(f.SaturatesGPer100g)),
		escMDCell(formatGCell(f.CarbsGPer100g)),
		escMDCell(formatGCell(f.SugarsGPer100g)),
		escMDCell(formatGCell(f.ProteinGPer100g)),
		escMDCell(formatSaltCell(f)),
		escMDCell(f.Notes),
	)
}

func bulletLine(f Food) string {
	label := formatEULabelInline(f)
	prefix := f.Name
	if !strings.HasPrefix(prefix, "**") {
		prefix = "**" + prefix + "**"
	}
	rest := strings.TrimSpace(f.Notes)
	if label != "" && rest != "" {
		return fmt.Sprintf("- %s: %s — %s", prefix, label, rest)
	}
	if label != "" {
		return fmt.Sprintf("- %s: %s", prefix, label)
	}
	return fmt.Sprintf("- %s: %s", prefix, rest)
}

func formatEnergyCell(f Food) string {
	if f.KcalPer100g == nil {
		return ""
	}
	NormalizeFoodLabel(&f)
	if f.EnergyKjPer100g != nil {
		return fmt.Sprintf("%.0f kJ / %.0f kcal", *f.EnergyKjPer100g, *f.KcalPer100g)
	}
	return fmt.Sprintf("%.0f kcal", *f.KcalPer100g)
}

func formatGCell(v *float64) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%.1f g", *v)
}

func formatSaltCell(f Food) string {
	if f.SaltGPer100g != nil {
		return fmt.Sprintf("%.2f g", *f.SaltGPer100g)
	}
	if f.SodiumMgPer100g != nil {
		return fmt.Sprintf("%.2f g", *f.SodiumMgPer100g/400)
	}
	return ""
}

func formatEULabelInline(f Food) string {
	var parts []string
	if e := formatEnergyCell(f); e != "" {
		parts = append(parts, "Energy "+e)
	}
	if f.FatGPer100g != nil {
		parts = append(parts, fmt.Sprintf("Fat %.1f g", *f.FatGPer100g))
	}
	if f.SaturatesGPer100g != nil {
		parts = append(parts, fmt.Sprintf("Sat %.1f g", *f.SaturatesGPer100g))
	}
	if f.CarbsGPer100g != nil {
		parts = append(parts, fmt.Sprintf("Carbs %.1f g", *f.CarbsGPer100g))
	}
	if f.SugarsGPer100g != nil {
		parts = append(parts, fmt.Sprintf("Sugars %.1f g", *f.SugarsGPer100g))
	}
	if f.ProteinGPer100g != nil {
		parts = append(parts, fmt.Sprintf("Protein %.1f g", *f.ProteinGPer100g))
	}
	if s := formatSaltCell(f); s != "" {
		parts = append(parts, "Salt "+s)
	}
	return strings.Join(parts, ", ")
}

func escMDCell(s string) string {
	return strings.ReplaceAll(s, "|", "/")
}

func bulletNameBody(text string) (string, string) {
	name := text
	body := text
	if idx := strings.Index(text, ":"); idx > 0 && idx < 80 {
		name = strings.TrimSpace(text[:idx])
		body = strings.TrimSpace(text[idx+1:])
	} else if idx := strings.Index(text, " ("); idx > 0 && idx < 80 {
		name = strings.TrimSpace(text[:idx])
	}
	name = cleanFoodDisplayName(name)
	return name, body
}
