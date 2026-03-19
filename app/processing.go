package app

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"

	appkit "github.com/TrueBlocks/trueblocks-art/packages/appkit/v2"
	"github.com/TrueBlocks/trueblocks-art/packages/color"
	"github.com/TrueBlocks/trueblocks-acrylic/v2/internal/db"
	"github.com/nfnt/resize"
)

type ProcessingResult struct {
	ProjectID int                          `json:"projectId"`
	Colors    []db.ProjectColorWithMatches `json:"colors"`
}

func (a *App) ProcessImage(imagePath string, name string, nColors int) (ProcessingResult, error) {
	dataDir := appkit.AppDirFor("acrylic")

	proj := db.Project{
		Name:     name,
		NColors:  nColors,
		TileSize: 1,
	}

	projectID, err := a.db.CreateProject(proj)
	if err != nil {
		return ProcessingResult{}, fmt.Errorf("create project: %w", err)
	}

	projDir := filepath.Join(dataDir, "projects", fmt.Sprintf("%d", projectID))
	if err := os.MkdirAll(projDir, 0755); err != nil {
		return ProcessingResult{}, fmt.Errorf("create project dir: %w", err)
	}

	ext := filepath.Ext(imagePath)
	destPath := filepath.Join(projDir, "original"+ext)
	if err := appkit.CopyFile(imagePath, destPath); err != nil {
		return ProcessingResult{}, fmt.Errorf("copy image: %w", err)
	}

	thumbPath := filepath.Join(projDir, "thumb"+ext)
	if err := generateThumbnail(imagePath, thumbPath); err != nil {
		return ProcessingResult{}, fmt.Errorf("generate thumbnail: %w", err)
	}

	relImage := filepath.Join("projects", fmt.Sprintf("%d", projectID), "original"+ext)
	relThumb := filepath.Join("projects", fmt.Sprintf("%d", projectID), "thumb"+ext)

	updateProj := db.Project{
		ID:            projectID,
		Name:          name,
		ImagePath:     relImage,
		ThumbnailPath: relThumb,
		NColors:       nColors,
		TileSize:      1,
		AspectRatio:   "original",
	}
	if err := a.db.UpdateProject(updateProj); err != nil {
		return ProcessingResult{}, fmt.Errorf("update project paths: %w", err)
	}

	result, err := a.processProjectColors(projectID, imagePath, nColors, false)
	if err != nil {
		return ProcessingResult{}, err
	}

	return result, nil
}

func (a *App) ReprocessProject(projectID int) (ProcessingResult, error) {
	proj, err := a.db.GetProject(projectID)
	if err != nil {
		return ProcessingResult{}, fmt.Errorf("get project: %w", err)
	}

	dataDir := appkit.AppDirFor("acrylic")
	imagePath := filepath.Join(dataDir, proj.ImagePath)

	if err := a.db.ClearProjectColors(projectID); err != nil {
		return ProcessingResult{}, fmt.Errorf("clear colors: %w", err)
	}

	return a.processProjectColors(projectID, imagePath, proj.NColors, proj.MatchOwnedOnly)
}

func (a *App) processProjectColors(projectID int, imagePath string, nColors int, ownedOnly bool) (ProcessingResult, error) {
	f, err := os.Open(imagePath)
	if err != nil {
		return ProcessingResult{}, fmt.Errorf("open image: %w", err)
	}
	defer func() { _ = f.Close() }()

	img, _, err := image.Decode(f)
	if err != nil {
		return ProcessingResult{}, fmt.Errorf("decode image: %w", err)
	}

	dominant := color.ExtractDominantColors(img, nColors)

	var paints []color.PaintProduct
	if ownedOnly {
		dbPaints, err := a.db.GetOwnedPaints()
		if err != nil {
			return ProcessingResult{}, fmt.Errorf("get owned paints: %w", err)
		}
		paints = dbPaintsToColorPaints(dbPaints)
	} else {
		dbPaints, err := a.db.GetPaints()
		if err != nil {
			return ProcessingResult{}, fmt.Errorf("get paints: %w", err)
		}
		paints = dbPaintsToColorPaints(dbPaints)
	}

	type colorResult struct {
		dominant color.DominantColor
		order    int
		matches  []color.PaintMatch
		recipe   []color.MixingPart
	}

	results := make([]colorResult, len(dominant))
	for i, dc := range dominant {
		matches := color.FindPaintMatches(paints, dc.R, dc.G, dc.B, "", 3)
		recipe := color.CalculateMixingRecipe(paints, dc.R, dc.G, dc.B, "")

		results[i] = colorResult{
			dominant: dc,
			order:    i,
			matches:  matches,
			recipe:   recipe,
		}
	}

	tx, err := a.db.Begin()
	if err != nil {
		return ProcessingResult{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, cr := range results {
		hex := color.RGBToHex(cr.dominant.R, cr.dominant.G, cr.dominant.B)
		res, err := tx.Exec(
			`INSERT INTO project_colors (project_id, sort_order, r, g, b, hex, pixel_count)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			projectID, cr.order, int(cr.dominant.R), int(cr.dominant.G), int(cr.dominant.B), hex, cr.dominant.Count)
		if err != nil {
			return ProcessingResult{}, fmt.Errorf("insert color: %w", err)
		}
		colorID64, _ := res.LastInsertId()
		colorID := int(colorID64)

		for rank, m := range cr.matches {
			res, err := tx.Exec(
				`INSERT INTO color_matches (color_id, match_type, rank, delta_e, match_rating)
				VALUES (?, ?, ?, ?, ?)`,
				colorID, "single", rank+1, m.DeltaE, m.MatchRating)
			if err != nil {
				return ProcessingResult{}, fmt.Errorf("insert match: %w", err)
			}
			matchID64, _ := res.LastInsertId()
			if _, err := tx.Exec(
				`INSERT INTO match_parts (match_id, paint_id, parts) VALUES (?, ?, ?)`,
				int(matchID64), m.Paint.ID, 1); err != nil {
				return ProcessingResult{}, fmt.Errorf("insert match part: %w", err)
			}
		}

		if len(cr.recipe) > 0 {
			mixedR, mixedG, mixedB := computeMixedColor(cr.recipe)
			mixedLab := color.RGBToLab(mixedR, mixedG, mixedB)
			targetLab := color.RGBToLab(cr.dominant.R, cr.dominant.G, cr.dominant.B)
			deltaE := color.DeltaE2000(targetLab, mixedLab)

			res, err := tx.Exec(
				`INSERT INTO color_matches (color_id, match_type, rank, delta_e, match_rating)
				VALUES (?, ?, ?, ?, ?)`,
				colorID, "recipe", 1, deltaE, color.MatchRating(deltaE))
			if err != nil {
				return ProcessingResult{}, fmt.Errorf("insert recipe match: %w", err)
			}
			matchID64, _ := res.LastInsertId()
			for _, part := range cr.recipe {
				if _, err := tx.Exec(
					`INSERT INTO match_parts (match_id, paint_id, parts) VALUES (?, ?, ?)`,
					int(matchID64), part.Paint.ID, part.Parts); err != nil {
					return ProcessingResult{}, fmt.Errorf("insert recipe part: %w", err)
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return ProcessingResult{}, fmt.Errorf("commit transaction: %w", err)
	}

	colors, err := a.db.GetProjectColorsWithMatches(projectID)
	if err != nil {
		return ProcessingResult{}, fmt.Errorf("get results: %w", err)
	}

	return ProcessingResult{ProjectID: projectID, Colors: colors}, nil
}

func (a *App) DeleteProjectWithFiles(id int) error {
	proj, err := a.db.GetProject(id)
	if err != nil {
		return fmt.Errorf("get project: %w", err)
	}

	if err := a.db.DeleteProject(id); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}

	dataDir := appkit.AppDirFor("acrylic")
	projDir := filepath.Dir(filepath.Join(dataDir, proj.ImagePath))
	if err := os.RemoveAll(projDir); err != nil {
		return fmt.Errorf("remove project directory: %w", err)
	}

	return nil
}

func (a *App) GetImagePath(relativePath string) string {
	return filepath.Join(appkit.AppDirFor("acrylic"), relativePath)
}

func dbPaintsToColorPaints(dbPaints []db.Paint) []color.PaintProduct {
	result := make([]color.PaintProduct, len(dbPaints))
	for i, p := range dbPaints {
		result[i] = color.PaintProduct{
			ID:       p.ID,
			Brand:    p.Brand,
			Name:     p.Name,
			Series:   p.Series,
			Opacity:  p.Opacity,
			Pigments: p.Pigments,
			RGB:      [3]uint8{uint8(p.R), uint8(p.G), uint8(p.B)},
			Hex:      p.Hex,
			Lab:      color.Lab{L: p.LabL, A: p.LabA, B: p.LabB},
		}
	}
	return result
}

func computeMixedColor(recipe []color.MixingPart) (uint8, uint8, uint8) {
	totalParts := 0
	var rSum, gSum, bSum float64
	for _, p := range recipe {
		totalParts += p.Parts
		rSum += float64(p.Paint.RGB[0]) * float64(p.Parts)
		gSum += float64(p.Paint.RGB[1]) * float64(p.Parts)
		bSum += float64(p.Paint.RGB[2]) * float64(p.Parts)
	}
	if totalParts == 0 {
		return 0, 0, 0
	}
	return uint8(rSum / float64(totalParts)),
		uint8(gSum / float64(totalParts)),
		uint8(bSum / float64(totalParts))
}

func generateThumbnail(srcPath, dstPath string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}

	thumb := resize.Thumbnail(80, 80, img, resize.Lanczos3)

	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	return encodePNG(out, thumb)
}
