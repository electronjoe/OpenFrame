package slideshow

import (
    "fmt"
    "image"
    "math"
    "os"

    "github.com/hajimehoshi/ebiten/v2"

	_ "image/jpeg"
)

const maxTileSize = 2048

// TiledImage holds one large image that may be split into multiple sub-images (tiles)
// if its dimensions exceed Ebiten’s max texture size.
type TiledImage struct {
    tiles       []*ebiten.Image
    totalWidth  int
    totalHeight int
}

// loadTiledEbitenImage decodes an image from disk, then splits it into multiple
// tiles if it's larger than Ebiten’s max texture size.
func loadTiledEbitenImage(filePath string) (*TiledImage, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return nil, fmt.Errorf("unable to open file %s: %w", filePath, err)
    }
    defer file.Close()

    src, _, err := image.Decode(file)
    if err != nil {
        return nil, fmt.Errorf("unable to decode image %s: %w", filePath, err)
    }

    w := src.Bounds().Dx()
    h := src.Bounds().Dy()

    var tiles []*ebiten.Image

    // Chop up the source image into sub-images each no larger than maxTileSize
    for y := 0; y < h; y += maxTileSize {
        for x := 0; x < w; x += maxTileSize {
            subRect := image.Rect(
                x,
                y,
                minInt(x+maxTileSize, w),
                minInt(y+maxTileSize, h),
            )

            subImg := src.(interface {
                SubImage(r image.Rectangle) image.Image
            }).SubImage(subRect)

            tile := ebiten.NewImageFromImage(subImg)
            tiles = append(tiles, tile)
        }
    }

    return &TiledImage{
        tiles:       tiles,
        totalWidth:  w,
        totalHeight: h,
    }, nil
}

func minInt(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func computeScale(imgW, imgH, screenW, screenH int) float64 {
    if imgW == 0 || imgH == 0 {
        return 1.0
    }
    scaleW := float64(screenW) / float64(imgW)
    scaleH := float64(screenH) / float64(imgH)
    return math.Min(scaleW, scaleH)
}
