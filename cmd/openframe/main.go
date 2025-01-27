package main

import (
    "log"
    "sort"
    "time"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/electronjoe/OpenFrame/internal/config"
    "github.com/electronjoe/OpenFrame/internal/photo"
    "github.com/electronjoe/OpenFrame/internal/slideshow"
)

func main() {
    // 1. Read config
    cfg, err := config.Read()
    if err != nil {
        log.Fatalf("Failed to read config: %v", err)
    }

    // 2. Load photos
    photos, err := photo.Load(cfg.Albums)
    if err != nil {
        log.Fatalf("Failed to load photos: %v", err)
    }
    if len(photos) == 0 {
        log.Println("No photos found. Exiting.")
        return
    }

    // 3. Sort by date/time ascending
    sort.Slice(photos, func(i, j int) bool {
        return photos[i].TakenTime.Before(photos[j].TakenTime)
    })

    // 4. Create our slideshow game
    game := slideshow.NewSlideshowGame(
        photos,
        time.Duration(cfg.Interval)*time.Second,
        cfg.DateOverlay,
    )

    // 5. Load the very first image on startup
    if err := game.LoadCurrentSlide(); err != nil {
        game.SetLoadingError(err)
    }

    // 6. Configure Ebiten
    ebiten.SetFullscreen(true)
    ebiten.SetWindowResizable(false)
    ebiten.SetWindowTitle("OpenFrame Slideshow")

    // 7. Run Ebiten game loop
    if err := ebiten.RunGame(game); err != nil {
        log.Fatalf("Ebiten run error: %v", err)
    }
}
