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

// Photo represents a single photo's metadata (including orientation).
type Photo struct {
	FilePath    string
	TakenTime   time.Time
	Width       int
	Height      int
	Orientation int // EXIF orientation value, 1–8
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
				takenTime, width, height, orientation, err := extractMetadata(path)
				if err != nil {
					// Not critical; just log a warning and skip this file
					log.Printf("Warning: could not extract metadata for %s: %v", path, err)
					return nil
				}
				photos = append(photos, Photo{
					FilePath:    path,
					TakenTime:   takenTime,
					Width:       width,
					Height:      height,
					Orientation: orientation,
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

// extractMetadata obtains the photo's timestamp (from EXIF or file mod time),
// the image dimensions, and the EXIF orientation (1–8).
func extractMetadata(path string) (time.Time, int, int, int, error) {
	takenTime, orientation, err := extractTimeAndOrientation(path)
	if err != nil {
		return time.Time{}, 0, 0, 0, err
	}

	width, height, err := extractDimensions(path)
	if err != nil {
		return time.Time{}, 0, 0, 0, err
	}

	return takenTime, width, height, orientation, nil
}

// extractTimeAndOrientation reads EXIF data to get date/time and orientation.
// If not found, orientation defaults to 1 (no transform).
func extractTimeAndOrientation(path string) (time.Time, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return time.Time{}, 1, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	var takenTime time.Time
	var orientation = 1 // default if tag missing or invalid

	x, errDecode := exif.Decode(f)
	if errDecode == nil && x != nil {
		// Attempt to read EXIF DateTime
		if t, errDate := x.DateTime(); errDate == nil {
			takenTime = t
		}
		// Attempt to read Orientation tag
		tagOrient, errOrient := x.Get(exif.Orientation)
		if errOrient == nil && tagOrient != nil {
			if orientVal, errConv := tagOrient.Int(0); errConv == nil {
				orientation = orientVal
			}
		}
	}

	// Fallback to file mod time if EXIF time was not available
	if takenTime.IsZero() {
		info, errStat := os.Stat(path)
		if errStat == nil {
			takenTime = info.ModTime()
		} else {
			// If we somehow can't get mod time, just pick epoch
			takenTime = time.Unix(0, 0)
		}
	}

	return takenTime, orientation, nil
}

// extractDimensions uses image.DecodeConfig to get width and height
// without decoding the full image.
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
