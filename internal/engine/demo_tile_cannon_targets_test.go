package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func TestTerrainCannonWithoutSpriteRemainsARealAimTarget(t *testing.T) {
	w := testWorld(t)
	w.Player.X, w.Player.Y = 160, 150
	a := &WorldActor{Active: true, ActorList: "moving", X: 144, Y: 60, PreviousX: 144, PreviousY: 60, Health: 2,
		fixedTileState: &FixedTileState{Kind: 1, X: 144, WorldY: w.ScrollY + 60}, fixedTileArt: &visualassets.FixedTileKind{Kind: 1},
		part: &visualassets.ActorPart{DamageMode: "fixed-tile"}, Collision: CollisionRect{Left: 154, Top: 64, Right: 172, Bottom: 88}}
	w.Actors = []*WorldActor{a}
	before := forecastDigest(w)
	if a.Visible || a.Sprite != "" {
		t.Fatal("terrain-cannon fixture must use only its terrain artwork")
	}
	if bounds, ok := presentationTargetBounds(w, a); !ok || bounds != a.Collision || !presentationShotOpportunity(w) {
		t.Fatal("damageable terrain cannon was excluded because it has no sprite")
	}
	if forecastDigest(w) != before {
		t.Fatal("target observation changed the live terrain cannon")
	}
	a.Active = false
	if _, ok := presentationTargetBounds(w, a); ok {
		t.Fatal("destroyed terrain cannon remained an aim target")
	}
}

func TestCentralBonusRemainsReachableFromBottomFlightLane(t *testing.T) {
	w := testWorld(t)
	w.Player.X, w.Player.Y = 160, 176
	w.Equipment.SpeedTier = 2
	bonus := &WorldCollectible{ID: 1, Active: true, X: 160, Y: 100, Motion: CashMotion{X: 160, Y: 100, Mode: 7}}
	w.Collectibles = []*WorldCollectible{bonus}
	goal := presentationChooseGoal(w)
	if goal.bonus != bonus || !goal.valid(w) {
		t.Fatal("central reward was excluded solely because the ship was near the bottom")
	}
}

func TestLinkedBodyForwardingDamageRemainsAnAimTarget(t *testing.T) {
	w := testWorld(t)
	w.Player.X, w.Player.Y = 160, 150
	head := presentationTestEnemy(1, 230, 70)
	head.part.DamageMode = "group"
	body := presentationTestEnemy(2, 160, 70)
	body.leader, body.part.Linked, body.part.DamageMode = head, true, "group"
	w.Actors = []*WorldActor{head, body}
	if _, ok := presentationTargetBounds(w, body); !ok || !presentationShotOpportunity(w) {
		t.Fatal("a linked body forwarding real damage to its leader was excluded from aiming")
	}
	if !w.weaponHitPoint(160, 70, 1) || head.Health != 2 {
		t.Fatal("fixture body did not forward an ordinary point hit to its real leader")
	}
	body.part.DamageMode = "block-shot"
	if _, ok := presentationTargetBounds(w, body); ok {
		t.Fatal("immune linked body became an aim target")
	}
}
