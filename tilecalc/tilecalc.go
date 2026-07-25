// Package tilecalc calculates tile arrangements and coverage for a given tile size.
package tilecalc

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Layout is one rows×cols arrangement of a fixed tile count.
type Layout struct {
	Rows   int     `json:"rows"`
	Cols   int     `json:"cols"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Unit   string  `json:"unit"`
	Graph  string  `json:"graph,omitempty"`
}

// PricingResult is derived cost info when a price is provided.
type PricingResult struct {
	Price        float64 `json:"price"`
	Per          int     `json:"per"`
	PricePerTile float64 `json:"price_per_tile"`
	CostPerM2    float64 `json:"cost_per_m2"`
	Tiles        int     `json:"tiles"`
	PacksNeeded  int     `json:"packs_needed"`
	TotalCost    float64 `json:"total_cost"`
	TileAreaM2   float64 `json:"tile_area_m2"`
}

// ArrangeResult is the response for arrangement mode.
type ArrangeResult struct {
	TileWidth  int            `json:"tile_width_cm"`
	TileHeight int            `json:"tile_height_cm"`
	Count      int            `json:"count"`
	TotalAreaM float64        `json:"total_area_m2"`
	Layouts    []Layout       `json:"layouts"`
	Pricing    *PricingResult `json:"pricing,omitempty"`
}

// CutSpec counts cut pieces of a given size.
type CutSpec struct {
	Size  string `json:"size"`
	Count int    `json:"count"`
}

// CoveragePattern is one orientation’s coverage breakdown.
type CoveragePattern struct {
	TileWidth  int            `json:"tile_width_cm"`
	TileHeight int            `json:"tile_height_cm"`
	TotalTiles int            `json:"total_tiles"`
	FullTiles  int            `json:"full_tiles"`
	Cuts       []CutSpec      `json:"cuts,omitempty"`
	Graph      string         `json:"graph,omitempty"`
	Pricing    *PricingResult `json:"pricing,omitempty"`
}

// CoverageResult is the response for coverage mode.
type CoverageResult struct {
	SpaceWidth  int               `json:"space_width_cm"`
	SpaceHeight int               `json:"space_height_cm"`
	SpaceAreaM2 float64           `json:"space_area_m2"`
	Patterns    []CoveragePattern `json:"patterns"`
}

// Options controls unit conversion, split filters, graphs, and pricing.
type Options struct {
	MinSplit               int
	MaxSplit               int
	MinSplitSet            bool
	MaxSplitSet            bool
	ToMeters               bool
	ToInches               bool
	Graph                  bool
	SingleDimensionPattern bool
	// Price is the amount charged for Per tiles. Per defaults to 1 (per tile).
	// Examples: Price=16, Per=1 → 16/tile; Price=16, Per=6 → 16 per pack of 6.
	Price    float64
	Per      int
	HasPrice bool
}

// ParseDimensions turns "123x456" into two ints.
func ParseDimensions(s string) (int, int, error) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(s)), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected WxH, got %q", s)
	}
	w, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	h, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return w, h, nil
}

// NormalizeTileSize applies square-tile logic and validates dimensions.
func NormalizeTileSize(width, height int) (int, int, error) {
	if width > 0 && height == 0 {
		height = width
	}
	if height > 0 && width == 0 {
		width = height
	}
	if width <= 0 || height <= 0 {
		return 0, 0, fmt.Errorf("must specify size (e.g. 15x40) or width/height")
	}
	return width, height, nil
}

// Arrange computes all valid row/col layouts for count tiles of size w×h.
func Arrange(width, height, count int, opts Options) (*ArrangeResult, error) {
	width, height, err := NormalizeTileSize(width, height)
	if err != nil {
		return nil, err
	}
	if count <= 0 {
		return nil, fmt.Errorf("must specify a positive count")
	}
	if err := validatePricing(opts); err != nil {
		return nil, err
	}

	combos := calculateCombos(count, opts.MinSplit, opts.MaxSplit, opts.MinSplitSet, opts.MaxSplitSet)
	factor, unit := unitConversion(opts)

	layouts := make([]Layout, 0, len(combos))
	for _, c := range combos {
		l := Layout{
			Rows:   c.Rows,
			Cols:   c.Cols,
			Width:  float64(c.Cols*width) * factor,
			Height: float64(c.Rows*height) * factor,
			Unit:   unit,
		}
		if opts.Graph {
			l.Graph = asciiGrid(c.Rows, c.Cols)
		}
		layouts = append(layouts, l)
	}

	result := &ArrangeResult{
		TileWidth:  width,
		TileHeight: height,
		Count:      count,
		TotalAreaM: float64(count*width*height) / 10000.0,
		Layouts:    layouts,
	}
	if opts.HasPrice {
		result.Pricing = calculatePricing(width, height, count, opts)
	}
	return result, nil
}

// Coverage computes how many tiles (and cuts) fill a space.
func Coverage(tileW, tileH, spaceW, spaceH int, opts Options) (*CoverageResult, error) {
	tileW, tileH, err := NormalizeTileSize(tileW, tileH)
	if err != nil {
		return nil, err
	}
	if spaceW <= 0 || spaceH <= 0 {
		return nil, fmt.Errorf("must specify positive space dimensions")
	}
	if err := validatePricing(opts); err != nil {
		return nil, err
	}

	patterns := []CoveragePattern{calculateCoverage(tileW, tileH, spaceW, spaceH, opts)}
	if !opts.SingleDimensionPattern && tileW != tileH {
		patterns = append(patterns, calculateCoverage(tileH, tileW, spaceW, spaceH, opts))
	}

	return &CoverageResult{
		SpaceWidth:  spaceW,
		SpaceHeight: spaceH,
		SpaceAreaM2: float64(spaceW*spaceH) / 10000.0,
		Patterns:    patterns,
	}, nil
}

func validatePricing(opts Options) error {
	if !opts.HasPrice {
		return nil
	}
	if opts.Price < 0 {
		return fmt.Errorf("price must be non-negative")
	}
	per := opts.Per
	if per == 0 {
		per = 1
	}
	if per < 1 {
		return fmt.Errorf("per must be a positive integer (tiles covered by price)")
	}
	return nil
}

func calculatePricing(tileW, tileH, tiles int, opts Options) *PricingResult {
	per := opts.Per
	if per < 1 {
		per = 1
	}
	tileArea := float64(tileW*tileH) / 10000.0
	pricePerTile := opts.Price / float64(per)
	costPerM2 := 0.0
	if tileArea > 0 {
		costPerM2 = pricePerTile / tileArea
	}
	packs := int(math.Ceil(float64(tiles) / float64(per)))
	return &PricingResult{
		Price:        opts.Price,
		Per:          per,
		PricePerTile: pricePerTile,
		CostPerM2:    costPerM2,
		Tiles:        tiles,
		PacksNeeded:  packs,
		TotalCost:    float64(packs) * opts.Price,
		TileAreaM2:   tileArea,
	}
}

type layout struct{ Rows, Cols int }

func calculateCombos(count, minS, maxS int, minSet, maxSet bool) []layout {
	var combos []layout
	for rows := 1; rows <= count; rows++ {
		if count%rows != 0 {
			continue
		}
		cols := count / rows
		if minSet && (rows < minS || cols < minS) {
			continue
		}
		if maxSet && rows > maxS && cols > maxS {
			continue
		}
		combos = append(combos, layout{Rows: rows, Cols: cols})
	}
	return combos
}

func unitConversion(opts Options) (float64, string) {
	if opts.ToMeters {
		return 0.01, "m"
	}
	if opts.ToInches {
		return 1.0 / 2.54, "in"
	}
	return 1.0, "cm"
}

func asciiGrid(rows, cols int) string {
	var b strings.Builder
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			b.WriteString("[]")
		}
		if r < rows-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func calculateCoverage(tileW, tileH, spaceW, spaceH int, opts Options) CoveragePattern {
	fullCols := spaceW / tileW
	remW := spaceW % tileW
	fullRows := spaceH / tileH
	remH := spaceH % tileH

	cols := fullCols
	if remW > 0 {
		cols++
	}
	rows := fullRows
	if remH > 0 {
		rows++
	}

	fullCount := 0
	cutsMap := make(map[string]int)

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			w := tileW
			if c == fullCols && remW > 0 {
				w = remW
			}
			h := tileH
			if r == fullRows && remH > 0 {
				h = remH
			}
			if w == tileW && h == tileH {
				fullCount++
			} else {
				key := fmt.Sprintf("%dx%d", w, h)
				cutsMap[key]++
			}
		}
	}

	cuts := make([]CutSpec, 0, len(cutsMap))
	for size, cnt := range cutsMap {
		cuts = append(cuts, CutSpec{Size: size, Count: cnt})
	}

	totalTiles := rows * cols
	p := CoveragePattern{
		TileWidth:  tileW,
		TileHeight: tileH,
		TotalTiles: totalTiles,
		FullTiles:  fullCount,
		Cuts:       cuts,
	}
	if opts.Graph {
		p.Graph = asciiGrid(rows, cols)
	}
	if opts.HasPrice {
		p.Pricing = calculatePricing(tileW, tileH, totalTiles, opts)
	}
	return p
}
