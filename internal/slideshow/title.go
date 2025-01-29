package slideshow

import (
    "fmt"
    "image"
    "os"

    "github.com/hajimehoshi/ebiten/v2"
    // We include blank imports for standard image decoders
    _ "image/gif"
    _ "image/jpeg"
    _ "image/png"

    "github.com/electronjoe/OpenFrame/internal/photo"
)

const maxTileSize = 2048

// TiledImage holds one large image that may be split into multiple sub-images (tiles)
// if its dimensions exceed Ebiten’s max texture size (maxTileSize).
type TiledImage struct {
    tiles       []*ebiten.Image
    totalWidth  int
    totalHeight int
}

// loadTiledEbitenImage decodes an image from disk (using p.FilePath), applies any EXIF orientation
// transform, then splits it into sub-tiles if it's larger than Ebiten’s max texture size.
func loadTiledEbitenImage(p photo.Photo) (*TiledImage, error) {
    file, err := os.Open(p.FilePath)
    if err != nil {
        return nil, fmt.Errorf("unable to open file %s: %w", p.FilePath, err)
    }
    defer file.Close()

    // Decode the raw image (ignoring orientation at first)
    src, _, err := image.Decode(file)
    if err != nil {
        return nil, fmt.Errorf("unable to decode image %s: %w", p.FilePath, err)
    }

    // Apply orientation (rotate/flip if needed)
    src = applyEXIFOrientation(src, p.Orientation)

    // After orientation, determine final width & height
    w := src.Bounds().Dx()
    h := src.Bounds().Dy()

    // Now slice the (possibly large) image into tiles
    var tiles []*ebiten.Image
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

// applyEXIFOrientation rotates/flips the image based on the EXIF orientation value (1–8).
// Orientation reference:
//   1 - 0° (normal),   2 - flip horizontal,  3 - 180°,       4 - flip vertical
//   5 - transpose,     6 - rotate 90 CW,     7 - transverse, 8 - rotate 270 CW
func applyEXIFOrientation(src image.Image, orientation int) image.Image {
    switch orientation {
    case 2:
        return flipHorizontal(src)
    case 3:
        return rotate180(src)
    case 4:
        return flipVertical(src)
    case 5:
        return transpose(src)
    case 6:
        return rotate90(src)
    case 7:
        return transverse(src)
    case 8:
        return rotate270(src)
    default:
        // 1 => no transform
        return src
    }
}

// Below are helper functions for various flips/rotations.
// Each allocates a new RGBA and copies pixels accordingly.

// flipHorizontal: left-right mirror
func flipHorizontal(src image.Image) image.Image {
    b := src.Bounds()
    w, h := b.Dx(), b.Dy()
    dst := image.NewRGBA(b)
    for y := 0; y < h; y++ {
        for x := 0; x < w; x++ {
            dst.Set(w-1-x, y, src.At(x+b.Min.X, y+b.Min.Y))
        }
    }
    return dst
}

// flipVertical: top-bottom mirror
func flipVertical(src image.Image) image.Image {
    b := src.Bounds()
    w, h := b.Dx(), b.Dy()
    dst := image.NewRGBA(b)
    for y := 0; y < h; y++ {
        for x := 0; x < w; x++ {
            dst.Set(x, h-1-y, src.At(x+b.Min.X, y+b.Min.Y))
        }
    }
    return dst
}

// rotate180: 180° rotation
func rotate180(src image.Image) image.Image {
    // 180° is just horizontal + vertical flip
    b := src.Bounds()
    w, h := b.Dx(), b.Dy()
    dst := image.NewRGBA(b)
    for y := 0; y < h; y++ {
        for x := 0; x < w; x++ {
            dst.Set(w-1-x, h-1-y, src.At(x+b.Min.X, y+b.Min.Y))
        }
    }
    return dst
}

// rotate90: 90° clockwise
func rotate90(src image.Image) image.Image {
    b := src.Bounds()
    w, h := b.Dx(), b.Dy()
    // rotated image has size h x w
    dst := image.NewRGBA(image.Rect(0, 0, h, w))
    for y := 0; y < h; y++ {
        for x := 0; x < w; x++ {
            dst.Set(h-1-y, x, src.At(x+b.Min.X, y+b.Min.Y))
        }
    }
    return dst
}

// rotate270: 270° clockwise (or 90° CCW)
func rotate270(src image.Image) image.Image {
    // same as rotate90 three times
    b := src.Bounds()
    w, h := b.Dx(), b.Dy()
    dst := image.NewRGBA(image.Rect(0, 0, h, w))
    for y := 0; y < h; y++ {
        for x := 0; x < w; x++ {
            dst.Set(y, w-1-x, src.At(x+b.Min.X, y+b.Min.Y))
        }
    }
    return dst
}

// transpose: flip over top-left/bottom-right diagonal (x,y) -> (y,x)
func transpose(src image.Image) image.Image {
    b := src.Bounds()
    w, h := b.Dx(), b.Dy()
    dst := image.NewRGBA(image.Rect(0, 0, h, w))
    for y := 0; y < h; y++ {
        for x := 0; x < w; x++ {
            dst.Set(y, x, src.At(x+b.Min.X, y+b.Min.Y))
        }
    }
    return dst
}

// transverse: flip over top-right/bottom-left diagonal
// This is like transpose, then flip horizontally or vertically.
func transverse(src image.Image) image.Image {
    // do transpose first
    trans := transpose(src)
    // then flip horizontally
    return flipHorizontal(trans)
}

// Optionally, you could implement transpose in a single pass by direct indexing, etc.
// The above approach is straightforward for clarity.

// computeScale calculates a uniform scale so the image fits within screenW x screenH.
func computeScale(imgW, imgH, screenW, screenH int) float64 {
    if imgW == 0 || imgH == 0 {
        return 1.0
    }
    scaleW := float64(screenW) / float64(imgW)
    scaleH := float64(screenH) / float64(imgH)
    if scaleW < scaleH {
        return scaleW
    }
    return scaleH
}
