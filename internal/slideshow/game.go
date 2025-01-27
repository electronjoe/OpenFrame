package slideshow

import (
    "errors"
    "fmt"
    "time"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"

    "github.com/electronjoe/OpenFrame/internal/photo"
)

// Slide holds up to two photos to be displayed side-by-side if both are portrait.
type Slide struct {
    Photos []photo.Photo // 1 or 2 Photos
}

// BuildSlidesFromPhotos scans through all photos in chronological order and combines
// consecutive portrait images into a single Slide when feasible.
func BuildSlidesFromPhotos(photos []photo.Photo) []Slide {
    var slides []Slide
    i := 0
    for i < len(photos) {
        current := photos[i]
        // Attempt to pair with next if it exists, both are portrait, and we allow side-by-side
        if i+1 < len(photos) {
            next := photos[i+1]
            if isPortrait(current) && isPortrait(next) && displayAllowsSideBySide() {
                slides = append(slides, Slide{Photos: []photo.Photo{current, next}})
                i += 2
                continue
            }
        }
        // Otherwise, single-photo slide
        slides = append(slides, Slide{Photos: []photo.Photo{current}})
        i++
    }
    return slides
}

func isPortrait(p photo.Photo) bool {
    // A basic check: height > width
    return p.Height > p.Width
}

func displayAllowsSideBySide() bool {
    // In a typical scenario, always true if the display is wide enough (e.g., a 16:9 TV).
    // Here, we can keep it simple and return true, or add real dimension checks if desired.
    return true
}

// SlideshowGame orchestrates the slideshow of slides (which each contain 1 or 2 photos).
type SlideshowGame struct {
    slides      []Slide
    currentIndex int

    // currentTiledImages will hold the decoded TiledImages for the current slide:
    // 1 or 2 TiledImages depending on how many photos are in that slide.
    currentTiledImages []*TiledImage

    loadingError error

    // Timing
    interval   time.Duration
    switchTime time.Time

    // Overlay flags
    dateOverlay bool
}

// NewSlideshowGame initializes a slideshow from a slice of slides.
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

// Update is called every tick (~60 times/sec). Handles slide timing and key press.
func (g *SlideshowGame) Update() error {
    // Exit on ESC
    if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
        return errors.New("exit requested")
    }

    // If it's time to switch to the next slide, do so
    if time.Now().After(g.switchTime) {
        g.advanceSlide()
    }

    return nil
}

// Draw is called every frame (~60 times/sec). Renders the current slide.
func (g *SlideshowGame) Draw(screen *ebiten.Image) {
    // If no slides, just show a message
    if len(g.slides) == 0 {
        drawDebugString(screen, "No slides (photos) found.")
        return
    }

    // If there's a loading error, display it
    if g.loadingError != nil {
        drawDebugString(screen, "Error loading image(s):\n"+g.loadingError.Error())
        return
    }

    // If images aren’t loaded yet, display a “Loading” message
    if len(g.currentTiledImages) == 0 {
        drawDebugString(screen, "Loading slide...")
        return
    }

    // Actually draw the 1 or 2 TiledImages side by side
    slide := g.slides[g.currentIndex]
    drawSlide(screen, slide, g.currentTiledImages, g.dateOverlay)
}

// Layout specifies the “logical” screen size. Ebiten will scale to the actual screen.
func (g *SlideshowGame) Layout(outsideWidth, outsideHeight int) (int, int) {
    // Common for 1080p, but you could choose e.g., 1920 x 1080
    return 1920, 1080
}

// LoadCurrentSlide loads up to two TiledImages for the current slide.
func (g *SlideshowGame) LoadCurrentSlide() error {
    if g.currentIndex < 0 || g.currentIndex >= len(g.slides) {
        return fmt.Errorf("invalid currentIndex %d", g.currentIndex)
    }

    g.freeSlideImages()

    slide := g.slides[g.currentIndex]
    var newImages []*TiledImage

    // Load each photo in the slide
    for _, p := range slide.Photos {
        tiled, err := loadTiledEbitenImage(p.FilePath)
        if err != nil {
            return err
        }
        newImages = append(newImages, tiled)
    }

    g.currentTiledImages = newImages
    return nil
}

// SetLoadingError allows main() or other code to record a loading error that prevents normal draw.
func (g *SlideshowGame) SetLoadingError(err error) {
    g.loadingError = err
}

// internal: move to the next slide
func (g *SlideshowGame) advanceSlide() {
    // Move index
    g.currentIndex = (g.currentIndex + 1) % len(g.slides)

    // Free the old images
    g.freeSlideImages()

    // Attempt to load the new slide
    if err := g.LoadCurrentSlide(); err != nil {
        g.loadingError = err
    } else {
        g.loadingError = nil
    }

    // Reset switch time
    g.switchTime = time.Now().Add(g.interval)
}

// internal: dispose of any existing TiledImages from the current slide
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
