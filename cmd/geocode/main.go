package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/rwcarlsen/goexif/exif"
)

// ImageMetadata holds the metadata for an image.
type ImageMetadata struct {
	// FriendlyLocation is a human-friendly geographic name (e.g. "Zion National Park")
	FriendlyLocation string  `json:"friendly_location"`
	// Optionally include the raw GPS coordinates
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
}

func main() {
	// Parse command-line flag for the root directory
	rootDir := flag.String("root", "", "Root directory containing sub-directories with images")
	flag.Parse()

	if *rootDir == "" {
		log.Fatal("Please provide a root directory using the -root flag")
	}

	// List entries in the root directory.
	entries, err := os.ReadDir(*rootDir)
	if err != nil {
		log.Fatalf("Failed to read root directory: %v", err)
	}

	// Process each sub-directory.
	for _, entry := range entries {
		if entry.IsDir() {
			subDirPath := filepath.Join(*rootDir, entry.Name())
			log.Printf("Processing sub-directory: %s", subDirPath)
			processSubDir(subDirPath)
		}
	}
}

// processSubDir processes one sub-directory:
// it scans for image files, extracts metadata from each image,
// and writes a metadata.json file mapping image filenames to their metadata.
func processSubDir(dir string) {
	// Map of image filename to its metadata.
	metadataMap := make(map[string]ImageMetadata)

	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("Failed to read directory %s: %v", dir, err)
		return
	}

	for _, entry := range entries {
		// Skip directories
		if entry.IsDir() {
			continue
		}

		// Process files with an image extension.
		if isImage(entry.Name()) {
			filePath := filepath.Join(dir, entry.Name())
			meta, err := extractMetadata(filePath)
			if err != nil {
				log.Printf("Error processing %s: %v", filePath, err)
				continue
			}
			metadataMap[entry.Name()] = meta
		}
	}

	// Write the metadata map as JSON into metadata.json in the current sub-directory.
	jsonPath := filepath.Join(dir, "metadata.json")
	jsonData, err := json.MarshalIndent(metadataMap, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal JSON for directory %s: %v", dir, err)
		return
	}
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		log.Printf("Failed to write JSON file %s: %v", jsonPath, err)
		return
	}

	log.Printf("Wrote metadata file: %s", jsonPath)
}

// isImage returns true if the fileName has a common image extension.
func isImage(fileName string) bool {
	lower := strings.ToLower(fileName)
	return strings.HasSuffix(lower, ".jpg") ||
		strings.HasSuffix(lower, ".jpeg") ||
		strings.HasSuffix(lower, ".png")
}

// extractMetadata opens the image file, extracts EXIF GPS information,
// and returns an ImageMetadata struct.
// If no GPS data is found, it returns an error.
func extractMetadata(filePath string) (ImageMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return ImageMetadata{}, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	// Decode EXIF data using goexif.
	x, err := exif.Decode(file)
	if err != nil {
		return ImageMetadata{}, fmt.Errorf("decoding exif: %w", err)
	}

	lat, long, err := x.LatLong()
	if err != nil {
		return ImageMetadata{}, fmt.Errorf("no GPS data: %w", err)
	}

	// Get a human friendly location name from the coordinates.
	friendly := reverseGeocode(lat, long)

	return ImageMetadata{
		FriendlyLocation: friendly,
		Latitude:         lat,
		Longitude:        long,
	}, nil
}

// reverseGeocode is a stub function that simulates converting latitude and longitude
// into a human-friendly geographic name. In a real implementation, you could
// call an external geocoding service (e.g. Google Geocoding API, Nominatim, etc.).
func reverseGeocode(lat, long float64) string {
	// For demonstration, we just return a formatted string.
	return fmt.Sprintf("Location at (%.5f, %.5f)", lat, long)
}

