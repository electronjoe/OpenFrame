package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/rwcarlsen/goexif/exif"
	"golang.org/x/image/font/basicfont"

	_ "image/jpeg"
)

const (
	DefaultConfigPath = ".openframe/config.json"
)

// Config represents the JSON config structure.
type Config struct {
	Albums          []string `json:"albums"`
	DateOverlay     bool     `json:"dateOverlay"`
	LocationOverlay bool     `json:"locationOverlay"`
	Schedule        struct {
		OnTime  string `json:"onTime"`
		OffTime string `json:"offTime"`
	} `json:"schedule"`
	Interval  int `json:"interval"`  // Slideshow interval (seconds)
	HdmiInput int `json:"hdmiInput"` // Not used in Phase 2, relevant for Phase 3 (CEC)
}

// Photo represents a single photo's metadata.
type Photo struct {
	FilePath  string
	TakenTime time.Time
	// Additional fields (latitude, longitude, etc.) can be added later
}

// SlideshowGame implements the ebiten.Game interface for the photo slideshow.
// It only holds the photo metadata array plus a *single* loaded image at a time.
type SlideshowGame struct {
	photos       []Photo // All photo metadata (no in-memory decoded images)
	currentIndex int
	currentImage *ebiten.Image

	switchTime  time.Time     // When to move to the next photo
	interval    time.Duration // Interval between photos
	dateOverlay bool
}

// Update is called every tick (typically 60 times per second).
func (g *SlideshowGame) Update() error {
	// Check for Exit key
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return fmt.Errorf("exit requested")
	}

	// Check if it's time to move to the next photo
	if time.Now().After(g.switchTime) {
		g.currentIndex = (g.currentIndex + 1) % len(g.photos)
		if err := g.loadCurrentImage(); err != nil {
			log.Printf("Failed to load image: %v", err)
		}
		g.switchTime = time.Now().Add(g.interval)
	}
	return nil
}

// Draw renders the current photo and overlays (date, location, etc.).
func (g *SlideshowGame) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 255}) // Clear screen with black

	if len(g.photos) == 0 {
		ebitenutil.DebugPrint(screen, "No photos to display.")
		return
	}

	if g.currentImage == nil {
		ebitenutil.DebugPrint(screen, "Image not loaded.")
		return
	}

	// Calculate scaling to fit screen
	sw, sh := screen.Size()
	iw, ih := g.currentImage.Size()

	// Scale to fit, preserving aspect ratio
	scale := min(float64(sw)/float64(iw), float64(sh)/float64(ih))

	// Center the image
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(iw)/2, -float64(ih)/2) // Move origin to center of image
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(sw)/2, float64(sh)/2) // Move image to the center of screen

	screen.DrawImage(g.currentImage, op)

	// If date overlay is enabled, draw the date/time in the bottom-left corner
	if g.dateOverlay {
		taken := g.photos[g.currentIndex].TakenTime
		dateStr := taken.Format("2006-01-02 15:04:05") // e.g., 2023-12-31 15:04:05
		face := basicfont.Face7x13
		text.Draw(screen, dateStr, face, 20, sh-20, color.White)
	}
}

// Layout returns the game’s logical screen size; Ebiten will scale it to the actual window.
func (g *SlideshowGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	// A common choice: 1920x1080 for a TV
	return 1920, 1080
}

// loadCurrentImage loads (or reloads) the image for the currentIndex photo.
func (g *SlideshowGame) loadCurrentImage() error {
	// If we already have an image loaded, we can let it go (for GC).
	// (Ebiten Images are cleaned up when dereferenced; there's no explicit Dispose method.)
	g.currentImage = nil

	photo := g.photos[g.currentIndex]
	file, err := os.Open(photo.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	g.currentImage = ebiten.NewImageFromImage(img)
	return nil
}

// -------------------- Main entry point -------------------- //

func main() {
	// 1. Read config
	cfg, err := readConfig()
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	// 2. Load and index photo metadata (paths + times)
	photos, err := loadPhotos(cfg)
	if err != nil {
		log.Fatalf("Failed to load photos: %v", err)
	}
	if len(photos) == 0 {
		log.Println("No photos found. Exiting.")
		return
	}

	// 3. Sort photos by TakenTime ascending
	sort.Slice(photos, func(i, j int) bool {
		return photos[i].TakenTime.Before(photos[j].TakenTime)
	})

	// 4. Create the slideshow game struct (just metadata + first image loaded)
	interval := time.Duration(cfg.Interval) * time.Second
	game := &SlideshowGame{
		photos:      photos,
		interval:    interval,
		dateOverlay: cfg.DateOverlay,
	}
	// Manually load the first photo right away
	if err := game.loadCurrentImage(); err != nil {
		log.Printf("Failed to load initial image: %v", err)
	}
	game.switchTime = time.Now().Add(interval)

	// 5. Configure Ebiten window for full-screen
	ebiten.SetFullscreen(true)
	ebiten.SetWindowResizable(false)
	ebiten.SetWindowTitle("OpenFrame Slideshow")

	// 6. Run the slideshow
	if err := ebiten.RunGame(game); err != nil {
		log.Fatalf("Ebiten run error: %v", err)
	}
}

// readConfig loads and parses the JSON config from ~/.openframe/config.json
func readConfig() (Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, DefaultConfigPath)
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file at %s: %w", configPath, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Apply some defaults if needed
	if cfg.Interval == 0 {
		cfg.Interval = 10 // default to 10 seconds
	}

	return cfg, nil
}

// loadPhotos walks through each album directory and collects photo metadata (paths + times).
func loadPhotos(cfg Config) ([]Photo, error) {
	var photos []Photo

	for _, albumDir := range cfg.Albums {
		err := filepath.WalkDir(albumDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				// If an error occurs walking this path, log & skip
				log.Printf("Error accessing path %s: %v", path, err)
				return nil
			}
			if d.IsDir() {
				return nil
			}
			// Simple filter for image files by extension
			if isImageFile(path) {
				takenTime, err := extractTakenTime(path)
				if err != nil {
					// Log a warning, skip
					log.Printf("Warning: could not extract time for %s: %v", path, err)
					return nil
				}
				photos = append(photos, Photo{
					FilePath:  path,
					TakenTime: takenTime,
				})
			}
			return nil
		})
		if err != nil {
			log.Printf("Error walking directory %s: %v", albumDir, err)
		}
	}

	return photos, nil
}

// isImageFile does a naive file extension check for JPEG/PNG/etc.
func isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif":
		return true
	}
	return false
}

// extractTakenTime attempts to read EXIF data; falls back to file mod time if EXIF not found.
func extractTakenTime(path string) (time.Time, error) {
	f, err := os.Open(path)
	if err != nil {
		return time.Time{}, err
	}
	defer f.Close()

	// Try EXIF
	x, err := exif.Decode(f)
	if err == nil && x != nil {
		t, errDate := x.DateTime()
		if errDate == nil {
			return t, nil
		}
	}

	// Fallback: file mod time
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// min is a helper function to return the smaller of two floats.
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
