package photo

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

// Photo represents a single photo's metadata.
type Photo struct {
	FilePath  string
	TakenTime time.Time
	Width     int
	Height    int
}

// Load walks each album directory, gathering metadata for each image file.
func Load(albumDirs []string) ([]Photo, error) {
	var photos []Photo
	for _, albumDir := range albumDirs {
		err := filepath.WalkDir(albumDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				log.Printf("Error accessing %s: %v", path, err)
				// Skip this file/dir but keep walking
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if isImageFile(path) {
				takenTime, width, height, err := extractMetadata(path)
				if err != nil {
					// Not critical; just log a warning and skip this file
					log.Printf("Warning: could not extract metadata for %s: %v", path, err)
					return nil
				}
				photos = append(photos, Photo{
					FilePath:  path,
					TakenTime: takenTime,
					Width:     width,
					Height:    height,
				})
			}
			return nil
		})
		if err != nil {
			// Log but continue; one bad directory shouldn’t break the entire load
			log.Printf("Error walking directory %s: %v", albumDir, err)
		}
	}
	return photos, nil
}

// isImageFile checks for common image file extensions.
func isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif":
		return true
	}
	return false
}

// extractMetadata obtains the photo's timestamp (from EXIF or file mod time) and dimensions.
func extractMetadata(path string) (time.Time, int, int, error) {
	// 1) Extract the photo's timestamp
	takenTime, err := extractTakenTime(path)
	if err != nil {
		return time.Time{}, 0, 0, err
	}

	// 2) Extract the photo's width/height
	width, height, err := extractDimensions(path)
	if err != nil {
		return time.Time{}, 0, 0, err
	}

	return takenTime, width, height, nil
}

// extractTakenTime looks for EXIF DateTime; falls back to file mod time if unavailable.
func extractTakenTime(path string) (time.Time, error) {
	f, err := os.Open(path)
	if err != nil {
		return time.Time{}, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	// Try to decode EXIF
	x, err := exif.Decode(f)
	if err == nil && x != nil {
		if t, errDate := x.DateTime(); errDate == nil {
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

// extractDimensions uses image.DecodeConfig to get width and height without decoding the full image.
func extractDimensions(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, fmt.Errorf("open file for dimensions: %w", err)
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, fmt.Errorf("decode config failed for %s: %w", path, err)
	}
	return cfg.Width, cfg.Height, nil
}
