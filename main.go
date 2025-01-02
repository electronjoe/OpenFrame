package main

import (
    "encoding/json"
    "fmt"
    "io/fs"
    "io/ioutil"
    "log"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "time"

    "github.com/rwcarlsen/goexif/exif"
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
    Interval  int `json:"interval"`
    HdmiInput int `json:"hdmiInput"`
}

// Photo represents a single photo's metadata.
type Photo struct {
    FilePath  string
    TakenTime time.Time
    // Additional fields (latitude, longitude, etc.) can be added later
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

// loadPhotos walks through each album directory and collects photo metadata.
func loadPhotos(cfg Config) ([]Photo, error) {
    var photos []Photo

    for _, albumDir := range cfg.Albums {
        // Use WalkDir (or Walk) to traverse
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

// runConsoleSlideshow prints out the photo paths in a loop at the configured interval.
func runConsoleSlideshow(photos []Photo, intervalSecs int) {
    if len(photos) == 0 {
        fmt.Println("No photos found, nothing to display.")
        return
    }

    ticker := time.NewTicker(time.Duration(intervalSecs) * time.Second)
    defer ticker.Stop()

    idx := 0
    for {
        photo := photos[idx]
        fmt.Printf("Displaying: %s (Taken %s)\n", photo.FilePath, photo.TakenTime.Format(time.RFC3339))

        idx = (idx + 1) % len(photos)

        <-ticker.C // wait for next tick
    }
}

func main() {
    // 1. Read config
    cfg, err := readConfig()
    if err != nil {
        log.Fatalf("Failed to read config: %v", err)
    }

    // 2. Load and index photos
    photos, err := loadPhotos(cfg)
    if err != nil {
        log.Fatalf("Failed to load photos: %v", err)
    }

    // 3. Sort photos by TakenTime ascending
    sort.Slice(photos, func(i, j int) bool {
        return photos[i].TakenTime.Before(photos[j].TakenTime)
    })

    // 4. Simple console-based slideshow
    log.Printf("Loaded %d photos. Starting console slideshow...\n", len(photos))
    runConsoleSlideshow(photos, cfg.Interval)
}
