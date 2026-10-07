package engine

import "testing"

func TestCurrentShipImageCenterNativeOptional(t *testing.T) {
	data := originalWorldData(t, 1)
	cases := 0
	nativeCombatRows(t, "ship-image-center-trace.csv", func(v []int64) {
		w, err := NewWorld(data)
		if err != nil {
			t.Fatal(err)
		}
		w.Player.X, w.Player.Y = int(v[2]), int(v[3])
		if v[0] == 0 {
			w.Player.Inertia = int(v[1]) - 6
		} else {
			w.PlayerAlive = false
			w.PlayerSprite = w.commonAnimations["player-death"].Animation.Frames[v[1]].Sprite
		}
		context := w.weaponContext(Input{}, false)
		if context.ShipCenterX != int(v[4]) || context.ShipCenterY != int(v[5]) {
			t.Fatalf("current ship image center %v: %d/%d", v, context.ShipCenterX, context.ShipCenterY)
		}
		cases++
	})
	if cases != 66 {
		t.Fatalf("current image center coverage:%d", cases)
	}
}
