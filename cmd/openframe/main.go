package main

import (
    "log"
    "math/rand"
    "sort"
    "time"

    "github.com/hajimehoshi/ebiten/v2"

    "github.com/electronjoe/OpenFrame/internal/cec"
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

    // 3. Sort or shuffle
    if cfg.Randomize {
        rand.Seed(time.Now().UnixNano())
        rand.Shuffle(len(photos), func(i, j int) {
            photos[i], photos[j] = photos[j], photos[i]
        })
    } else {
        sort.Slice(photos, func(i, j int) bool {
            return photos[i].TakenTime.Before(photos[j].TakenTime)
        })
    }

    // 4. Build slides
    slides := slideshow.BuildSlidesFromPhotos(photos)

    // 5. Create the slideshow game
    game := slideshow.NewSlideshowGame(
        slides,
        time.Duration(cfg.Interval)*time.Second,
        cfg.DateOverlay,
    )

    // 6. Load the first slide
    if err := game.LoadCurrentSlide(); err != nil {
        game.SetLoadingError(err)
    }

    // 7. Prepare remote command channel
    remoteEvents := make(chan cec.RemoteCommand, 10)
    // Start the CEC listener in a goroutine
    cec.StartCECListener(remoteEvents)

    // 8. Assign the channel to the game
    game.SetRemoteCommandChan(remoteEvents)

    // 9. Configure Ebiten
    ebiten.SetFullscreen(true)
    ebiten.SetWindowResizable(false)
    ebiten.SetWindowTitle("OpenFrame Slideshow")
    ebiten.SetCursorMode(ebiten.CursorModeHidden)

    // 10. Run the Ebiten game loop
    if err := ebiten.RunGame(game); err != nil {
        log.Fatalf("Ebiten run error: %v", err)
    }
}
