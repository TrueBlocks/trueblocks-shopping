package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	appkit "github.com/TrueBlocks/trueblocks-art/packages/appkit/v2"
)

// Settings stores application configuration and state
type Settings struct {
	// Window position and size
	WindowX      int `json:"windowX"`
	WindowY      int `json:"windowY"`
	WindowWidth  int `json:"windowWidth"`
	WindowHeight int `json:"windowHeight"`

	// View settings
	NColors         int    `json:"nColors"`
	TileSize        int    `json:"tileSize"`
	PosterizeMode   bool   `json:"posterizeMode"`
	SmoothingPasses int    `json:"smoothingPasses"`
	AspectRatio     string `json:"aspectRatio"` // "original", "landscape", "portrait", "square"

	// Last loaded image (auto-loaded on startup)
	LastImage string `json:"lastImage"`

	// Recently processed images
	RecentImages []RecentImage `json:"recentImages"`
}

// RecentImage represents a recently processed image
type RecentImage struct {
	OriginalPath string    `json:"originalPath"`
	CopiedPath   string    `json:"copiedPath"`
	Filename     string    `json:"filename"`
	ProcessedAt  time.Time `json:"processedAt"`
}

const (
	maxRecentImages = 20
	appDataDir      = ".local/share/trueblocks/acrylic"
	imagesDir       = "images"
	settingsFile    = "settings.json"
)

// getAppDataPath returns the full path to the app data directory
func getAppDataPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, appDataDir), nil
}

// getImagesPath returns the full path to the images directory
func GetImagesPath() (string, error) {
	appData, err := getAppDataPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(appData, imagesDir), nil
}

// getSettingsPath returns the full path to the settings file
func getSettingsPath() (string, error) {
	appData, err := getAppDataPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(appData, settingsFile), nil
}

// EnsureDirectories creates the necessary directories if they don't exist
func EnsureDirectories() error {
	imagesPath, err := GetImagesPath()
	if err != nil {
		return err
	}
	return os.MkdirAll(imagesPath, 0755)
}

// Load reads settings from disk
func Load() (*Settings, error) {
	settingsPath, err := getSettingsPath()
	if err != nil {
		return nil, err
	}

	// Return defaults if file doesn't exist
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		return &Settings{
			WindowWidth:     1280,
			WindowHeight:    800,
			NColors:         10,
			TileSize:        1,
			PosterizeMode:   false,
			SmoothingPasses: 0,
			AspectRatio:     "original",
			RecentImages:    []RecentImage{},
		}, nil
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, err
	}

	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		// Return defaults on parse error
		return &Settings{
			WindowWidth:     1280,
			WindowHeight:    800,
			NColors:         10,
			TileSize:        1,
			PosterizeMode:   false,
			SmoothingPasses: 0,
			AspectRatio:     "original",
			RecentImages:    []RecentImage{},
		}, nil
	}

	// Ensure valid defaults for new fields in old settings files
	if s.NColors < 2 {
		s.NColors = 10
	}
	if s.TileSize < 1 {
		s.TileSize = 1
	}
	if s.SmoothingPasses < 0 {
		s.SmoothingPasses = 0
	}
	if s.AspectRatio == "" {
		s.AspectRatio = "original"
	}

	return &s, nil
}

// Save writes settings to disk
func (s *Settings) Save() error {
	if err := EnsureDirectories(); err != nil {
		return err
	}

	settingsPath, err := getSettingsPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsPath, data, 0644)
}

// UpdateWindowPosition updates the window position in settings
func (s *Settings) UpdateWindowPosition(x, y int) {
	s.WindowX = x
	s.WindowY = y
}

// UpdateWindowSize updates the window size in settings
func (s *Settings) UpdateWindowSize(width, height int) {
	s.WindowWidth = width
	s.WindowHeight = height
}

// AddRecentImage adds an image to the recent list and copies it to the images directory
func (s *Settings) AddRecentImage(originalPath string) (string, error) {
	if err := EnsureDirectories(); err != nil {
		return "", err
	}

	imagesPath, err := GetImagesPath()
	if err != nil {
		return "", err
	}

	// Use original filename, overwrite if exists
	filename := filepath.Base(originalPath)
	destPath := filepath.Join(imagesPath, filename)

	// Copy the file (overwrites existing)
	if err := appkit.CopyFile(originalPath, destPath); err != nil {
		return "", err
	}

	// Set as last image (will be auto-loaded on startup)
	s.LastImage = destPath

	// Add to recent images
	recent := RecentImage{
		OriginalPath: originalPath,
		CopiedPath:   destPath,
		Filename:     filename,
		ProcessedAt:  time.Now(),
	}

	// Prepend to list
	s.RecentImages = append([]RecentImage{recent}, s.RecentImages...)

	// Trim to max size
	if len(s.RecentImages) > maxRecentImages {
		s.RecentImages = s.RecentImages[:maxRecentImages]
	}

	return destPath, nil
}

// GetLastImage returns the path to the last loaded image, or empty if none or file doesn't exist
func (s *Settings) GetLastImage() string {
	if s.LastImage == "" {
		return ""
	}
	// Verify file still exists
	if _, err := os.Stat(s.LastImage); os.IsNotExist(err) {
		return ""
	}
	return s.LastImage
}

// GetRecentImages returns the list of recently processed images
func (s *Settings) GetRecentImages() []RecentImage {
	return s.RecentImages
}

