package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

func attachmentMidpoint(previous, current float64) float64 { return (previous + current) / 2 }

func TestFirstGuardianEyeRetainsRigidInterpolatedOffset(t *testing.T) {
	w := testWorld(t)
	w.ScrollY = 0
	state := NewFirstGuardianState(20)
	state.Active, state.ExtensionPhase, state.BodyWorldY, state.BodyVelocity, state.BodyTimer = true, 1, 32, 1, 100
	w.FirstGuardian = &state
	w.firstGuardianArt = &visualassets.GuardianVisual{BodyX: 112, EyeX: 152, EyeOffsetY: 66, EyeFrames: []string{"eye"}}
	w.firstGuardianActor = &WorldActor{X: 112, Y: 32, Active: true, Visible: true}
	w.advanceFirstGuardian()
	body := w.firstGuardianActor
	if len(body.Extras) != 1 {
		t.Fatal("eye attachment missing")
	}
	eye := body.Extras[0]
	if !eye.Interpolate || body.PreviousY != 32 || eye.PreviousY != 98 || eye.Y != 99 {
		t.Fatalf("body/eye endpoints differ: body%+v eye%+v", body, eye)
	}
	if attachmentMidpoint(eye.PreviousY, eye.Y)-attachmentMidpoint(body.PreviousY, body.Y) != 66 {
		t.Fatal("intermediate eye detached from guardian body")
	}
}

func TestFifthMouthUsesCurrentPoseWithParentTranslation(t *testing.T) {
	for _, clock := range []int{4, -4} {
		w := testWorld(t)
		overlay := visualassets.GuardianOverlay{ID: "mouth", OffsetX: 105, PositiveOffsetY: 85, NegativeOffsetY: 229, CounterStride: 4, Frames: []string{"closed", "open"}}
		actor := &WorldActor{X: 50, Y: 60, PreviousX: 48, PreviousY: 57, fifthPart: &visualassets.GuardianComponent{Overlays: []visualassets.GuardianOverlay{overlay}}}
		state := &FifthGuardianPartState{X: 50, Y: 60, Clock: clock}
		w.setFifthMouthOverlays(actor, state)
		if len(actor.Extras) != 1 {
			t.Fatal("mouth attachment missing")
		}
		mouth := actor.Extras[0]
		y := overlay.PositiveOffsetY
		if clock < 0 {
			y = overlay.NegativeOffsetY
		}
		if mouth.Sprite != "open" || !mouth.Interpolate || mouth.X != 155 || mouth.Y != float64(60+y) {
			t.Fatalf("current source mouth pose changed: %+v", mouth)
		}
		if attachmentMidpoint(mouth.PreviousX, mouth.X)-attachmentMidpoint(actor.PreviousX, actor.X) != 105 || attachmentMidpoint(mouth.PreviousY, mouth.Y)-attachmentMidpoint(actor.PreviousY, actor.Y) != float64(y) {
			t.Fatal("mouth pose offset interpolated independently from the body")
		}
	}
}

func TestExtendingBeamInterpolatesItsOwnEndpointsAndKeepsDiscreteWidth(t *testing.T) {
	for variant := 0; variant < 2; variant++ {
		w := testWorld(t)
		cover := visualassets.TilePatch{Columns: 1, Rows: 1, Tiles: []uint16{1}}
		clip := visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "tip"}}}
		kind := visualassets.FixedSpriteKind{Behavior: "extending-beam", Variants: []visualassets.FixedSpriteVariant{{ID: 0, Animation: clip, Cover: &cover}, {ID: 1, Animation: clip, Cover: &cover}}}
		actor := &WorldActor{fixedKind: &kind, fixedState: FixedSpriteState{X: 104, Y: 40, Phase: 2, Variant: variant}}
		w.composeFixedSprite(actor)
		previous := FixedSpriteBeamLayout(actor.fixedState)
		actor.fixedState.Phase, actor.fixedState.Y = 3, 39
		w.composeFixedSprite(actor)
		current := FixedSpriteBeamLayout(actor.fixedState)
		tip := actor.Extras[0]
		if !tip.Interpolate || tip.PreviousX != float64(previous.TipX) || tip.PreviousY != float64(previous.TipY) || tip.X != float64(current.TipX) || tip.Y != float64(current.TipY) {
			t.Fatalf("beam tip history lost its extension endpoint: %+v", tip)
		}
		if attachmentMidpoint(tip.PreviousX, tip.X) == tip.X {
			t.Fatal("beam tip still jumps directly to its next extension")
		}
		if len(actor.TileOverlays) != current.ShaftColumns {
			t.Fatal("shaft width no longer uses the current discrete source pose")
		}
		for column, tile := range actor.TileOverlays {
			if !tile.Interpolate || tile.PreviousX != float64(previous.ShaftX+column*16) || tile.X != float64(current.ShaftX+column*16) || tile.PreviousY != 40 || tile.Y != 39 {
				t.Fatalf("beam shaft endpoint differs: %+v", tile)
			}
		}
	}
}

func TestFourthCrawlerCoverInterpolatesCameraAndIgnoresCrawlerMotion(t *testing.T) {
	w := testWorld(t)
	cover := visualassets.TilePatch{Columns: 1, Rows: 3, Tiles: []uint16{1, 2, 3}}
	kind := visualassets.FixedSpriteKind{MotionTables: map[string][]int{"cover_x": {40}, "cover_world_y": {1000}}, Variants: []visualassets.FixedSpriteVariant{{ID: 0, Cover: &cover}}}
	actor := &WorldActor{X: 40, Y: 100, fourthCrawler: &FourthCrawlerState{X: 40, Y: 1000, Nest: 0, Visible: true, Phase: 1}, fourthCrawlerArt: &kind}
	w.ScrollY = 900
	w.updateFourthCrawlerCover(actor)
	w.ScrollY = 901
	actor.X, actor.Y = 60, 105
	actor.fourthCrawler.X, actor.fourthCrawler.Y = 60, 1006
	w.updateFourthCrawlerCover(actor)
	tile := actor.TileOverlays[0]
	if !tile.Interpolate || tile.X != 40 || tile.PreviousX != 40 || tile.Y != 99 || tile.PreviousY != 100 {
		t.Fatalf("fixed nest cover followed the moving crawler: %+v", tile)
	}
	if attachmentMidpoint(tile.PreviousY, tile.Y) != 99.5 {
		t.Fatal("nest cover did not retain camera interpolation")
	}
	if (WorldSpriteAttachment{X: 10, Y: 160}).Interpolate {
		t.Fatal("screen-fixed HUD defaults changed")
	}
}

func TestHiddenFirstGuardianStartsAtCurrentDisplayEndpoint(t *testing.T) {
	w := testWorld(t)
	w.ScrollY = 0
	state := NewFirstGuardianState(20)
	state.Active = true
	w.FirstGuardian = &state
	w.firstGuardianArt = &visualassets.GuardianVisual{BodyX: 112, EyeX: 152, EyeOffsetY: 66, EyeFrames: []string{"eye"}}
	w.firstGuardianActor = &WorldActor{X: 0, Y: -4096, Active: true}
	w.advanceFirstGuardian()
	body := w.firstGuardianActor
	eye := body.Extras[0]
	if body.PreviousX != body.X || body.PreviousY != body.Y || eye.PreviousX != eye.X || eye.PreviousY != eye.Y {
		t.Fatal("newly visible guardian interpolates from hidden coordinates")
	}
}

func TestRetainedFifthMouthSnapsAfterCheckpointCameraShiftOptional(t *testing.T) {
	w := fifthResourceWorld(t)
	if err := w.activateFifthGuardian(visualassets.FixedEncounter{}, true); err != nil {
		t.Fatal(err)
	}
	body := w.fifthFinalActors[0]
	state := &w.FifthFinal.Parts[0]
	state.Clock = 8
	body.Extras = nil
	w.setFifthMouthOverlays(body, state)
	if len(body.Extras) == 0 {
		t.Fatal("source mouth pose missing")
	}
	oldY := body.Extras[0].Y
	w.prepareCheckpointActors(64, false)
	if !body.snapDisplayHistory || body.Extras[0].Y != oldY+64 || body.Extras[0].PreviousY != body.Extras[0].Y {
		t.Fatal("retained mouth did not follow checkpoint camera admission")
	}
	// The first resumed callback may recompute the parent's scene position.
	body.X, body.Y = body.X+3, body.Y-1
	body.Extras[0].X, body.Extras[0].Y = body.Extras[0].X+3, body.Extras[0].Y-1
	w.finishActorUpdate(body)
	if body.snapDisplayHistory || body.PreviousX != body.X || body.PreviousY != body.Y || body.Extras[0].PreviousX != body.Extras[0].X || body.Extras[0].PreviousY != body.Extras[0].Y {
		t.Fatal("first resumed assembly still interpolates across the camera reset")
	}
}
