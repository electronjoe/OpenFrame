package slideshow

import (
    "image/color"
    "time"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/ebitenutil"
    "github.com/hajimehoshi/ebiten/v2/text"
    "golang.org/x/image/font/basicfont"
)

// drawDebugString just prints text in the top-left corner of the screen.
func drawDebugString(screen *ebiten.Image, msg string) {
    screen.Fill(color.RGBA{0, 0, 0, 255}) // Clear to black
    ebitenutil.DebugPrint(screen, msg)
}

// drawTiledImage handles the logic of centering/scaling the entire TiledImage on screen.
func drawTiledImage(screen *ebiten.Image, t *TiledImage) {
    screen.Fill(color.RGBA{0, 0, 0, 255}) // Clear to black

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
            op.GeoM.Translate(-float64(maxTileSize)/2, -float64(maxTileSize)/2)
            op.GeoM.Scale(scale, scale)
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

// drawDateOverlay renders the given timestamp in the bottom-left corner.
func drawDateOverlay(screen *ebiten.Image, takenTime time.Time) {
    face := basicfont.Face7x13
    _, sh := screen.Size()
    dateStr := takenTime.Format("2006-01-02 15:04:05")
    text.Draw(screen, dateStr, face, 20, sh-20, color.White)
}
