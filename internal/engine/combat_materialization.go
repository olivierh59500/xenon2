package engine

import "xenon2/internal/visualassets"

// MaterializeAttachment applies the original signed-word expansion around the
// current ship image center. The sixteenth phase hides the attachment.
func MaterializeAttachment(x, y int, image visualassets.SpriteRegion, shipCenterX, shipCenterY, phase int) (int, int, bool) {
	if phase == 16 {
		return x, y, false
	}
	if phase == 0 {
		return x, y, true
	}
	center := func(position, anchor, size int) int {
		v := int16(position) - int16(anchor)
		v = v*2 + int16(size)
		return int(v >> 1)
	}
	cx := center(x, image.AnchorX, image.Width)
	cy := center(y, image.AnchorY, image.Height-1)
	return int(int16(shipCenterX + int(int16(cx-shipCenterX))*phase)), int(int16(shipCenterY + int(int16(cy-shipCenterY))*phase)), true
}
