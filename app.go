package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TrueBlocks/trueblocks-shopping/internal/color"
	"github.com/TrueBlocks/trueblocks-shopping/internal/inventory"
	"github.com/TrueBlocks/trueblocks-shopping/internal/settings"

	"github.com/jung-kurt/gofpdf"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct - the main controller for Wails bindings
type App struct {
	ctx            context.Context
	paintInventory []inventory.PaintProduct
	settings       *settings.Settings
	nColors        int
	lastImagePath  string
}

// PaletteResult represents a dominant color and its paint matches
type PaletteResult struct {
	DominantColor ColorInfo    `json:"dominantColor"`
	PaintMatches  []PaintMatch `json:"paintMatches"`
	MixingRecipe  []MixingPart `json:"mixingRecipe"`
}

// ColorInfo represents a color with its RGB and Hex values
type ColorInfo struct {
	R   uint8  `json:"r"`
	G   uint8  `json:"g"`
	B   uint8  `json:"b"`
	Hex string `json:"hex"`
}

// PaintMatch represents a matched paint with delta E score
type PaintMatch struct {
	Paint       inventory.PaintProduct `json:"paint"`
	DeltaE      float64                `json:"deltaE"`
	MatchRating string                 `json:"matchRating"`
}

// MixingPart represents a paint and its proportion in a mixing recipe
type MixingPart struct {
	Paint inventory.PaintProduct `json:"paint"`
	Parts int                    `json:"parts"`
}

// ProcessingResult is the complete result from image processing
type ProcessingResult struct {
	ImageData   string          `json:"imageData"`
	Palette     []PaletteResult `json:"palette"`
	ImageWidth  int             `json:"imageWidth"`
	ImageHeight int             `json:"imageHeight"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		nColors: 10,
	}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load settings
	var err error
	a.settings, err = settings.Load()
	if err != nil {
		runtime.LogError(ctx, fmt.Sprintf("Failed to load settings: %v", err))
		a.settings = &settings.Settings{
			WindowWidth:  1280,
			WindowHeight: 800,
			NColors:      10,
			TileSize:     1,
		}
	}

	// Initialize nColors from settings
	a.nColors = a.settings.NColors
	if a.nColors < 2 {
		a.nColors = 10
	}

	// Load paint inventory
	a.paintInventory, err = inventory.LoadPaints()
	if err != nil {
		runtime.LogError(ctx, fmt.Sprintf("Failed to load paint inventory: %v", err))
	} else {
		runtime.LogInfo(ctx, fmt.Sprintf("Loaded %d paints from inventory", len(a.paintInventory)))
	}
}

// domReady is called when the DOM is ready
func (a *App) domReady(ctx context.Context) {
	// Register file drop handler
	runtime.OnFileDrop(ctx, func(x, y int, paths []string) {
		if len(paths) > 0 {
			a.handleFileDrop(paths[0])
		}
	})

	runtime.LogInfo(ctx, "DOM ready, file drop handler registered")

	// Auto-load last image if available
	if lastImage := a.settings.GetLastImage(); lastImage != "" {
		runtime.LogInfo(ctx, fmt.Sprintf("Auto-loading last image: %s", lastImage))
		a.loadLastImage(lastImage)
	}
}

// loadLastImage loads a previously saved image on startup
func (a *App) loadLastImage(imagePath string) {
	// Emit processing started event
	runtime.EventsEmit(a.ctx, "processing:started", imagePath)

	// Process in a goroutine to avoid blocking
	go func() {
		result, err := a.ProcessImage(imagePath)
		if err != nil {
			runtime.EventsEmit(a.ctx, "processing:error", err.Error())
			return
		}
		runtime.EventsEmit(a.ctx, "processing:complete", result)
	}()
}

// handleFileDrop processes a dropped file
func (a *App) handleFileDrop(path string) {
	runtime.LogInfo(a.ctx, fmt.Sprintf("File dropped: %s", path))

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(path))
	validExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
	}

	if !validExtensions[ext] {
		runtime.EventsEmit(a.ctx, "processing:error", "Invalid file type. Please drop a JPG, PNG, or GIF image.")
		return
	}

	// Copy image to app data directory and add to recent list
	copiedPath, err := a.settings.AddRecentImage(path)
	if err != nil {
		runtime.LogWarning(a.ctx, fmt.Sprintf("Failed to copy image: %v", err))
		// Continue processing with original path
		copiedPath = path
	} else {
		runtime.LogInfo(a.ctx, fmt.Sprintf("Image copied to: %s", copiedPath))
		// Save settings after adding recent image
		if err := a.settings.Save(); err != nil {
			runtime.LogWarning(a.ctx, fmt.Sprintf("Failed to save settings: %v", err))
		}
	}

	// Emit processing started event
	runtime.EventsEmit(a.ctx, "processing:started", path)

	// Process in a goroutine to avoid blocking
	go func() {
		result, err := a.ProcessImage(path)
		if err != nil {
			runtime.EventsEmit(a.ctx, "processing:error", err.Error())
			return
		}
		runtime.EventsEmit(a.ctx, "processing:complete", result)
	}()
}

// ProcessImage processes an image and returns the palette with paint matches
func (a *App) ProcessImage(imagePath string) (*ProcessingResult, error) {
	a.lastImagePath = imagePath
	runtime.EventsEmit(a.ctx, "processing:progress", "Loading image...")

	// Open the image file
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	imageWidth := bounds.Dx()
	imageHeight := bounds.Dy()

	runtime.EventsEmit(a.ctx, "processing:progress", "Extracting dominant colors...")

	// Extract dominant colors using K-Means
	dominantColors := color.ExtractDominantColors(img, a.nColors)

	runtime.EventsEmit(a.ctx, "processing:progress", "Matching paints...")

	// Match each dominant color to paints
	var palette []PaletteResult
	for _, dc := range dominantColors {
		colorInfo := ColorInfo{
			R:   dc.R,
			G:   dc.G,
			B:   dc.B,
			Hex: fmt.Sprintf("#%02X%02X%02X", dc.R, dc.G, dc.B),
		}

		matches := a.findPaintMatches(dc.R, dc.G, dc.B, 3)
		mixingRecipe := a.calculateMixingRecipe(dc.R, dc.G, dc.B)

		palette = append(palette, PaletteResult{
			DominantColor: colorInfo,
			PaintMatches:  matches,
			MixingRecipe:  mixingRecipe,
		})
	}

	runtime.EventsEmit(a.ctx, "processing:progress", "Encoding image...")

	// Read the file again for base64 encoding
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image for encoding: %w", err)
	}

	// Determine MIME type
	mimeType := "image/jpeg"
	switch strings.ToLower(filepath.Ext(imagePath)) {
	case ".png":
		mimeType = "image/png"
	case ".gif":
		mimeType = "image/gif"
	}

	base64Image := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(imageData))

	return &ProcessingResult{
		ImageData:   base64Image,
		Palette:     palette,
		ImageWidth:  imageWidth,
		ImageHeight: imageHeight,
	}, nil
}

// GetPaintMatch finds the best paint matches for a given RGB color
func (a *App) GetPaintMatch(r, g, b uint8) []PaintMatch {
	return a.findPaintMatches(r, g, b, 3)
}

// findPaintMatches finds the top N paint matches for a given color
func (a *App) findPaintMatches(r, g, b uint8, topN int) []PaintMatch {
	if len(a.paintInventory) == 0 {
		return nil
	}

	// Convert target color to Lab
	targetLab := color.RGBToLab(r, g, b)

	type paintDistance struct {
		paint  inventory.PaintProduct
		deltaE float64
	}

	var distances []paintDistance

	for _, paint := range a.paintInventory {
		// Only include Golden brand paints
		if paint.Brand != "Golden" {
			continue
		}
		// Calculate Delta E using CIEDE2000
		deltaE := color.DeltaE2000(targetLab, paint.Lab)
		distances = append(distances, paintDistance{
			paint:  paint,
			deltaE: deltaE,
		})
	}

	// Sort by delta E
	sort.Slice(distances, func(i, j int) bool {
		return distances[i].deltaE < distances[j].deltaE
	})

	// Take top N
	var matches []PaintMatch
	for i := 0; i < topN && i < len(distances); i++ {
		match := PaintMatch{
			Paint:       distances[i].paint,
			DeltaE:      distances[i].deltaE,
			MatchRating: getMatchRating(distances[i].deltaE),
		}
		matches = append(matches, match)
	}

	return matches
}

// getMatchRating returns a human-readable rating based on Delta E
func getMatchRating(deltaE float64) string {
	switch {
	case deltaE < 1.0:
		return "Perfect Match"
	case deltaE < 2.0:
		return "Excellent Match"
	case deltaE < 5.0:
		return "Good Alternative"
	case deltaE < 10.0:
		return "Approximate"
	default:
		return "No Match"
	}
}

// GetPaintInventory returns the complete paint inventory
func (a *App) GetPaintInventory() []inventory.PaintProduct {
	return a.paintInventory
}

// PickColorFromCoordinates allows manual color picking from image coordinates
// This is called from the React canvas onClick
func (a *App) PickColorFromCoordinates(r, g, b uint8) []PaintMatch {
	return a.GetPaintMatch(r, g, b)
}

// GetMixingRecipe returns a mixing recipe for a given RGB color
func (a *App) GetMixingRecipe(r, g, b uint8) []MixingPart {
	return a.calculateMixingRecipe(r, g, b)
}

// calculateMixingRecipe finds a combination of 2-3 paints that can be mixed to approximate the target color
func (a *App) calculateMixingRecipe(r, g, b uint8) []MixingPart {
	if len(a.paintInventory) == 0 {
		return nil
	}

	// Get Golden paints only
	var goldenPaints []inventory.PaintProduct
	for _, p := range a.paintInventory {
		if p.Brand == "Golden" {
			goldenPaints = append(goldenPaints, p)
		}
	}

	if len(goldenPaints) < 2 {
		return nil
	}

	targetR, targetG, targetB := float64(r), float64(g), float64(b)
	targetLab := color.RGBToLab(r, g, b)

	type mixResult struct {
		paints []inventory.PaintProduct
		parts  []int
		deltaE float64
	}

	var bestMix mixResult
	bestMix.deltaE = 1000.0 // Start with a high value

	// Try all combinations of 2 paints with different ratios
	for i := 0; i < len(goldenPaints); i++ {
		for j := i + 1; j < len(goldenPaints); j++ {
			paint1 := goldenPaints[i]
			paint2 := goldenPaints[j]

			// Try ratios from 1:9 to 9:1
			for p1 := 1; p1 <= 9; p1++ {
				p2 := 10 - p1

				// Mix the colors (simple weighted average in RGB space)
				mixR := (float64(paint1.RGB[0])*float64(p1) + float64(paint2.RGB[0])*float64(p2)) / 10.0
				mixG := (float64(paint1.RGB[1])*float64(p1) + float64(paint2.RGB[1])*float64(p2)) / 10.0
				mixB := (float64(paint1.RGB[2])*float64(p1) + float64(paint2.RGB[2])*float64(p2)) / 10.0

				// Calculate distance to target
				mixLab := color.RGBToLab(uint8(mixR), uint8(mixG), uint8(mixB))
				deltaE := color.DeltaE2000(targetLab, mixLab)

				if deltaE < bestMix.deltaE {
					bestMix = mixResult{
						paints: []inventory.PaintProduct{paint1, paint2},
						parts:  []int{p1, p2},
						deltaE: deltaE,
					}
				}
			}
		}
	}

	// Also try 3-paint combinations for better accuracy (limited search)
	// Only try if 2-paint mix isn't good enough
	if bestMix.deltaE > 3.0 && len(goldenPaints) >= 3 {
		// Use the best single match as a base and find two others
		singleMatches := a.findPaintMatches(r, g, b, 5)
		if len(singleMatches) > 0 {
			basePaint := singleMatches[0].Paint

			for i := 0; i < len(goldenPaints); i++ {
				if goldenPaints[i].ID == basePaint.ID {
					continue
				}
				for j := i + 1; j < len(goldenPaints); j++ {
					if goldenPaints[j].ID == basePaint.ID {
						continue
					}

					paint2 := goldenPaints[i]
					paint3 := goldenPaints[j]

					// Try a few 3-way ratios
					for p1 := 4; p1 <= 8; p1++ {
						for p2 := 1; p2 <= 10-p1-1; p2++ {
							p3 := 10 - p1 - p2
							if p3 < 1 {
								continue
							}

							mixR := (float64(basePaint.RGB[0])*float64(p1) + float64(paint2.RGB[0])*float64(p2) + float64(paint3.RGB[0])*float64(p3)) / 10.0
							mixG := (float64(basePaint.RGB[1])*float64(p1) + float64(paint2.RGB[1])*float64(p2) + float64(paint3.RGB[1])*float64(p3)) / 10.0
							mixB := (float64(basePaint.RGB[2])*float64(p1) + float64(paint2.RGB[2])*float64(p2) + float64(paint3.RGB[2])*float64(p3)) / 10.0

							mixLab := color.RGBToLab(uint8(mixR), uint8(mixG), uint8(mixB))
							deltaE := color.DeltaE2000(targetLab, mixLab)

							if deltaE < bestMix.deltaE {
								bestMix = mixResult{
									paints: []inventory.PaintProduct{basePaint, paint2, paint3},
									parts:  []int{p1, p2, p3},
									deltaE: deltaE,
								}
							}
						}
					}
				}
			}
		}
	}

	// Check if the best single paint is better than any mix
	singleMatch := a.findPaintMatches(r, g, b, 1)
	if len(singleMatch) > 0 && singleMatch[0].DeltaE < bestMix.deltaE {
		// Single paint is better, return just that
		return []MixingPart{
			{Paint: singleMatch[0].Paint, Parts: 10},
		}
	}

	// Simplify parts if possible (e.g., 4:6 -> 2:3)
	simplifiedParts := simplifyRatio(bestMix.parts)

	// Build result
	var recipe []MixingPart
	for i, paint := range bestMix.paints {
		recipe = append(recipe, MixingPart{
			Paint: paint,
			Parts: simplifiedParts[i],
		})
	}

	// Avoid using local variable
	_ = targetR
	_ = targetG
	_ = targetB

	return recipe
}

// simplifyRatio reduces a ratio to its simplest form
func simplifyRatio(parts []int) []int {
	if len(parts) == 0 {
		return parts
	}

	// Find GCD of all parts
	gcd := parts[0]
	for i := 1; i < len(parts); i++ {
		gcd = gcdFunc(gcd, parts[i])
	}

	if gcd <= 1 {
		return parts
	}

	result := make([]int, len(parts))
	for i, p := range parts {
		result[i] = p / gcd
	}
	return result
}

// gcdFunc calculates the greatest common divisor
func gcdFunc(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// WindowSettings represents window position and size
type WindowSettings struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// GetWindowSettings returns the saved window position and size
func (a *App) GetWindowSettings() WindowSettings {
	if a.settings == nil {
		return WindowSettings{Width: 1280, Height: 800}
	}
	return WindowSettings{
		X:      a.settings.WindowX,
		Y:      a.settings.WindowY,
		Width:  a.settings.WindowWidth,
		Height: a.settings.WindowHeight,
	}
}

// SaveWindowSettings saves the window position and size
func (a *App) SaveWindowSettings(x, y, width, height int) error {
	if a.settings == nil {
		return nil
	}
	a.settings.UpdateWindowPosition(x, y)
	a.settings.UpdateWindowSize(width, height)
	return a.settings.Save()
}

// RecentImageInfo represents a recent image for the frontend
type RecentImageInfo struct {
	OriginalPath string `json:"originalPath"`
	CopiedPath   string `json:"copiedPath"`
	Filename     string `json:"filename"`
	ProcessedAt  string `json:"processedAt"`
}

// GetRecentImages returns the list of recently processed images
func (a *App) GetRecentImages() []RecentImageInfo {
	if a.settings == nil {
		return nil
	}
	var images []RecentImageInfo
	for _, img := range a.settings.GetRecentImages() {
		images = append(images, RecentImageInfo{
			OriginalPath: img.OriginalPath,
			CopiedPath:   img.CopiedPath,
			Filename:     img.Filename,
			ProcessedAt:  img.ProcessedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return images
}

// GetNColors returns the current number of colors to extract
func (a *App) GetNColors() int {
	return a.nColors
}

// SetNColors sets the number of colors to extract (clamped to 2-40) and saves to settings
func (a *App) SetNColors(n int) {
	if n < 2 {
		n = 2
	} else if n > 40 {
		n = 40
	}
	a.nColors = n
	if a.settings != nil {
		a.settings.NColors = n
		a.settings.Save()
	}
}

// GetTileSize returns the current tile size
func (a *App) GetTileSize() int {
	if a.settings != nil {
		return a.settings.TileSize
	}
	return 1
}

// SetTileSize sets the tile size (clamped to 1-16) and saves to settings
func (a *App) SetTileSize(n int) {
	if n < 1 {
		n = 1
	} else if n > 16 {
		n = 16
	}
	if a.settings != nil {
		a.settings.TileSize = n
		a.settings.Save()
	}
}

// GetPosterizeMode returns the current posterize mode
func (a *App) GetPosterizeMode() bool {
	if a.settings != nil {
		return a.settings.PosterizeMode
	}
	return false
}

// SetPosterizeMode sets the posterize mode and saves to settings
func (a *App) SetPosterizeMode(enabled bool) {
	if a.settings != nil {
		a.settings.PosterizeMode = enabled
		a.settings.Save()
	}
}

// GetSmoothingPasses returns the current smoothing passes
func (a *App) GetSmoothingPasses() int {
	if a.settings != nil {
		return a.settings.SmoothingPasses
	}
	return 0
}

// SetSmoothingPasses sets the smoothing passes (clamped to 0-10) and saves to settings
func (a *App) SetSmoothingPasses(n int) {
	if n < 0 {
		n = 0
	} else if n > 10 {
		n = 10
	}
	if a.settings != nil {
		a.settings.SmoothingPasses = n
		a.settings.Save()
	}
}

// GetAspectRatio returns the current aspect ratio setting
func (a *App) GetAspectRatio() string {
	if a.settings != nil && a.settings.AspectRatio != "" {
		return a.settings.AspectRatio
	}
	return "original"
}

// SetAspectRatio sets the aspect ratio and saves to settings
func (a *App) SetAspectRatio(ratio string) {
	validRatios := map[string]bool{"original": true, "landscape": true, "portrait": true, "square": true}
	if !validRatios[ratio] {
		ratio = "original"
	}
	if a.settings != nil {
		a.settings.AspectRatio = ratio
		a.settings.Save()
	}
}

// ReprocessImage reprocesses the last image with current settings
func (a *App) ReprocessImage() {
	if a.lastImagePath == "" {
		return
	}

	runtime.EventsEmit(a.ctx, "processing:started", a.lastImagePath)

	go func() {
		result, err := a.ProcessImage(a.lastImagePath)
		if err != nil {
			runtime.EventsEmit(a.ctx, "processing:error", err.Error())
			return
		}
		runtime.EventsEmit(a.ctx, "processing:complete", result)
	}()
}

// PDFPaintPart represents a paint in the mixing recipe for PDF export
type PDFPaintPart struct {
	Name     string `json:"name"`
	Brand    string `json:"brand"`
	Series   int    `json:"series"`
	Pigments string `json:"pigments"`
	Opacity  string `json:"opacity"`
	Hex      string `json:"hex"`
	RGB      []int  `json:"rgb"`
	Parts    int    `json:"parts"`
}

// PDFExportData contains all data needed to generate the PDF
type PDFExportData struct {
	ColorIndex        int            `json:"colorIndex"`        // 1-based color index
	ImageData         string         `json:"imageData"`         // Base64 PNG of the modified canvas
	OriginalImageData string         `json:"originalImageData"` // Base64 PNG of the original image
	TargetHex         string         `json:"targetHex"`         // Target color hex
	TargetRGB         []int          `json:"targetRGB"`         // Target color RGB
	ResultHex         string         `json:"resultHex"`         // Mixed result hex
	MixingRecipe      []PDFPaintPart `json:"mixingRecipe"`      // Paints to buy
	TotalParts        int            `json:"totalParts"`        // Total parts in recipe
}

// ExportColorDetailPDF generates a PDF with the color detail and opens it
func (a *App) ExportColorDetailPDF(data PDFExportData) error {
	// Create a temporary file that will be cleaned up when the app exits
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("acrylic-color-%d-*.pdf", data.ColorIndex))
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpFile.Close()
	savePath := tmpFile.Name()

	// Create PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	// Title
	pdf.SetFont("Arial", "B", 16)
	pdf.SetTextColor(0, 0, 0)
	pdf.Cell(0, 8, fmt.Sprintf("Color %d Detail - AcrylicMaster", data.ColorIndex))
	pdf.Ln(10)

	// Decode and add images side by side (modified on left, original on right)
	var pdfImgHeight float64 = 0
	if data.ImageData != "" {
		imgData, _ := strings.CutPrefix(data.ImageData, "data:image/png;base64,")
		decoded, err := base64.StdEncoding.DecodeString(imgData)
		if err == nil {
			img, err := png.Decode(bytes.NewReader(decoded))
			if err == nil {
				// Save temp image for modified
				tmpFile, err := os.CreateTemp("", "acrylic-modified-*.png")
				if err == nil {
					png.Encode(tmpFile, img)
					tmpFile.Close()
					defer os.Remove(tmpFile.Name())

					// Calculate image dimensions - maximize size while fitting side by side
					// A4 page is 210mm wide, with 15mm margins = 180mm usable
					// For two images side by side with 5mm gap: (180 - 5) / 2 = 87.5mm each
					bounds := img.Bounds()
					imgWidth := float64(bounds.Dx())
					imgHeight := float64(bounds.Dy())

					maxWidth := 87.0   // Each image max width
					maxHeight := 100.0 // Max height to leave room for text below

					// Calculate scale to fit within constraints
					scaleW := maxWidth / (imgWidth * 0.264583)
					scaleH := maxHeight / (imgHeight * 0.264583)
					scale := scaleW
					if scaleH < scaleW {
						scale = scaleH
					}

					pdfImgWidth := imgWidth * 0.264583 * scale
					pdfImgHeight = imgHeight * 0.264583 * scale

					startY := pdf.GetY()

					// Modified image on left with label above
					pdf.SetFont("Arial", "B", 9)
					pdf.SetXY(15, startY)
					pdf.Cell(pdfImgWidth, 5, "Modified")
					pdf.ImageOptions(tmpFile.Name(), 15, startY+6, pdfImgWidth, pdfImgHeight, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")

					// Original image on right (if available)
					if data.OriginalImageData != "" {
						origData, _ := strings.CutPrefix(data.OriginalImageData, "data:image/png;base64,")
						origDecoded, err := base64.StdEncoding.DecodeString(origData)
						if err == nil {
							origImg, err := png.Decode(bytes.NewReader(origDecoded))
							if err == nil {
								tmpOrigFile, err := os.CreateTemp("", "acrylic-original-*.png")
								if err == nil {
									png.Encode(tmpOrigFile, origImg)
									tmpOrigFile.Close()
									defer os.Remove(tmpOrigFile.Name())

									// Label and image on right side (proper gap)
									rightX := 15 + pdfImgWidth + 6
									pdf.SetFont("Arial", "B", 9)
									pdf.SetXY(rightX, startY)
									pdf.Cell(pdfImgWidth, 5, "Original")
									pdf.ImageOptions(tmpOrigFile.Name(), rightX, startY+6, pdfImgWidth, pdfImgHeight, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
								}
							}
						}
					}

					pdf.SetY(startY + 6 + pdfImgHeight + 5)
				}
			}
		}
	}

	// Target vs Result colors (compact inline layout)
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(0, 6, "Color Comparison")
	pdf.Ln(7)

	y := pdf.GetY()

	// Target color swatch (smaller)
	if len(data.TargetRGB) >= 3 {
		pdf.SetFillColor(data.TargetRGB[0], data.TargetRGB[1], data.TargetRGB[2])
		pdf.Rect(15, y, 15, 12, "F")
		pdf.SetXY(32, y)
		pdf.SetFont("Arial", "B", 9)
		pdf.Cell(20, 6, "Target")
		pdf.SetFont("Arial", "", 9)
		pdf.SetXY(32, y+5)
		pdf.Cell(35, 6, fmt.Sprintf("%s RGB(%d,%d,%d)", data.TargetHex, data.TargetRGB[0], data.TargetRGB[1], data.TargetRGB[2]))
	}

	// Result color swatch (smaller)
	resultRGB := hexToRGB(data.ResultHex)
	pdf.SetFillColor(resultRGB[0], resultRGB[1], resultRGB[2])
	pdf.Rect(100, y, 15, 12, "F")
	pdf.SetXY(117, y)
	pdf.SetFont("Arial", "B", 9)
	pdf.Cell(20, 6, "Result")
	pdf.SetFont("Arial", "", 9)
	pdf.SetXY(117, y+5)
	pdf.Cell(30, 6, data.ResultHex)

	pdf.Ln(16)

	// Mixing Recipe Bar
	if len(data.MixingRecipe) > 0 {
		pdf.SetFont("Arial", "B", 11)
		pdf.Cell(0, 6, "Mixing Recipe")
		pdf.Ln(8)

		barY := pdf.GetY()
		barHeight := 12.0
		barWidth := 180.0
		x := 15.0

		for _, part := range data.MixingRecipe {
			partWidth := barWidth * float64(part.Parts) / float64(data.TotalParts)
			rgb := hexToRGB(part.Hex)
			pdf.SetFillColor(rgb[0], rgb[1], rgb[2])
			pdf.Rect(x, barY, partWidth, barHeight, "F")

			// Text on bar
			luminance := (0.299*float64(rgb[0]) + 0.587*float64(rgb[1]) + 0.114*float64(rgb[2])) / 255
			if luminance > 0.5 {
				pdf.SetTextColor(0, 0, 0)
			} else {
				pdf.SetTextColor(255, 255, 255)
			}
			pdf.SetFont("Arial", "", 7)
			if partWidth > 15 {
				pdf.SetXY(x+1, barY+2)
				pdf.CellFormat(partWidth-2, 4, part.Name, "", 0, "C", false, 0, "")
				pdf.SetXY(x+1, barY+6)
				pdf.CellFormat(partWidth-2, 4, fmt.Sprintf("%d pt", part.Parts), "", 0, "C", false, 0, "")
			}
			x += partWidth
		}

		pdf.SetTextColor(0, 0, 0)
		pdf.Ln(barHeight + 6)

		// Paint Details (compact - all on one line per paint)
		pdf.SetFont("Arial", "B", 11)
		pdf.Cell(0, 6, "Paints to Buy")
		pdf.Ln(7)

		for _, part := range data.MixingRecipe {
			y := pdf.GetY()

			// Color swatch (smaller)
			rgb := hexToRGB(part.Hex)
			pdf.SetFillColor(rgb[0], rgb[1], rgb[2])
			pdf.Rect(15, y, 12, 10, "F")

			// Paint info (compact single line)
			pdf.SetXY(29, y)
			pdf.SetFont("Arial", "B", 9)
			pdf.Cell(45, 5, part.Name)
			pdf.SetFont("Arial", "", 8)
			pdf.Cell(25, 5, part.Brand)
			pdf.Cell(15, 5, fmt.Sprintf("S%d", part.Series))
			pdf.Cell(30, 5, part.Pigments)
			pdf.Cell(20, 5, part.Opacity)
			pdf.Cell(20, 5, part.Hex)
			pdf.Cell(20, 5, fmt.Sprintf("%d/%d parts", part.Parts, data.TotalParts))
			pdf.Ln(11)
		}
	}

	// Save PDF
	if err := pdf.OutputFileAndClose(savePath); err != nil {
		return fmt.Errorf("failed to save PDF: %w", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("Saved PDF to: %s", savePath))

	// Open the PDF with the default application
	exec.Command("open", savePath).Start()

	return nil
}

// ShoppingListPaint represents a paint for the shopping list
type ShoppingListPaint struct {
	Name     string `json:"name"`
	Brand    string `json:"brand"`
	Series   int    `json:"series"`
	Pigments string `json:"pigments"`
	Opacity  string `json:"opacity"`
	Hex      string `json:"hex"`
}

// PaletteColorInfo represents a color in the palette for PDF export
type PaletteColorInfo struct {
	Hex                string             `json:"hex"`
	ColorNumber        int                `json:"colorNumber"`
	MixingRecipe       []MixingRecipeItem `json:"mixingRecipe"`
	IsolationImageData string             `json:"isolationImageData"` // Base64 PNG of isolated color pixels
}

// MixingRecipeItem represents a paint in a mixing recipe
type MixingRecipeItem struct {
	Name  string `json:"name"`
	Parts int    `json:"parts"`
	Hex   string `json:"hex"`
}

// ComparisonPDFData contains data for the comparison view PDF
type ComparisonPDFData struct {
	ModifiedImageData string              `json:"modifiedImageData"` // Base64 PNG of modified image
	OriginalImageData string              `json:"originalImageData"` // Base64 PNG of original image
	ShoppingList      []ShoppingListPaint `json:"shoppingList"`      // Unique paints to buy
	Palette           []PaletteColorInfo  `json:"palette"`           // Color palette with mixing recipes
}

// ExportComparisonPDF generates a PDF with side-by-side images and shopping list
func (a *App) ExportComparisonPDF(data ComparisonPDFData) error {
	// Create a temporary file that will be cleaned up when the app exits
	tmpFile, err := os.CreateTemp("", "acrylic-comparison-*.pdf")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpFile.Close()
	savePath := tmpFile.Name()

	// Create PDF (landscape for better image display)
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)

	// ========== PAGE 1: Images ==========
	pdf.AddPage()

	// Title
	pdf.SetFont("Arial", "B", 16)
	pdf.SetTextColor(0, 0, 0)
	pdf.Cell(0, 8, "AcrylicMaster - Image Comparison")
	pdf.Ln(12)

	// Decode and add images side by side
	var pdfImgHeight float64 = 0
	startY := pdf.GetY()

	// A4 landscape: 297mm wide, with 10mm margins = 277mm usable
	// Two images with 6mm gap: (277 - 6) / 2 = 135.5mm each
	maxWidth := 135.0
	maxHeight := 140.0 // Larger now that shopping list is on separate page

	// Modified image on left
	if data.ModifiedImageData != "" {
		imgData, _ := strings.CutPrefix(data.ModifiedImageData, "data:image/png;base64,")
		decoded, err := base64.StdEncoding.DecodeString(imgData)
		if err == nil {
			img, err := png.Decode(bytes.NewReader(decoded))
			if err == nil {
				tmpFile, err := os.CreateTemp("", "comparison-modified-*.png")
				if err == nil {
					png.Encode(tmpFile, img)
					tmpFile.Close()
					defer os.Remove(tmpFile.Name())

					bounds := img.Bounds()
					imgWidth := float64(bounds.Dx())
					imgHeight := float64(bounds.Dy())

					scaleW := maxWidth / (imgWidth * 0.264583)
					scaleH := maxHeight / (imgHeight * 0.264583)
					scale := scaleW
					if scaleH < scaleW {
						scale = scaleH
					}

					pdfImgWidth := imgWidth * 0.264583 * scale
					pdfImgHeight = imgHeight * 0.264583 * scale

					// Label
					pdf.SetFont("Arial", "B", 10)
					pdf.SetXY(10, startY)
					pdf.Cell(pdfImgWidth, 5, "Modified")
					pdf.ImageOptions(tmpFile.Name(), 10, startY+6, pdfImgWidth, pdfImgHeight, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")

					// Original image on right
					if data.OriginalImageData != "" {
						origData, _ := strings.CutPrefix(data.OriginalImageData, "data:image/png;base64,")
						origDecoded, err := base64.StdEncoding.DecodeString(origData)
						if err == nil {
							origImg, err := png.Decode(bytes.NewReader(origDecoded))
							if err == nil {
								tmpOrigFile, err := os.CreateTemp("", "comparison-original-*.png")
								if err == nil {
									png.Encode(tmpOrigFile, origImg)
									tmpOrigFile.Close()
									defer os.Remove(tmpOrigFile.Name())

									rightX := 10 + pdfImgWidth + 6
									pdf.SetFont("Arial", "B", 10)
									pdf.SetXY(rightX, startY)
									pdf.Cell(pdfImgWidth, 5, "Original")
									pdf.ImageOptions(tmpOrigFile.Name(), rightX, startY+6, pdfImgWidth, pdfImgHeight, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
								}
							}
						}
					}
				}
			}
		}
	}

	// ========== PAGE 2: Shopping List ==========
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 8, "Shopping List")
	pdf.Ln(12)

	// Table header
	pdf.SetFillColor(240, 240, 240)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(15, 8, "", "1", 0, "C", true, 0, "") // Color swatch column
	pdf.CellFormat(80, 8, "Paint Name", "1", 0, "L", true, 0, "")
	pdf.CellFormat(40, 8, "Brand", "1", 0, "L", true, 0, "")
	pdf.CellFormat(20, 8, "Series", "1", 0, "C", true, 0, "")
	pdf.CellFormat(60, 8, "Pigments", "1", 0, "L", true, 0, "")
	pdf.CellFormat(40, 8, "Opacity", "1", 0, "L", true, 0, "")
	pdf.Ln(-1)

	// Table rows
	pdf.SetFont("Arial", "", 10)
	for _, paint := range data.ShoppingList {
		y := pdf.GetY()

		// Color swatch
		rgb := hexToRGB(paint.Hex)
		pdf.SetFillColor(rgb[0], rgb[1], rgb[2])
		pdf.Rect(pdf.GetX()+2, y+1.5, 11, 5, "F")
		pdf.CellFormat(15, 8, "", "1", 0, "C", false, 0, "")

		pdf.SetFillColor(255, 255, 255)
		pdf.CellFormat(80, 8, paint.Name, "1", 0, "L", false, 0, "")
		pdf.CellFormat(40, 8, paint.Brand, "1", 0, "L", false, 0, "")
		pdf.CellFormat(20, 8, fmt.Sprintf("%d", paint.Series), "1", 0, "C", false, 0, "")
		pdf.CellFormat(60, 8, paint.Pigments, "1", 0, "L", false, 0, "")
		pdf.CellFormat(40, 8, paint.Opacity, "1", 0, "L", false, 0, "")
		pdf.Ln(-1)
	}

	// ========== PAGE 3+: Color Mixing Details (one page per color) ==========
	if len(data.Palette) > 0 {
		// Decode original image once for reuse on all color pages
		var origImg image.Image
		var origTmpFile *os.File
		if data.OriginalImageData != "" {
			origData, _ := strings.CutPrefix(data.OriginalImageData, "data:image/png;base64,")
			origDecoded, err := base64.StdEncoding.DecodeString(origData)
			if err == nil {
				origImg, _ = png.Decode(bytes.NewReader(origDecoded))
				if origImg != nil {
					origTmpFile, _ = os.CreateTemp("", "original-ref-*.png")
					if origTmpFile != nil {
						png.Encode(origTmpFile, origImg)
						origTmpFile.Close()
						defer os.Remove(origTmpFile.Name())
					}
				}
			}
		}

		for _, color := range data.Palette {
			pdf.AddPage()

			// Page title
			pdf.SetFont("Arial", "B", 14)
			pdf.SetTextColor(0, 0, 0)
			pdf.Cell(0, 8, fmt.Sprintf("Color #%d - %s", color.ColorNumber, color.Hex))
			pdf.Ln(12)

			// Layout for side-by-side images
			// A4 landscape: 297mm wide, with 10mm margins = 277mm usable
			// Two images with 6mm gap: (277 - 6) / 2 = 135.5mm each max
			maxImgWidth := 130.0
			maxImgHeight := 100.0
			imgGap := 6.0
			leftMargin := 10.0

			startY := pdf.GetY()
			var pdfImgWidth, pdfImgHeight float64

			// Isolation image on left ("This Color" pixels only)
			if color.IsolationImageData != "" {
				imgData, _ := strings.CutPrefix(color.IsolationImageData, "data:image/png;base64,")
				decoded, err := base64.StdEncoding.DecodeString(imgData)
				if err == nil {
					img, err := png.Decode(bytes.NewReader(decoded))
					if err == nil {
						tmpFile, err := os.CreateTemp("", "isolation-page-*.png")
						if err == nil {
							png.Encode(tmpFile, img)
							tmpFile.Close()
							defer os.Remove(tmpFile.Name())

							// Calculate aspect-ratio-preserving dimensions
							bounds := img.Bounds()
							imgW := float64(bounds.Dx())
							imgH := float64(bounds.Dy())
							scaleW := maxImgWidth / (imgW * 0.264583)
							scaleH := maxImgHeight / (imgH * 0.264583)
							scale := scaleW
							if scaleH < scaleW {
								scale = scaleH
							}
							pdfImgWidth = imgW * 0.264583 * scale
							pdfImgHeight = imgH * 0.264583 * scale

							// Label above image
							pdf.SetFont("Arial", "B", 10)
							pdf.SetXY(leftMargin, startY)
							pdf.Cell(pdfImgWidth, 5, "This Color")
							pdf.ImageOptions(tmpFile.Name(), leftMargin, startY+6, pdfImgWidth, pdfImgHeight, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
						}
					}
				}
			}

			// Original image on right
			if origTmpFile != nil && pdfImgWidth > 0 {
				rightX := leftMargin + pdfImgWidth + imgGap
				pdf.SetFont("Arial", "B", 10)
				pdf.SetXY(rightX, startY)
				pdf.Cell(pdfImgWidth, 5, "Original")
				pdf.ImageOptions(origTmpFile.Name(), rightX, startY+6, pdfImgWidth, pdfImgHeight, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
			}

			// Move below images
			pdf.SetY(startY + 6 + pdfImgHeight + 10)

			// Color swatch and mixing bar section
			swatchSize := 25.0
			barHeight := 16.0
			barWidth := 240.0
			swatchY := pdf.GetY()

			// Color swatch
			rgb := hexToRGB(color.Hex)
			pdf.SetFillColor(rgb[0], rgb[1], rgb[2])
			pdf.Rect(leftMargin, swatchY, swatchSize, swatchSize, "F")
			pdf.SetDrawColor(0, 0, 0)
			pdf.Rect(leftMargin, swatchY, swatchSize, swatchSize, "D")

			// Color info next to swatch
			pdf.SetFont("Arial", "B", 12)
			pdf.SetTextColor(0, 0, 0)
			pdf.SetXY(leftMargin+swatchSize+5, swatchY)
			pdf.Cell(50, 6, fmt.Sprintf("Color #%d", color.ColorNumber))
			pdf.SetFont("Arial", "", 10)
			pdf.SetXY(leftMargin+swatchSize+5, swatchY+7)
			pdf.Cell(50, 6, color.Hex)

			// Calculate total parts for the proportion bar
			totalParts := 0
			for _, part := range color.MixingRecipe {
				totalParts += part.Parts
			}

			// Mixing recipe bar
			barX := leftMargin + swatchSize + 5
			barY := swatchY + 16
			currentX := barX

			if totalParts > 0 {
				pdf.SetFont("Arial", "B", 10)
				pdf.SetXY(barX, barY-6)
				pdf.Cell(100, 5, "Mixing Recipe")
				barY += 2

				for _, part := range color.MixingRecipe {
					partWidth := (float64(part.Parts) / float64(totalParts)) * barWidth
					partRgb := hexToRGB(part.Hex)
					pdf.SetFillColor(partRgb[0], partRgb[1], partRgb[2])
					pdf.Rect(currentX, barY, partWidth, barHeight, "F")

					// Text on bar with contrasting color
					luminance := (0.299*float64(partRgb[0]) + 0.587*float64(partRgb[1]) + 0.114*float64(partRgb[2])) / 255
					if luminance > 0.5 {
						pdf.SetTextColor(0, 0, 0)
					} else {
						pdf.SetTextColor(255, 255, 255)
					}
					pdf.SetFont("Arial", "", 8)
					if partWidth > 25 {
						// Name on top line, parts on bottom
						name := part.Name
						maxChars := int(partWidth / 2.2)
						if len(name) > maxChars && maxChars > 3 {
							name = name[:maxChars-3] + "..."
						}
						pdf.SetXY(currentX+1, barY+2)
						pdf.CellFormat(partWidth-2, 6, name, "", 0, "C", false, 0, "")
						pdf.SetXY(currentX+1, barY+8)
						pdf.CellFormat(partWidth-2, 6, fmt.Sprintf("%d parts", part.Parts), "", 0, "C", false, 0, "")
					} else if partWidth > 12 {
						// Just show parts
						pdf.SetXY(currentX+1, barY+5)
						pdf.CellFormat(partWidth-2, 6, fmt.Sprintf("%d", part.Parts), "", 0, "C", false, 0, "")
					}
					currentX += partWidth
				}
				// Border around the whole bar
				pdf.SetDrawColor(100, 100, 100)
				pdf.Rect(barX, barY, barWidth, barHeight, "D")
			}

			pdf.SetTextColor(0, 0, 0) // Reset text color
		}
	}

	// Save PDF
	if err := pdf.OutputFileAndClose(savePath); err != nil {
		return fmt.Errorf("failed to save PDF: %w", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("Saved comparison PDF to: %s", savePath))

	// Open the PDF with the default application
	exec.Command("open", savePath).Start()

	return nil
}

// PbnPaletteItem represents a color in the palette for Paint-by-Numbers PDF
type PbnPaletteItem struct {
	Hex                 string             `json:"hex"`
	ColorNumber         int                `json:"colorNumber"`
	HighlightedPbnImage string             `json:"highlightedPbnImage"` // Paint-by-numbers with this color highlighted
	MixingRecipe        []MixingRecipeItem `json:"mixingRecipe"`
}

// PaintByNumbersPDFData contains data for the Paint-by-Numbers PDF
type PaintByNumbersPDFData struct {
	PaintByNumbersImageData string              `json:"paintByNumbersImageData"` // Base64 PNG of paint-by-numbers outline
	FullPagePbnImageData    string              `json:"fullPagePbnImageData"`    // Base64 PNG of minimal full-page paint-by-numbers
	OriginalImageData       string              `json:"originalImageData"`       // Base64 PNG of original image
	ShoppingList            []ShoppingListPaint `json:"shoppingList"`            // Unique paints to buy
	Palette                 []PbnPaletteItem    `json:"palette"`                 // Color palette with mixing recipes
}

// ExportPaintByNumbersPDF generates a Paint-by-Numbers PDF
func (a *App) ExportPaintByNumbersPDF(data PaintByNumbersPDFData) error {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "acrylic-pbn-*.pdf")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpFile.Close()
	savePath := tmpFile.Name()

	// Create PDF (landscape for better image display)
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)

	// Decode images once for reuse
	var pbnTmpFile, origTmpFile *os.File
	var pbnImgWidth, pbnImgHeight float64

	// Decode paint-by-numbers image
	if data.PaintByNumbersImageData != "" {
		imgData, _ := strings.CutPrefix(data.PaintByNumbersImageData, "data:image/png;base64,")
		decoded, err := base64.StdEncoding.DecodeString(imgData)
		if err == nil {
			img, err := png.Decode(bytes.NewReader(decoded))
			if err == nil {
				pbnTmpFile, err = os.CreateTemp("", "pbn-outline-*.png")
				if err == nil {
					png.Encode(pbnTmpFile, img)
					pbnTmpFile.Close()
					defer os.Remove(pbnTmpFile.Name())

					bounds := img.Bounds()
					pbnImgWidth = float64(bounds.Dx())
					pbnImgHeight = float64(bounds.Dy())
				}
			}
		}
	}

	// Decode original image
	if data.OriginalImageData != "" {
		imgData, _ := strings.CutPrefix(data.OriginalImageData, "data:image/png;base64,")
		decoded, err := base64.StdEncoding.DecodeString(imgData)
		if err == nil {
			img, err := png.Decode(bytes.NewReader(decoded))
			if err == nil {
				origTmpFile, err = os.CreateTemp("", "pbn-original-*.png")
				if err == nil {
					png.Encode(origTmpFile, img)
					origTmpFile.Close()
					defer os.Remove(origTmpFile.Name())
				}
			}
		}
	}

	// ========== PAGE 1: Overview ==========
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.SetTextColor(0, 0, 0)
	pdf.Cell(0, 8, "AcrylicMaster - Paint by Numbers")
	pdf.Ln(12)

	startY := pdf.GetY()
	leftMargin := 10.0

	// A4 landscape: 297mm wide, with 10mm margins = 277mm usable
	maxWidth := 135.0
	maxHeight := 140.0

	// Calculate scaled dimensions
	var pdfImgWidth, pdfImgHeight float64
	if pbnImgWidth > 0 && pbnImgHeight > 0 {
		scale := min(maxWidth/pbnImgWidth, maxHeight/pbnImgHeight)
		pdfImgWidth = pbnImgWidth * scale
		pdfImgHeight = pbnImgHeight * scale
	}

	imgGap := 6.0

	// Paint-by-Numbers image on left
	if pbnTmpFile != nil && pdfImgWidth > 0 {
		pdf.SetFont("Arial", "B", 10)
		pdf.SetXY(leftMargin, startY)
		pdf.Cell(pdfImgWidth, 5, "Paint by Numbers")
		pdf.ImageOptions(pbnTmpFile.Name(), leftMargin, startY+6, pdfImgWidth, pdfImgHeight, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	}

	// Original image on right
	if origTmpFile != nil && pdfImgWidth > 0 {
		rightX := leftMargin + pdfImgWidth + imgGap
		pdf.SetFont("Arial", "B", 10)
		pdf.SetXY(rightX, startY)
		pdf.Cell(pdfImgWidth, 5, "Original")
		pdf.ImageOptions(origTmpFile.Name(), rightX, startY+6, pdfImgWidth, pdfImgHeight, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	}

	// ========== PAGE 2: Full-Page Paint-by-Numbers ==========
	if data.FullPagePbnImageData != "" {
		pdf.AddPage()

		// Decode full-page image
		imgData, _ := strings.CutPrefix(data.FullPagePbnImageData, "data:image/png;base64,")
		decoded, err := base64.StdEncoding.DecodeString(imgData)
		if err == nil {
			img, err := png.Decode(bytes.NewReader(decoded))
			if err == nil {
				fullTmpFile, err := os.CreateTemp("", "pbn-full-*.png")
				if err == nil {
					png.Encode(fullTmpFile, img)
					fullTmpFile.Close()
					defer os.Remove(fullTmpFile.Name())

					// A4 landscape: 297mm x 210mm, with margins = 277mm x 190mm usable (no title)
					pageWidth := 277.0
					pageHeight := 190.0
					imgStartY := 10.0

					bounds := img.Bounds()
					imgW := float64(bounds.Dx())
					imgH := float64(bounds.Dy())
					scale := min(pageWidth/imgW, pageHeight/imgH)
					scaledW := imgW * scale
					scaledH := imgH * scale

					// Center the image horizontally
					imgX := leftMargin + (pageWidth-scaledW)/2

					pdf.ImageOptions(fullTmpFile.Name(), imgX, imgStartY, scaledW, scaledH, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
				}
			}
		}
	}

	// ========== PAGE 3: Shopping List ==========
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.SetTextColor(0, 0, 0)
	pdf.Cell(0, 8, "Shopping List")
	pdf.Ln(12)

	// Table header
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(240, 240, 240)
	colWidths := []float64{15, 80, 50, 20, 60, 30}
	headers := []string{"", "Paint", "Brand", "Series", "Pigments", "Opacity"}
	for i, header := range headers {
		pdf.CellFormat(colWidths[i], 8, header, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// Table rows
	pdf.SetFont("Arial", "", 9)
	for _, paint := range data.ShoppingList {
		// Color swatch
		rgb := hexToRGB(paint.Hex)
		pdf.SetFillColor(rgb[0], rgb[1], rgb[2])
		cellY := pdf.GetY()
		pdf.Rect(leftMargin+2, cellY+1, 10, 6, "F")
		pdf.SetFillColor(255, 255, 255)
		pdf.CellFormat(colWidths[0], 8, "", "1", 0, "C", false, 0, "")

		pdf.CellFormat(colWidths[1], 8, paint.Name, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[2], 8, paint.Brand, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[3], 8, fmt.Sprintf("%d", paint.Series), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[4], 8, paint.Pigments, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[5], 8, paint.Opacity, "1", 0, "C", false, 0, "")
		pdf.Ln(-1)
	}

	// ========== PAGES 4-N: One page per color ==========
	for _, color := range data.Palette {
		pdf.AddPage()

		// Title with color number and swatch
		pdf.SetFont("Arial", "B", 14)
		pdf.SetTextColor(0, 0, 0)
		pdf.Cell(0, 8, fmt.Sprintf("Color #%d", color.ColorNumber))
		pdf.Ln(10)

		// Color swatch
		swatchSize := 20.0
		swatchY := pdf.GetY()
		rgb := hexToRGB(color.Hex)
		pdf.SetFillColor(rgb[0], rgb[1], rgb[2])
		pdf.Rect(leftMargin, swatchY, swatchSize, swatchSize, "F")
		pdf.SetDrawColor(0, 0, 0)
		pdf.Rect(leftMargin, swatchY, swatchSize, swatchSize, "D")

		// Hex code next to swatch
		pdf.SetFont("Arial", "", 12)
		pdf.SetXY(leftMargin+swatchSize+5, swatchY+5)
		pdf.Cell(50, 10, color.Hex)

		pdf.SetY(swatchY + swatchSize + 10)

		// Images section
		imageStartY := pdf.GetY()
		imgMaxWidth := 120.0
		imgMaxHeight := 90.0

		// Calculate scaled dimensions for this section
		var sectionImgWidth, sectionImgHeight float64
		if pbnImgWidth > 0 && pbnImgHeight > 0 {
			scale := min(imgMaxWidth/pbnImgWidth, imgMaxHeight/pbnImgHeight)
			sectionImgWidth = pbnImgWidth * scale
			sectionImgHeight = pbnImgHeight * scale
		}

		// Decode and draw the highlighted paint-by-numbers image for this color
		var highlightedTmpFile *os.File
		if color.HighlightedPbnImage != "" && sectionImgWidth > 0 {
			imgData, _ := strings.CutPrefix(color.HighlightedPbnImage, "data:image/png;base64,")
			decoded, err := base64.StdEncoding.DecodeString(imgData)
			if err == nil {
				img, err := png.Decode(bytes.NewReader(decoded))
				if err == nil {
					highlightedTmpFile, err = os.CreateTemp("", fmt.Sprintf("pbn-highlight-%d-*.png", color.ColorNumber))
					if err == nil {
						png.Encode(highlightedTmpFile, img)
						highlightedTmpFile.Close()
						defer os.Remove(highlightedTmpFile.Name())

						pdf.SetFont("Arial", "B", 10)
						pdf.SetXY(leftMargin, imageStartY)
						pdf.Cell(sectionImgWidth, 5, "Paint by Numbers")
						pdf.ImageOptions(highlightedTmpFile.Name(), leftMargin, imageStartY+6, sectionImgWidth, sectionImgHeight, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
					}
				}
			}
		} else if pbnTmpFile != nil && sectionImgWidth > 0 {
			// Fallback to generic paint-by-numbers image
			pdf.SetFont("Arial", "B", 10)
			pdf.SetXY(leftMargin, imageStartY)
			pdf.Cell(sectionImgWidth, 5, "Paint by Numbers")
			pdf.ImageOptions(pbnTmpFile.Name(), leftMargin, imageStartY+6, sectionImgWidth, sectionImgHeight, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
		}

		// Original image on right
		if origTmpFile != nil && sectionImgWidth > 0 {
			rightX := leftMargin + sectionImgWidth + imgGap
			pdf.SetFont("Arial", "B", 10)
			pdf.SetXY(rightX, imageStartY)
			pdf.Cell(sectionImgWidth, 5, "Original")
			pdf.ImageOptions(origTmpFile.Name(), rightX, imageStartY+6, sectionImgWidth, sectionImgHeight, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
		}

		// Move below images
		pdf.SetY(imageStartY + 6 + sectionImgHeight + 10)

		// Mixing Recipe section
		if len(color.MixingRecipe) > 0 {
			pdf.SetFont("Arial", "B", 12)
			pdf.Cell(0, 8, "Mixing Recipe")
			pdf.Ln(10)

			// Calculate total parts
			totalParts := 0
			for _, part := range color.MixingRecipe {
				totalParts += part.Parts
			}

			// Draw mixing bar
			barWidth := 240.0
			barHeight := 20.0
			barX := leftMargin
			barY := pdf.GetY()
			currentX := barX

			for _, part := range color.MixingRecipe {
				partWidth := (float64(part.Parts) / float64(totalParts)) * barWidth
				partRgb := hexToRGB(part.Hex)
				pdf.SetFillColor(partRgb[0], partRgb[1], partRgb[2])
				pdf.Rect(currentX, barY, partWidth, barHeight, "F")

				// Text with contrasting color
				luminance := (0.299*float64(partRgb[0]) + 0.587*float64(partRgb[1]) + 0.114*float64(partRgb[2])) / 255
				if luminance > 0.5 {
					pdf.SetTextColor(0, 0, 0)
				} else {
					pdf.SetTextColor(255, 255, 255)
				}
				pdf.SetFont("Arial", "", 9)
				if partWidth > 30 {
					name := part.Name
					maxChars := int(partWidth / 2.5)
					if len(name) > maxChars && maxChars > 3 {
						name = name[:maxChars-3] + "..."
					}
					pdf.SetXY(currentX+2, barY+3)
					pdf.CellFormat(partWidth-4, 6, name, "", 0, "C", false, 0, "")
					pdf.SetXY(currentX+2, barY+10)
					pdf.CellFormat(partWidth-4, 6, fmt.Sprintf("%d parts", part.Parts), "", 0, "C", false, 0, "")
				} else if partWidth > 15 {
					pdf.SetXY(currentX+1, barY+6)
					pdf.CellFormat(partWidth-2, 8, fmt.Sprintf("%d", part.Parts), "", 0, "C", false, 0, "")
				}
				currentX += partWidth
			}

			// Border around the bar
			pdf.SetDrawColor(100, 100, 100)
			pdf.Rect(barX, barY, barWidth, barHeight, "D")
			pdf.SetTextColor(0, 0, 0)
		}
	}

	// Save PDF
	if err := pdf.OutputFileAndClose(savePath); err != nil {
		return fmt.Errorf("failed to save PDF: %w", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("Saved Paint-by-Numbers PDF to: %s", savePath))

	// Open the PDF with the default application
	exec.Command("open", savePath).Start()

	return nil
}

// hexToRGB converts a hex color string to RGB values
func hexToRGB(hex string) []int {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return []int{0, 0, 0}
	}
	var r, g, b int
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return []int{r, g, b}
}
