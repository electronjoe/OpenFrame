package slideshow

import (
    "errors"
    "fmt"
    "time"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
    "github.com/electronjoe/OpenFrame/internal/photo"
)

type SlideshowGame struct {
    photos       []photo.Photo
    currentIndex int

    // Current slide data
    currentSlide *TiledImage
    loadingError error

    // Timing
    interval   time.Duration
    switchTime time.Time

    // Configurable flags
    dateOverlay bool
}

// NewSlideshowGame is a constructor function for SlideshowGame.
func NewSlideshowGame(
    photos []photo.Photo,
    interval time.Duration,
    dateOverlay bool,
) *SlideshowGame {
    return &SlideshowGame{
        photos:      photos,
        interval:    interval,
        switchTime:  time.Now().Add(interval),
        dateOverlay: dateOverlay,
    }
}

// Update is called every tick (about 60 times per second).
func (g *SlideshowGame) Update() error {
    // Exit on ESC
    if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
        return errors.New("exit requested")
    }

    // If it's time to switch to the next photo, do so
    if time.Now().After(g.switchTime) {
        g.advanceSlide()
    }
    return nil
}

// Draw is called every frame (also ~60 times per second).
func (g *SlideshowGame) Draw(screen *ebiten.Image) {
    if len(g.photos) == 0 {
        drawDebugString(screen, "No photos found.")
        return
    }

    if g.loadingError != nil {
        drawDebugString(screen, "Error loading image:\n"+g.loadingError.Error())
        return
    }

    if g.currentSlide == nil {
        // If the slide is not yet loaded or is in the process of being loaded
        drawDebugString(screen, "Loading slide...")
        return
    }

    // Actually draw the tiled image on screen
    drawTiledImage(screen, g.currentSlide)

    // If date overlay is enabled, draw the photo timestamp
    if g.dateOverlay {
        curPhoto := g.photos[g.currentIndex]
        drawDateOverlay(screen, curPhoto.TakenTime)
    }
}

// Layout is Ebiten’s required method to specify the game’s “logical” screen size.
func (g *SlideshowGame) Layout(outsideWidth, outsideHeight int) (int, int) {
    // Common choice for 1080p
    return 1920, 1080
}

// --------------- Public-facing helper methods --------------- //

// LoadCurrentSlide loads (decodes + tiles) the current Photo on demand.
func (g *SlideshowGame) LoadCurrentSlide() error {
    if g.currentIndex < 0 || g.currentIndex >= len(g.photos) {
        return fmt.Errorf("invalid currentIndex %d", g.currentIndex)
    }

    tiled, err := loadTiledEbitenImage(g.photos[g.currentIndex].FilePath)
    if err != nil {
        return err
    }

    g.currentSlide = tiled
    return nil
}

func (g *SlideshowGame) SetLoadingError(err error) {
    g.loadingError = err
}

// --------------- Internal logic --------------- //

func (g *SlideshowGame) advanceSlide() {
    // Move index
    g.currentIndex = (g.currentIndex + 1) % len(g.photos)

    // Free the old slide
    g.freeSlide()

    // Attempt to load the new slide
    if err := g.LoadCurrentSlide(); err != nil {
        g.loadingError = err
    } else {
        g.loadingError = nil
    }

    // Reset switch time
    g.switchTime = time.Now().Add(g.interval)
}

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
