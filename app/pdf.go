package app

import (
	"fmt"
	"path/filepath"

	appkit "github.com/TrueBlocks/trueblocks-art/packages/appkit/v2"
	"github.com/jung-kurt/gofpdf"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/TrueBlocks/trueblocks-acrylic/v2/internal/db"
)

func (a *App) ExportComparisonPDF(projectID int) (string, error) {
	proj, err := a.db.GetProject(projectID)
	if err != nil {
		return "", fmt.Errorf("get project: %w", err)
	}

	colors, err := a.db.GetProjectColorsWithMatches(projectID)
	if err != nil {
		return "", fmt.Errorf("get colors: %w", err)
	}

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: proj.Name + " - Comparison.pdf",
		Filters: []runtime.FileFilter{
			{DisplayName: "PDF Files", Pattern: "*.pdf"},
		},
	})
	if err != nil || savePath == "" {
		return "", err
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 18)
	pdf.Cell(190, 12, proj.Name+" - Color Comparison")
	pdf.Ln(16)

	for _, cwm := range colors {
		c := cwm.Color
		pdf.SetFillColor(c.R, c.G, c.B)
		pdf.Rect(10, pdf.GetY(), 15, 10, "F")
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetX(28)
		pdf.Cell(40, 10, c.Hex)

		for _, m := range cwm.Matches {
			pdf.SetX(70)
			desc := fmt.Sprintf("%s (ΔE %.1f)", m.MatchRating, m.DeltaE)
			pdf.Cell(60, 10, desc)
			for _, part := range m.Parts {
				pdf.SetX(130)
				pdf.Cell(60, 10, fmt.Sprintf("%s x%d", part.Paint.Name, part.Parts))
				pdf.Ln(5)
			}
		}
		pdf.Ln(8)

		if pdf.GetY() > 260 {
			pdf.AddPage()
		}
	}

	if err := pdf.OutputFileAndClose(savePath); err != nil {
		return "", fmt.Errorf("write PDF: %w", err)
	}
	return savePath, nil
}

func (a *App) ExportPaintByNumbersPDF(projectID int) (string, error) {
	proj, err := a.db.GetProject(projectID)
	if err != nil {
		return "", fmt.Errorf("get project: %w", err)
	}

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: proj.Name + " - Paint by Numbers.pdf",
		Filters: []runtime.FileFilter{
			{DisplayName: "PDF Files", Pattern: "*.pdf"},
		},
	})
	if err != nil || savePath == "" {
		return "", err
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 18)
	pdf.Cell(190, 12, proj.Name+" - Paint by Numbers")
	pdf.Ln(16)
	pdf.SetFont("Helvetica", "", 12)
	pdf.Cell(190, 10, "Paint-by-numbers export coming soon.")

	if err := pdf.OutputFileAndClose(savePath); err != nil {
		return "", fmt.Errorf("write PDF: %w", err)
	}
	return savePath, nil
}

func (a *App) ExportColorDetailPDF(projectID int, colorIndex int) (string, error) {
	proj, err := a.db.GetProject(projectID)
	if err != nil {
		return "", fmt.Errorf("get project: %w", err)
	}

	colors, err := a.db.GetProjectColorsWithMatches(projectID)
	if err != nil {
		return "", fmt.Errorf("get colors: %w", err)
	}

	if colorIndex < 0 || colorIndex >= len(colors) {
		return "", fmt.Errorf("color index %d out of range", colorIndex)
	}

	cwm := colors[colorIndex]
	c := cwm.Color

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: fmt.Sprintf("%s - Color %s.pdf", proj.Name, c.Hex),
		Filters: []runtime.FileFilter{
			{DisplayName: "PDF Files", Pattern: "*.pdf"},
		},
	})
	if err != nil || savePath == "" {
		return "", err
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 18)
	pdf.Cell(190, 12, fmt.Sprintf("Color Detail: %s", c.Hex))
	pdf.Ln(16)

	pdf.SetFillColor(c.R, c.G, c.B)
	pdf.Rect(10, pdf.GetY(), 40, 40, "F")
	pdf.SetX(55)
	pdf.SetFont("Helvetica", "", 12)
	pdf.Cell(60, 10, fmt.Sprintf("RGB: %d, %d, %d", c.R, c.G, c.B))
	pdf.Ln(12)

	writeMatches(pdf, cwm.Matches)

	if err := pdf.OutputFileAndClose(savePath); err != nil {
		return "", fmt.Errorf("write PDF: %w", err)
	}
	return savePath, nil
}

func (a *App) ExportShoppingListPDF(projectID int) (string, error) {
	proj, err := a.db.GetProject(projectID)
	if err != nil {
		return "", fmt.Errorf("get project: %w", err)
	}

	colors, err := a.db.GetProjectColorsWithMatches(projectID)
	if err != nil {
		return "", fmt.Errorf("get colors: %w", err)
	}

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: proj.Name + " - Shopping List.pdf",
		Filters: []runtime.FileFilter{
			{DisplayName: "PDF Files", Pattern: "*.pdf"},
		},
	})
	if err != nil || savePath == "" {
		return "", err
	}

	needed := collectNeededPaints(colors)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 18)
	pdf.Cell(190, 12, proj.Name+" - Shopping List")
	pdf.Ln(16)

	if len(needed) == 0 {
		pdf.SetFont("Helvetica", "", 12)
		pdf.Cell(190, 10, "You have all the paints you need!")
	} else {
		pdf.SetFont("Helvetica", "B", 10)
		pdf.Cell(80, 8, "Paint Name")
		pdf.Cell(30, 8, "Brand")
		pdf.Cell(20, 8, "Series")
		pdf.Ln(10)
		pdf.SetFont("Helvetica", "", 10)
		for _, p := range needed {
			pdf.Cell(80, 8, p.Name)
			pdf.Cell(30, 8, p.Brand)
			pdf.Cell(20, 8, fmt.Sprintf("%d", p.Series))
			pdf.Ln(8)
		}
	}

	if err := pdf.OutputFileAndClose(savePath); err != nil {
		return "", fmt.Errorf("write PDF: %w", err)
	}
	return savePath, nil
}

func writeMatches(pdf *gofpdf.Fpdf, matches []db.ColorMatch) {
	y := pdf.GetY() + 45
	pdf.SetY(y)
	for _, m := range matches {
		pdf.SetFont("Helvetica", "B", 11)
		var label string
		if m.MatchType == "recipe" {
			label = "Mixing Recipe"
		} else {
			label = fmt.Sprintf("Match #%d", m.Rank)
		}
		pdf.Cell(60, 8, label)
		pdf.SetFont("Helvetica", "", 10)
		pdf.Cell(60, 8, fmt.Sprintf("%s (ΔE %.1f)", m.MatchRating, m.DeltaE))
		pdf.Ln(8)

		for _, part := range m.Parts {
			pdf.SetX(20)
			pdf.SetFillColor(part.Paint.R, part.Paint.G, part.Paint.B)
			pdf.Rect(20, pdf.GetY(), 8, 6, "F")
			pdf.SetX(30)
			info := fmt.Sprintf("%s (%s) — %d parts", part.Paint.Name, part.Paint.Brand, part.Parts)
			pdf.Cell(150, 6, info)
			pdf.Ln(7)
		}
		pdf.Ln(4)
	}
}

func collectNeededPaints(colors []db.ProjectColorWithMatches) []db.Paint {
	seen := make(map[string]bool)
	var needed []db.Paint
	for _, cwm := range colors {
		for _, m := range cwm.Matches {
			for _, part := range m.Parts {
				if !part.Paint.Owned && !seen[part.Paint.ID] {
					seen[part.Paint.ID] = true
					needed = append(needed, part.Paint)
				}
			}
		}
	}
	return needed
}

func (a *App) GetProjectImageAbsPath(projectID int) (string, error) {
	proj, err := a.db.GetProject(projectID)
	if err != nil {
		return "", err
	}
	return filepath.Join(appkit.AppDirFor("acrylic"), proj.ImagePath), nil
}
