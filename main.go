package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io/fs"
	"io/ioutil"
	"log"
	"math"
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
	MaxTileSize = 2048
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
	HdmiInput int `json:"hdmiInput"` // (used for potential CEC features, not shown here)
}

// Photo represents a single photo's metadata.
type Photo struct {
	FilePath  string
	TakenTime time.Time
}

// TiledImage holds one large image that may be split into multiple sub-images
// (tiles) if its dimensions exceed Ebiten's max texture size.
type TiledImage struct {
	tiles       []*ebiten.Image // each tile is <= MaxTileSize
	totalWidth  int
	totalHeight int
}

// SlideshowGame implements the ebiten.Game interface.
type SlideshowGame struct {
	photos         []Photo
	currentIndex   int          // which Photo index we're displaying
	currentSlide   *TiledImage  // the loaded slide for the current Photo
	switchTime     time.Time    // time when we'll move to the next photo
	interval       time.Duration
	dateOverlay    bool
	loadingError   error        // if we hit an error loading a tile
}

// Update is called every tick (typically 60 times per second).
func (g *SlideshowGame) Update() error {
	// Exit on ESC
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return fmt.Errorf("exit requested")
	}

	// If it's time to switch to the next photo, do so
	if time.Now().After(g.switchTime) {
		g.advanceSlide()
	}

	return nil
}

// Draw is called every frame to render the screen.
func (g *SlideshowGame) Draw(screen *ebiten.Image) {
	// Clear screen to black
	screen.Fill(color.RGBA{0, 0, 0, 255})

	if len(g.photos) == 0 {
		ebitenutil.DebugPrint(screen, "No photos found.")
		return
	}

	if g.loadingError != nil {
		ebitenutil.DebugPrint(screen, "Error loading image:\n"+g.loadingError.Error())
		return
	}

	if g.currentSlide == nil {
		// If the slide is not yet loaded or is in the process of being loaded
		ebitenutil.DebugPrint(screen, "Loading slide...")
		return
	}

	// Draw the tiled image in a scaled, centered manner
	sw, sh := screen.Size()
	scale := computeScale(g.currentSlide.totalWidth, g.currentSlide.totalHeight, sw, sh)

	// We'll draw each tile in the correct position
	// relative to the scaled & centered bounding box of the entire image.
	totalW := float64(g.currentSlide.totalWidth) * scale
	totalH := float64(g.currentSlide.totalHeight) * scale
	offsetX := (float64(sw) - totalW) / 2
	offsetY := (float64(sh) - totalH) / 2

	// Each tile is originally placed at some sub-rectangle in the full image.
	// We'll re-draw them with appropriate offsets.
	maxSize := MaxTileSize

	tileIndex := 0

	for tileY := 0; tileY*maxSize < g.currentSlide.totalHeight; tileY++ {
		for tileX := 0; tileX*maxSize < g.currentSlide.totalWidth; tileX++ {
			// The tile's top-left corner in original (unscaled) coords
			subX := tileX * maxSize
			subY := tileY * maxSize

			// Create a draw operation
			op := &ebiten.DrawImageOptions{}
			// Move origin so (0,0) is center of the sub-tile
			op.GeoM.Translate(-float64(maxSize)/2, -float64(maxSize)/2)
			// Scale
			op.GeoM.Scale(scale, scale)
			// Then shift so the sub-tile is placed at the correct position within the full image
			// plus offset to center the full image on screen
			op.GeoM.Translate(
				offsetX+float64(subX)*scale+float64(maxSize)*scale/2,
				offsetY+float64(subY)*scale+float64(maxSize)*scale/2,
			)

			tile := g.currentSlide.tiles[tileIndex]
			screen.DrawImage(tile, op)
			tileIndex++
		}
	}

	// If dateOverlay is enabled, draw photo timestamp in bottom-left
	if g.dateOverlay {
		face := basicfont.Face7x13
		curPhoto := g.photos[g.currentIndex]
		dateStr := curPhoto.TakenTime.Format("2006-01-02 15:04:05")
		text.Draw(screen, dateStr, face, 20, sh-20, color.White)
	}
}

// Layout is Ebiten’s required method to specify the game’s “logical” screen size.
func (g *SlideshowGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Common choice for 1080p
	return 1920, 1080
}

// -------------------- Main Entry -------------------- //

func main() {
	cfg, err := readConfig()
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	photos, err := loadPhotos(cfg)
	if err != nil {
		log.Fatalf("Failed to load photos: %v", err)
	}
	if len(photos) == 0 {
		log.Println("No photos found. Exiting.")
		return
	}

	// Sort by date/time ascending
	sort.Slice(photos, func(i, j int) bool {
		return photos[i].TakenTime.Before(photos[j].TakenTime)
	})

	// Create our game struct
	game := &SlideshowGame{
		photos:      photos,
		interval:    time.Duration(cfg.Interval) * time.Second,
		switchTime:  time.Now().Add(time.Duration(cfg.Interval) * time.Second),
		dateOverlay: cfg.DateOverlay,
	}

	// Load the very first image on startup
	if err := game.loadCurrentSlide(); err != nil {
		game.loadingError = err
	}

	// Setup Ebiten
	ebiten.SetFullscreen(true)
	ebiten.SetWindowResizable(false)
	ebiten.SetWindowTitle("OpenFrame Slideshow")

	if err := ebiten.RunGame(game); err != nil {
		log.Fatalf("Ebiten run error: %v", err)
	}
}

// -------------------- SlideshowGame Methods -------------------- //

// advanceSlide moves to the next Photo, unloads old TiledImage, and loads the new one.
func (g *SlideshowGame) advanceSlide() {
	// Move index
	g.currentIndex = (g.currentIndex + 1) % len(g.photos)

	// Free the old slide
	g.freeSlide()

	// Attempt to load the new slide
	if err := g.loadCurrentSlide(); err != nil {
		g.loadingError = err
	} else {
		g.loadingError = nil
	}

	// Reset switch time
	g.switchTime = time.Now().Add(g.interval)
}

// loadCurrentSlide loads (decodes + tiles) the current Photo on demand.
func (g *SlideshowGame) loadCurrentSlide() error {
	if g.currentIndex < 0 || g.currentIndex >= len(g.photos) {
		return fmt.Errorf("invalid currentIndex %d", g.currentIndex)
	}
	p := g.photos[g.currentIndex]
	tiled, err := loadTiledEbitenImage(p.FilePath)
	if err != nil {
		return err
	}
	g.currentSlide = tiled
	return nil
}

// freeSlide disposes of the current TiledImage so it can be GC'd.
func (g *SlideshowGame) freeSlide() {
	if g.currentSlide == nil {
		return
	}
	// Each tile is an *ebiten.Image that can be disposed
	for _, t := range g.currentSlide.tiles {
		t.Dispose()
	}
	g.currentSlide = nil
}

// -------------------- Reading Config -------------------- //

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
	// Default interval if not set
	if cfg.Interval <= 0 {
		cfg.Interval = 10
	}
	return cfg, nil
}

// -------------------- Photo Loading -------------------- //

// loadPhotos walks each album directory, gathering metadata for each image file.
func loadPhotos(cfg Config) ([]Photo, error) {
	var photos []Photo
	for _, albumDir := range cfg.Albums {
		err := filepath.WalkDir(albumDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				log.Printf("Error accessing %s: %v", path, err)
				return nil // skip but continue
			}
			if d.IsDir() {
				return nil
			}
			if isImageFile(path) {
				t, err := extractTakenTime(path)
				if err != nil {
					log.Printf("Warning: could not extract time for %s: %v", path, err)
					return nil
				}
				photos = append(photos, Photo{
					FilePath:  path,
					TakenTime: t,
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

func isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif":
		return true
	}
	return false
}

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

// -------------------- Tiling Logic -------------------- //

// loadTiledEbitenImage decodes an image from disk, then splits it into multiple
// tiles if it's larger than the Ebiten max texture size, returning a TiledImage.
func loadTiledEbitenImage(filePath string) (*TiledImage, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open file %s: %w", filePath, err)
	}
	defer file.Close()

	src, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("unable to decode image %s: %w", filePath, err)
	}

	w := src.Bounds().Dx()
	h := src.Bounds().Dy()

	maxSize := MaxTileSize
	tiles := make([]*ebiten.Image, 0)

	// Chop up the source image into sub-images each no larger than maxSize
	for y := 0; y < h; y += maxSize {
		for x := 0; x < w; x += maxSize {
			subRect := image.Rect(x, y, minInt(x+maxSize, w), minInt(y+maxSize, h))
			subImg := src.(interface {
				SubImage(r image.Rectangle) image.Image
			}).SubImage(subRect)

			tile := ebiten.NewImageFromImage(subImg)
			tiles = append(tiles, tile)
		}
	}

	return &TiledImage{
		tiles:       tiles,
		totalWidth:  w,
		totalHeight: h,
	}, nil
}

// -------------------- Helpers -------------------- //

func computeScale(imgW, imgH, screenW, screenH int) float64 {
	if imgW == 0 || imgH == 0 {
		return 1.0
	}
	scaleW := float64(screenW) / float64(imgW)
	scaleH := float64(screenH) / float64(imgH)
	return math.Min(scaleW, scaleH)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
