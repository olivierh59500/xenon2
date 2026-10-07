package engine

import "testing"

func TestPresentationRecognizesOriginalThirdTerrainCannonOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	w.ScrollY, w.MaximumScrollY = 1600, 2608
	w.Player.X, w.Player.Y = 256, 176
	for _, record := range w.Level.Encounters.Fixed {
		if record.EnemyKind == 5 && record.X == 232 && record.Y == 1704 {
			w.spawnThirdFixed(record)
			break
		}
	}
	cannon := w.Actors[0]
	if cannon.thirdCannon == nil {
		t.Fatal("original blocking cannon was not constructed")
	}
	w.advanceThirdCannon(cannon)
	if cannon.Visible || cannon.Collision.Empty() {
		t.Fatal("original cannon must expose its collider through terrain drawing")
	}
	player, random, health := w.Player, w.RandomState(), cannon.Health
	if !presentationShotOpportunity(w) {
		t.Fatal("ordinary cannon firing opportunity was missed")
	}
	if w.Player != player || w.RandomState() != random || cannon.Health != health {
		t.Fatal("target recognition changed live game state")
	}
	shot := &WorldSmallShot{Active: true, Shot: SmallShot{X: 256, Y: cannon.Collision.Top + 9, VelocityY: -9, Damage: 1}}
	w.advanceSmallShot(shot)
	if shot.Active || cannon.Health != health-1 {
		t.Fatal("recognized cannon did not receive its ordinary projectile callback")
	}
	w.Player.X = 64
	if presentationShotOpportunity(w) {
		t.Fatal("off-axis cannon justified an unnecessary forward shot")
	}
}
