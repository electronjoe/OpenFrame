package slideshow

import (
    "image/color"
    "math"
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

// drawPauseIndicator places Pause notification text at top left of the screen.
func drawPauseIndicator(screen *ebiten.Image) {
    text.Draw(screen, "Slideshow Paused", basicfont.Face7x13, 20, 30, color.White)
}

// drawDateOverlayLeft rotates the date 90° CCW and places it near the bottom-left edge.
func drawDateOverlayLeft(screen *ebiten.Image, takenTime time.Time) {
    dateStr := takenTime.Format("2006-01-02")
    drawVerticalText(screen, dateStr, true)
}

// drawDateOverlayRight rotates the date 90° CCW and places it near the bottom-right edge.
func drawDateOverlayRight(screen *ebiten.Image, takenTime time.Time) {
    dateStr := takenTime.Format("2006-01-02")
    drawVerticalText(screen, dateStr, false)
}

// drawVerticalText creates a small offscreen image of the date text, then rotates it 90° CCW
// and draws it at the screen edge (left if `isLeftEdge`, right otherwise).
func drawVerticalText(screen *ebiten.Image, textStr string, isLeftEdge bool) {
    face := basicfont.Face7x13

    // Measure the text in its normal orientation.
    bounds := text.BoundString(face, textStr)
    textWidth := bounds.Dx()
    textHeight := bounds.Dy()

    // Create an offscreen image big enough for the text.
    textImg := ebiten.NewImage(textWidth, textHeight)
    // Optional: fill a semi-transparent background if desired:
    // textImg.Fill(color.RGBA{0, 0, 0, 128})

    // Draw the text in normal (horizontal) orientation at top-left of the offscreen.
    // We typically draw so the text baseline is near the bottom of that offscreen rect:
    text.Draw(textImg, textStr, face, 0, textHeight-2, color.White)

    // Now we set up our transformation to rotate 90° CCW.
    // 90° CCW is -π/2 radians.
    op := &ebiten.DrawImageOptions{}

    // First, translate so the image center is at the origin (0,0).
    op.GeoM.Translate(-float64(textWidth)/2, -float64(textHeight)/2)

    // Rotate 90° counter-clockwise.
    op.GeoM.Rotate(-math.Pi / 2)

    // We’ll place the resulting, rotated image along the appropriate screen edge.
    screenW, screenH := screen.Size()
    margin := 20.0

    // After rotation:
    // - The "width" of the text (in the new orientation) will be textHeight.
    // - The "height" of the text (in the new orientation) will be textWidth.

    if isLeftEdge {
        // For the left edge, x is margin + half of new image width,
        // so that the left side lines up near margin. We want the bottom of text near screen bottom.
        finalX := margin + float64(textHeight)/2
        finalY := float64(screenH) - margin - float64(textWidth)/2

        op.GeoM.Translate(finalX, finalY)
    } else {
        // For the right edge, x is (screenW - margin - half of new image width).
        finalX := float64(screenW) - margin - float64(textHeight)/2
        finalY := float64(screenH) - margin - float64(textWidth)/2

        op.GeoM.Translate(finalX, finalY)
    }

    // Finally, draw the rotated text onto the main screen.
    screen.DrawImage(textImg, op)
}
