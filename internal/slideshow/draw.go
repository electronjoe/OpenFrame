package slideshow

import (
    "image/color"
    "time"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/ebitenutil"
    "github.com/hajimehoshi/ebiten/v2/text"
    "golang.org/x/image/font/basicfont"
)

// drawDebugString prints text in the top-left corner of the screen.
// Used for errors and debug messages.
func drawDebugString(screen *ebiten.Image, msg string) {
    screen.Fill(color.RGBA{0, 0, 0, 255}) // Clear to black
    ebitenutil.DebugPrint(screen, msg)
}

// drawSlide is the main function for rendering the current slide,
// which may have 1 or 2 photos (represented by up to 2 TiledImages).
func drawSlide(screen *ebiten.Image, slide Slide, tiledImages []*TiledImage, dateOverlay bool) {
    screen.Fill(color.RGBA{0, 0, 0, 255}) // Clear to black

    if len(tiledImages) == 1 {
        // Single-photo slide
        drawSingleImage(screen, tiledImages[0])
        if dateOverlay && len(slide.Photos) == 1 {
            drawDateOverlayLeft(screen, slide.Photos[0].TakenTime)
        }
    } else if len(tiledImages) == 2 {
        // Two-photo slide
        drawTwoPortraitsSideBySide(screen, tiledImages[0], tiledImages[1])

        // Draw date overlays bottom-left and bottom-right
        if dateOverlay && len(slide.Photos) == 2 {
            drawDateOverlayLeft(screen, slide.Photos[0].TakenTime)
            drawDateOverlayRight(screen, slide.Photos[1].TakenTime)
        }
    }
}

// drawSingleImage centers & scales one TiledImage to fit the screen.
func drawSingleImage(screen *ebiten.Image, t *TiledImage) {
    sw, sh := screen.Size()
    scale := computeScale(t.totalWidth, t.totalHeight, sw, sh)

    totalW := float64(t.totalWidth) * scale
    totalH := float64(t.totalHeight) * scale
    offsetX := (float64(sw) - totalW) / 2
    offsetY := (float64(sh) - totalH) / 2

    tileIndex := 0
    for tileY := 0; tileY*maxTileSize < t.totalHeight; tileY++ {
        for tileX := 0; tileX*maxTileSize < t.totalWidth; tileX++ {
            subX := tileX * maxTileSize
            subY := tileY * maxTileSize

            op := &ebiten.DrawImageOptions{}
            // translate center to (0,0)
            op.GeoM.Translate(-float64(maxTileSize)/2, -float64(maxTileSize)/2)
            // scale
            op.GeoM.Scale(scale, scale)
            // move back
            op.GeoM.Translate(
                offsetX+float64(subX)*scale+float64(maxTileSize)*scale/2,
                offsetY+float64(subY)*scale+float64(maxTileSize)*scale/2,
            )

            tile := t.tiles[tileIndex]
            screen.DrawImage(tile, op)
            tileIndex++
        }
    }
}

// drawTwoPortraitsSideBySide draws two portrait TiledImages (leftImg and rightImg)
// side by side on the given Ebiten screen. Each image is scaled independently
// so that it fits within half the screen’s width (and the full screen height)
// while retaining its aspect ratio. The left image is centered in the left half,
// and the right image is centered in the right half, maximizing each image’s size
// without overflowing their respective half of the screen.
func drawTwoPortraitsSideBySide(screen *ebiten.Image, leftImg, rightImg *TiledImage) {
    sw, sh := screen.Size()

    // Original dimensions
    lw, lh := leftImg.totalWidth, leftImg.totalHeight
    rw, rh := rightImg.totalWidth, rightImg.totalHeight

    // Separate scale factors: each must fit in sw/2 x sh
    leftScale := computeScale(lw, lh, sw/2, sh)
    scaledLW := float64(lw) * leftScale
    scaledLH := float64(lh) * leftScale

    rightScale := computeScale(rw, rh, sw/2, sh)
    scaledRW := float64(rw) * rightScale
    scaledRH := float64(rh) * rightScale

    // Center each in its own half horizontally, and in full screen vertically
    leftX := (float64(sw)/2 - scaledLW) / 2
    leftY := float64(sh)/2 - scaledLH/2

    rightX := float64(sw)/2 + ((float64(sw)/2 - scaledRW) / 2)
    rightY := float64(sh)/2 - scaledRH/2

    // Now draw them
    drawTiledImage(screen, leftImg, leftScale, leftX, leftY)
    drawTiledImage(screen, rightImg, rightScale, rightX, rightY)
}

// Helper that draws a TiledImage at (offsetX, offsetY) using the given scale.
func drawTiledImage(screen *ebiten.Image, t *TiledImage, scale, offsetX, offsetY float64) {
    tileIndex := 0
    for tileY := 0; tileY*maxTileSize < t.totalHeight; tileY++ {
        for tileX := 0; tileX*maxTileSize < t.totalWidth; tileX++ {
            subX := tileX * maxTileSize
            subY := tileY * maxTileSize

            op := &ebiten.DrawImageOptions{}

            // Translate to tile center so we can apply scale around the center
            op.GeoM.Translate(-float64(maxTileSize)/2, -float64(maxTileSize)/2)
            op.GeoM.Scale(scale, scale)

            // Compute the final on-screen position for this tile
            xPos := offsetX + float64(subX)*scale + float64(maxTileSize)*scale/2
            yPos := offsetY + float64(subY)*scale + float64(maxTileSize)*scale/2

            op.GeoM.Translate(xPos, yPos)

            tile := t.tiles[tileIndex]
            screen.DrawImage(tile, op)
            tileIndex++
        }
    }
}

// Utility for integer max.
func maxInt(a, b int) int {
    if a > b {
        return a
    }
    return b
}

// drawTiledImageWithOffset is a helper to position a TiledImage within a given bounding box.
func drawTiledImageWithOffset(screen *ebiten.Image, t *TiledImage, scale float64,
    offsetX, offsetY, boxWidth, boxHeight int) {

    tileIndex := 0
    for tileY := 0; tileY*maxTileSize < t.totalHeight; tileY++ {
        for tileX := 0; tileX*maxTileSize < t.totalWidth; tileX++ {
            subX := tileX * maxTileSize
            subY := tileY * maxTileSize

            op := &ebiten.DrawImageOptions{}
            // translate to center
            op.GeoM.Translate(-float64(maxTileSize)/2, -float64(maxTileSize)/2)
            op.GeoM.Scale(scale, scale)

            // The top-left corner for this sub-tile
            // We center it vertically within the boxHeight, but place it at offsetX horizontally.
            totalH := float64(t.totalHeight) * scale
            xPos := float64(offsetX) + (float64(subX)*scale + float64(maxTileSize)*scale/2)
            yPos := float64(offsetY) + (float64(boxHeight)-totalH)/2 + // center in box
                float64(subY)*scale + float64(maxTileSize)*scale/2

            op.GeoM.Translate(xPos, yPos)

            tile := t.tiles[tileIndex]
            screen.DrawImage(tile, op)
            tileIndex++
        }
    }
}

// drawDateOverlayLeft places the photo timestamp in the bottom-left corner.
func drawDateOverlayLeft(screen *ebiten.Image, takenTime time.Time) {
    face := basicfont.Face7x13
    _, sh := screen.Size()
    dateStr := takenTime.Format("2006-01-02")

    x := 20
    y := sh - 20
    text.Draw(screen, dateStr, face, x, y, color.White)
}

// drawPauseIndicator places Pause notification text at top left of the screen.
func drawPauseIndicator(screen *ebiten.Image) {
    text.Draw(screen, "Slideshow Paused", basicfont.Face7x13, 20, 30, color.White)
}

// drawDateOverlayRight places the photo timestamp in the bottom-right corner.
func drawDateOverlayRight(screen *ebiten.Image, takenTime time.Time) {
    face := basicfont.Face7x13
    sw, sh := screen.Size()
    dateStr := takenTime.Format("2006-01-02")

    // We can measure the text width to position it correctly.
    textBound := text.BoundString(face, dateStr)
    textWidth := textBound.Dx()

    x := sw - textWidth - 20
    y := sh - 20
    text.Draw(screen, dateStr, face, x, y, color.White)
}
