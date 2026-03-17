package inventory

import (
	"embed"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/TrueBlocks/trueblocks-art/packages/color"
)

//go:embed paints.csv
var paintsCSV embed.FS

// PaintProduct represents a physical acrylic paint product
type PaintProduct struct {
	ID       string    `json:"id"`
	Brand    string    `json:"brand"`
	Name     string    `json:"name"`
	Series   int       `json:"series"`
	Opacity  string    `json:"opacity"`
	Pigments string    `json:"pigments"`
	RGB      [3]uint8  `json:"rgb"`
	Hex      string    `json:"hex"`
	Lab      color.Lab `json:"lab"`
}

// LoadPaints loads and parses the embedded paint CSV data
func LoadPaints() ([]PaintProduct, error) {
	file, err := paintsCSV.Open("paints.csv")
	if err != nil {
		return nil, fmt.Errorf("failed to open embedded paints.csv: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Skip header row
	_, err = reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	var paints []PaintProduct

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV record: %w", err)
		}

		// Parse record: id,brand,name,series,opacity,pigments,r,g,b
		if len(record) < 9 {
			continue // Skip malformed rows
		}

		series, _ := strconv.Atoi(record[3])
		r, _ := strconv.Atoi(record[6])
		g, _ := strconv.Atoi(record[7])
		b, _ := strconv.Atoi(record[8])

		// Clamp RGB values
		r = clamp(r, 0, 255)
		g = clamp(g, 0, 255)
		b = clamp(b, 0, 255)

		rgb := [3]uint8{uint8(r), uint8(g), uint8(b)}

		// Convert RGB to Lab for Delta E calculations
		lab := color.RGBToLab(uint8(r), uint8(g), uint8(b))

		paint := PaintProduct{
			ID:       strings.TrimSpace(record[0]),
			Brand:    strings.TrimSpace(record[1]),
			Name:     strings.TrimSpace(record[2]),
			Series:   series,
			Opacity:  strings.TrimSpace(record[4]),
			Pigments: strings.TrimSpace(record[5]),
			RGB:      rgb,
			Hex:      fmt.Sprintf("#%02X%02X%02X", r, g, b),
			Lab:      lab,
		}

		paints = append(paints, paint)
	}

	return paints, nil
}

// clamp limits a value to min/max bounds
func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// GetBrands returns a list of unique brand names
func GetBrands(paints []PaintProduct) []string {
	brandMap := make(map[string]bool)
	for _, p := range paints {
		brandMap[p.Brand] = true
	}

	var brands []string
	for brand := range brandMap {
		brands = append(brands, brand)
	}
	return brands
}

// FilterByBrand returns paints filtered by brand name
func FilterByBrand(paints []PaintProduct, brand string) []PaintProduct {
	var filtered []PaintProduct
	for _, p := range paints {
		if p.Brand == brand {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// FilterByOpacity returns paints filtered by opacity type
func FilterByOpacity(paints []PaintProduct, opacity string) []PaintProduct {
	var filtered []PaintProduct
	for _, p := range paints {
		if p.Opacity == opacity {
			filtered = append(filtered, p)
		}
	}
	return filtered
}
