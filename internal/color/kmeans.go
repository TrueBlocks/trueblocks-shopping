package color

import (
	"image"
	"image/color"
	"math"
	"math/rand"
	"sort"

	"github.com/nfnt/resize"
)

// DominantColor represents an extracted dominant color
type DominantColor struct {
	R, G, B uint8
	Count   int // Number of pixels in this cluster
}

// ExtractDominantColors extracts k dominant colors from an image using K-Means clustering
func ExtractDominantColors(img image.Image, k int) []DominantColor {
	// Step 1: Resize image for performance (max 256x256)
	smallImg := resizeImage(img, 256)

	// Step 2: Extract all pixels
	pixels := extractPixels(smallImg)
	if len(pixels) == 0 {
		return nil
	}

	// Step 3: Create a deterministic random source based on image content
	// This ensures the same image always produces the same clustering
	seed := computeImageSeed(pixels)
	rng := rand.New(rand.NewSource(seed))

	// Step 4: Run K-Means++ clustering with deterministic RNG
	centroids := kMeansPlusPlus(pixels, k, 20, rng)

	// Step 5: Sort by cluster size (most dominant first)
	sort.Slice(centroids, func(i, j int) bool {
		return centroids[i].Count > centroids[j].Count
	})

	return centroids
}

// computeImageSeed generates a deterministic seed from image pixels
func computeImageSeed(pixels []pixel) int64 {
	// Use a simple hash of sampled pixel values
	var seed int64
	step := len(pixels) / 100
	if step < 1 {
		step = 1
	}
	for i := 0; i < len(pixels); i += step {
		p := pixels[i]
		seed = seed*31 + int64(p.r)
		seed = seed*31 + int64(p.g)
		seed = seed*31 + int64(p.b)
	}
	return seed
}

// resizeImage resizes an image to fit within maxDim while maintaining aspect ratio
func resizeImage(img image.Image, maxDim uint) image.Image {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// Determine new dimensions
	var newW, newH uint
	if w > h {
		newW = maxDim
		newH = uint(float64(h) * float64(maxDim) / float64(w))
	} else {
		newH = maxDim
		newW = uint(float64(w) * float64(maxDim) / float64(h))
	}

	// Use Lanczos3 for high-quality downsampling
	return resize.Resize(newW, newH, img, resize.Lanczos3)
}

// pixel represents an RGB pixel for clustering
type pixel struct {
	r, g, b float64
}

// extractPixels extracts all pixels from an image as float64 RGB values
func extractPixels(img image.Image) []pixel {
	bounds := img.Bounds()
	var pixels []pixel

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			r, g, b, a := c.RGBA()

			// Skip fully transparent pixels
			if a == 0 {
				continue
			}

			// Convert from 16-bit to 8-bit range
			pixels = append(pixels, pixel{
				r: float64(r >> 8),
				g: float64(g >> 8),
				b: float64(b >> 8),
			})
		}
	}

	return pixels
}

// centroid represents a cluster center
type centroid struct {
	r, g, b float64
	count   int
}

// kMeansPlusPlus performs K-Means clustering with K-Means++ initialization
func kMeansPlusPlus(pixels []pixel, k, maxIter int, rng *rand.Rand) []DominantColor {
	if len(pixels) == 0 || k <= 0 {
		return nil
	}

	// Limit k to number of unique pixels
	if k > len(pixels) {
		k = len(pixels)
	}

	// Step 1: K-Means++ initialization
	centroids := initializeCentroids(pixels, k, rng)

	// Step 2: Iterate until convergence or max iterations
	assignments := make([]int, len(pixels))

	for iter := 0; iter < maxIter; iter++ {
		// Assign each pixel to nearest centroid
		changed := false
		for i, p := range pixels {
			nearest := findNearestCentroid(p, centroids)
			if assignments[i] != nearest {
				assignments[i] = nearest
				changed = true
			}
		}

		// If no assignments changed, we've converged
		if !changed {
			break
		}

		// Update centroids
		centroids = updateCentroids(pixels, assignments, k)
	}

	// Convert to DominantColor results
	result := make([]DominantColor, len(centroids))
	for i, c := range centroids {
		result[i] = DominantColor{
			R:     uint8(math.Round(c.r)),
			G:     uint8(math.Round(c.g)),
			B:     uint8(math.Round(c.b)),
			Count: c.count,
		}
	}

	return result
}

// initializeCentroids uses K-Means++ to select initial centroids
func initializeCentroids(pixels []pixel, k int, rng *rand.Rand) []centroid {
	centroids := make([]centroid, 0, k)

	// Choose first centroid randomly (using deterministic RNG)
	idx := rng.Intn(len(pixels))
	centroids = append(centroids, centroid{
		r: pixels[idx].r,
		g: pixels[idx].g,
		b: pixels[idx].b,
	})

	// Choose remaining centroids with probability proportional to D(x)^2
	distances := make([]float64, len(pixels))

	for len(centroids) < k {
		// Calculate distance to nearest centroid for each pixel
		totalDist := 0.0
		for i, p := range pixels {
			minDist := math.MaxFloat64
			for _, c := range centroids {
				d := pixelDistanceSquared(p, c)
				if d < minDist {
					minDist = d
				}
			}
			distances[i] = minDist
			totalDist += minDist
		}

		// Choose next centroid with probability proportional to distance^2
		threshold := rng.Float64() * totalDist
		cumulative := 0.0
		for i, d := range distances {
			cumulative += d
			if cumulative >= threshold {
				centroids = append(centroids, centroid{
					r: pixels[i].r,
					g: pixels[i].g,
					b: pixels[i].b,
				})
				break
			}
		}
	}

	return centroids
}

// findNearestCentroid returns the index of the nearest centroid
func findNearestCentroid(p pixel, centroids []centroid) int {
	minDist := math.MaxFloat64
	nearest := 0

	for i, c := range centroids {
		d := pixelDistanceSquared(p, c)
		if d < minDist {
			minDist = d
			nearest = i
		}
	}

	return nearest
}

// pixelDistanceSquared calculates squared Euclidean distance
func pixelDistanceSquared(p pixel, c centroid) float64 {
	dr := p.r - c.r
	dg := p.g - c.g
	db := p.b - c.b
	return dr*dr + dg*dg + db*db
}

// updateCentroids recalculates centroids based on assigned pixels
func updateCentroids(pixels []pixel, assignments []int, k int) []centroid {
	sums := make([]struct{ r, g, b float64 }, k)
	counts := make([]int, k)

	for i, p := range pixels {
		cluster := assignments[i]
		sums[cluster].r += p.r
		sums[cluster].g += p.g
		sums[cluster].b += p.b
		counts[cluster]++
	}

	centroids := make([]centroid, k)
	for i := 0; i < k; i++ {
		if counts[i] > 0 {
			centroids[i] = centroid{
				r:     sums[i].r / float64(counts[i]),
				g:     sums[i].g / float64(counts[i]),
				b:     sums[i].b / float64(counts[i]),
				count: counts[i],
			}
		}
	}

	return centroids
}

// ColorDistance calculates the Euclidean distance between two colors
func ColorDistance(c1, c2 color.Color) float64 {
	r1, g1, b1, _ := c1.RGBA()
	r2, g2, b2, _ := c2.RGBA()

	dr := float64(r1>>8) - float64(r2>>8)
	dg := float64(g1>>8) - float64(g2>>8)
	db := float64(b1>>8) - float64(b2>>8)

	return math.Sqrt(dr*dr + dg*dg + db*db)
}
