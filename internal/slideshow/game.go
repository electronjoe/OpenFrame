package slideshow

import (
    "errors"
    "time"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"

    "github.com/electronjoe/OpenFrame/internal/cec"
    "github.com/electronjoe/OpenFrame/internal/photo"
)

// Slide holds up to two photos to be displayed side-by-side if both are portrait.
type Slide struct {
    Photos []photo.Photo // either 1 or 2 Photos
}

// BuildSlidesFromPhotos takes a set of photos and merges consecutive portraits
// into one Slide if side-by-side is desired.
func BuildSlidesFromPhotos(photos []photo.Photo) []Slide {
    var slides []Slide
    i := 0
    for i < len(photos) {
        current := photos[i]
        // Attempt to pair with next if it exists, both are portrait, etc.
        if i+1 < len(photos) {
            next := photos[i+1]
            if isPortrait(current) && isPortrait(next) && displayAllowsSideBySide() {
                slides = append(slides, Slide{Photos: []photo.Photo{current, next}})
                i += 2
                continue
            }
        }
        slides = append(slides, Slide{Photos: []photo.Photo{current}})
        i++
    }
    return slides
}

// isPortrait is a simple check: height > width (assuming it's stored in photo.Photo).
func isPortrait(p photo.Photo) bool {
    return p.Height > p.Width
}

// For simplicity, assume we generally allow side-by-side (e.g. 16:9 display).
func displayAllowsSideBySide() bool {
    return true
}

// SlideshowGame holds the state of our slideshow, including the slides, indexes, etc.
type SlideshowGame struct {
    slides            []Slide
    currentIndex      int
    currentTiledImages []*TiledImage
    loadingError      error

    interval   time.Duration
    switchTime time.Time

    dateOverlay bool
    paused      bool

    remoteCommandChan chan cec.RemoteCommand
}

// NewSlideshowGame creates a slideshow game struct.
func NewSlideshowGame(
    slides []Slide,
    interval time.Duration,
    dateOverlay bool,
) *SlideshowGame {
    return &SlideshowGame{
        slides:      slides,
        interval:    interval,
        switchTime:  time.Now().Add(interval),
        dateOverlay: dateOverlay,
    }
}

// SetRemoteCommandChan allows us to inject the remote events channel.
func (g *SlideshowGame) SetRemoteCommandChan(ch chan cec.RemoteCommand) {
    g.remoteCommandChan = ch
}

// Update is called by Ebiten ~60 times/sec. We read remote commands, handle them,
// and also auto-advance slides if not paused.
func (g *SlideshowGame) Update() error {
    // ESC to exit
    if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
        return errors.New("exit requested")
    }

    // Non-blocking read of remote commands
readLoop:
    for {
        select {
        case cmd := <-g.remoteCommandChan:
            g.handleRemoteCommand(cmd)
        default:
            break readLoop
        }
    }

    // If not paused, auto-advance slides on interval
    if !g.paused && time.Now().After(g.switchTime) {
        g.advanceSlide()
    }

    return nil
}

// handleRemoteCommand adjusts the slideshow based on remote input.
func (g *SlideshowGame) handleRemoteCommand(cmd cec.RemoteCommand) {
    switch cmd {
    case cec.RemoteLeft:
        g.previousSlide()
    case cec.RemoteRight:
        g.advanceSlide()
    case cec.RemoteSelect:
        g.paused = !g.paused
    default:
        // Unknown or unhandled
    }
}

// Draw is called every frame (~60fps). We render the current slide, plus any overlays.
func (g *SlideshowGame) Draw(screen *ebiten.Image) {
    // If there's a loading error, just display it
    if g.loadingError != nil {
        drawDebugString(screen, "Error loading image(s):\n"+g.loadingError.Error())
        return
    }

    // If no slides
    if len(g.slides) == 0 {
        drawDebugString(screen, "No slides found.")
        return
    }

    // Draw the current slide
    slide := g.slides[g.currentIndex]
    drawSlide(screen, slide, g.currentTiledImages, g.dateOverlay)

    // If paused, display an indicator in the top-left
    if g.paused {
        drawPauseIndicator(screen)
    }
}

// Layout sets the logical screen size. Ebiten will scale to the actual display.
func (g *SlideshowGame) Layout(outsideWidth, outsideHeight int) (int, int) {
    return 1920, 1080
}

// LoadCurrentSlide loads the images for the current index's slide.
func (g *SlideshowGame) LoadCurrentSlide() error {
    if g.currentIndex < 0 || g.currentIndex >= len(g.slides) {
        return nil
    }
    g.freeSlideImages()

    slide := g.slides[g.currentIndex]
    var newImages []*TiledImage
    for _, p := range slide.Photos {
        tiled, err := loadTiledEbitenImage(p)
        if err != nil {
            return err
        }
        newImages = append(newImages, tiled)
    }

    g.currentTiledImages = newImages
    return nil
}

// advanceSlide increments currentIndex (with wraparound) and loads that slide.
func (g *SlideshowGame) advanceSlide() {
    g.currentIndex = (g.currentIndex + 1) % len(g.slides)
    g.reloadSlide()
}

// previousSlide decrements currentIndex (with wraparound) and loads that slide.
func (g *SlideshowGame) previousSlide() {
    g.currentIndex = (g.currentIndex - 1 + len(g.slides)) % len(g.slides)
    g.reloadSlide()
}

// reloadSlide frees old images, loads new ones, and resets the slide timer.
func (g *SlideshowGame) reloadSlide() {
    g.freeSlideImages()
    if err := g.LoadCurrentSlide(); err != nil {
        g.loadingError = err
    } else {
        g.loadingError = nil
    }
    g.switchTime = time.Now().Add(g.interval)
}

// freeSlideImages disposes Ebiten images of the current slide (if any).
func (g *SlideshowGame) freeSlideImages() {
    if len(g.currentTiledImages) == 0 {
        return
    }
    for _, t := range g.currentTiledImages {
        for _, tile := range t.tiles {
            tile.Dispose()
        }
    }
    g.currentTiledImages = nil
}

// For completeness, if you also want the "SetLoadingError" method:
func (g *SlideshowGame) SetLoadingError(err error) {
    g.loadingError = err
}
