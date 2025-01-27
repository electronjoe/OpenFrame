package photo

import (
    "fmt"
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
}

// Load walks each album directory, gathering metadata for each image file.
func Load(albumDirs []string) ([]Photo, error) {
    var photos []Photo
    for _, albumDir := range albumDirs {
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
                    // Not critical; just log a warning and skip
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
            // Log but continue; one bad directory shouldn’t break entire load
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
        return time.Time{}, fmt.Errorf("open file: %w", err)
    }
    defer f.Close()

    // Try EXIF
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
