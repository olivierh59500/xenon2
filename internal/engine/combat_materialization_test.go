package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

func TestCombatMaterializationNativeTraceOptional(t *testing.T) {
	nativeCombatRows(t, "combat-materialization-trace.csv", func(v []int64) {
		image := visualassets.SpriteRegion{AnchorX: int(v[3]), AnchorY: int(v[4]), Width: int(v[5]), Height: int(v[6])}
		x, y, visible := MaterializeAttachment(int(v[1]), int(v[2]), image, int(v[7]), int(v[8]), int(v[0]))
		if x != int(v[9]) || y != int(v[10]) || visible != (v[11] != 0) {
			t.Fatalf("materialization %v: got %d,%d visible=%v", v, x, y, visible)
		}
	})
}
