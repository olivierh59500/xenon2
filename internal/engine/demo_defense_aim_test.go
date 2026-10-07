package engine

import "testing"

// The node is drawn through mutable terrain, so its normal actor renderer is
// invisible even while its source collision/damage callback is open.
func TestPresentationRecognizesOriginalTerrainDefenseNodeOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 2))
	if err != nil {
		t.Fatal(err)
	}
	w.Ready, w.MaterializationFrames = false, 0
	w.ScrollY, w.ScrollDelta = 2560, 0
	w.Player.X, w.Player.Y = 232, 100
	w.secondScheduler.DefenseFlags = 2
	node := w.secondNodes[0]
	w.advanceSecondNode(node)
	if node.Visible || node.Collision.Empty() || node.secondNode.Health == 0 {
		t.Fatal("original open node fixture is not terrain-only")
	}
	player, random, health := w.Player, w.RandomState(), node.Health
	if !presentationShotOpportunity(w) {
		t.Fatal("open terrain node was rejected because its actor renderer is invisible")
	}
	if w.Player != player || w.RandomState() != random || node.Health != health {
		t.Fatal("target recognition changed original game state")
	}
	shot := &WorldSmallShot{Active: true, Shot: SmallShot{X: 232, Y: 73, VelocityY: -9, Damage: 1}}
	w.advanceSmallShot(shot)
	if shot.Active || node.Health != health-1 {
		t.Fatal("recognized node did not receive its real ordinary projectile callback")
	}
	w.secondScheduler.DefenseFlags = 3
	w.advanceSecondNode(node)
	if presentationShotOpportunity(w) {
		t.Fatal("shielded original node justified an otherwise empty firing command")
	}
}
