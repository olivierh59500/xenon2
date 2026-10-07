package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func fourthContactFixture(t *testing.T, shield, y int, strong, protection, shades bool) (*World, *WorldActor) {
	t.Helper()
	w := testWorld(t)
	w.Level.Encounters = &visualassets.Encounters{}
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 2480, 2480, 2480
	w.Player.X, w.Player.Y = 100, y
	w.Equipment.Shield, w.Equipment.Protection = shield, protection
	if shades {
		w.Equipment.ShadesFrames = 10
	}
	w.MaterializationFrames, w.BackgroundStars = 0, nil
	w.Level.Ships = &visualassets.ShipArt{}
	for bank := 0; bank < 5; bank++ {
		name := []string{"player-ship-0", "player-ship-1", "player-ship-2", "player-ship-3", "player-ship-4"}[bank]
		w.Level.Ships.Atlas.Sprites = append(w.Level.Ships.Atlas.Sprites, visualassets.SpriteRegion{Name: name, Collision: &visualassets.CollisionBox{Width: 20, Height: 20}})
	}
	clip := visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "satellite"}}}
	art := &visualassets.GuardianGroup{ID: "middle-guardian", Components: make([]visualassets.GuardianComponent, 20), MotionParameters: map[string]int{"satellite_fire_rate": 0}}
	for i := range art.Components {
		art.Components[i].Health, art.Components[i].Animation = 20, clip
		art.Components[i].HeadingAnimations = make([]visualassets.ActorAnimation, 8)
		for heading := range 8 {
			art.Components[i].HeadingAnimations[heading] = clip
		}
	}
	state, err := NewFourthMiddleGuardian(art, w.ScrollY, nil)
	if err != nil {
		t.Fatal(err)
	}
	area := CollisionRect{Left: 110, Top: 20, Right: 140, Bottom: 68}
	state.Parts[16].Collision = area
	state.Parts[16].Arc.X, state.Parts[16].Arc.Y = 125<<16, 44<<16
	w.FourthMiddle, w.fourthMiddleArt = &state, art
	actor := &WorldActor{Active: true, Visible: true, ActorList: "moving", Atlas: "moving", Sprite: "satellite", X: 125, Y: 44, Health: 20, fourthIndex: 17, Collision: area,
		part: &visualassets.ActorPart{ResourceTag: 84, StrongHealth: strong, DamageMode: "fourth-guardian"}}
	w.fourthMiddleActors[16] = actor
	if err := w.bindWorldActor(actor); err != nil {
		t.Fatal(err)
	}
	w.Actors = []*WorldActor{actor}
	w.movingSpriteBoxes["satellite"] = visualassets.CollisionBox{X: -15, Y: -16, Width: 31, Height: 32}
	w.Level.Actors.Atlas.Sprites = append(w.Level.Actors.Atlas.Sprites, visualassets.SpriteRegion{Name: "satellite", Width: 32, Height: 32, AnchorX: 16, AnchorY: 16})
	for _, name := range []string{"explosion-small", "explosion-large"} {
		w.commonAnimations[name] = visualassets.NamedActorAnimation{Ending: "remove", Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: name, Duration: 4}}}}
	}
	w.SetRandomState(NewRandomState())
	return w, actor
}

func TestFourthSatellitePlayerContactKeepsArmoredColliderExtent(t *testing.T) {
	for _, fixture := range []struct {
		y, shield int
		shades    bool
		destroyed bool
	}{
		{1, 39, false, false},
		{32, 39, false, false},
		{68, 39, false, false},
		{32, 39, true, false},
		{32, 8, false, false},
	} {
		w, actor := fourthContactFixture(t, fixture.shield, fixture.y, false, false, fixture.shades)
		before := w.RandomState()
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
		if !actor.Active != fixture.destroyed || w.FourthMiddle.OuterTargets != 5-boolCount(fixture.destroyed) || w.Score != 300*boolCount(fixture.destroyed) {
			t.Fatalf("contact y%d shades%v shield%d lost its armored extent: alive%v outer%d score%d", fixture.y, fixture.shades, fixture.shield, actor.Active, w.FourthMiddle.OuterTargets, w.Score)
		}
		if w.RandomState() != before {
			t.Fatal("single satellite contact consumed unrelated gameplay randomness")
		}
		if fixture.shield == 8 && w.PlayerAlive {
			t.Fatal("lethal contact did not retain its original player-callback return")
		}
	}
	// The projectile route still admits a real interior hit. Ordinary ship
	// contact ignores the original guardian tag84; Shades invokes its callback
	// with the wider attacking collider rather than pretending to be a point.
	w, actor := fourthContactFixture(t, 39, 32, false, false, false)
	random := w.RandomState()
	if !w.weaponHitRect(CollisionRect{Left: 100, Top: 32, Right: 119, Bottom: 51}, 127, false) || actor.Active || w.Score != 300 || w.FourthMiddle.OuterTargets != 4 || w.RandomState() != random {
		t.Fatal("corrected player contact changed the original interior projectile hit")
	}
}

func TestFourthSatellitePlayerContactNativeCallbackOptional(t *testing.T) {
	comparisons := 0
	nativeCombatRows(t, "fourth-player-contact-trace.csv", func(v []int64) {
		w, actor := fourthContactFixture(t, int(v[0]), int(v[4]), v[1] != 0, v[2] != 0, v[3] != 0)
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
		effects := 0
		for _, entry := range w.Actors {
			if entry.Active && entry.ActorList == "transient" {
				effects++
			}
		}
		if w.Equipment.Shield != int(v[5]) || !w.PlayerAlive != (v[6] != 0) || w.FourthMiddle.Parts[16].Health != uint16(v[7]) || w.FourthMiddle.OuterTargets != int(v[8]) || w.Score != int(v[9]) || actor.Active != (v[10] != 0) || effects != int(v[11]) || w.RandomState().A != uint32(v[12]) || w.RandomState().B != uint32(v[13]) {
			t.Fatalf("original fourth contact differs%v: shield%d alive%v satellite%+v outer%d score%d effects%d random%+v", v, w.Equipment.Shield, w.PlayerAlive, w.FourthMiddle.Parts[16], w.FourthMiddle.OuterTargets, w.Score, effects, w.RandomState())
		}
		wantSound := ""
		switch v[14] {
		case 10:
			wantSound = "synthesized-effect-10"
		case 131:
			wantSound = "sampled-effect-03"
		case 133:
			wantSound = "sampled-effect-05"
		case 65535:
		default:
			t.Fatalf("unknown source contact sound%d", v[14])
		}
		if w.SoundRequests[2] != wantSound {
			t.Fatalf("source contact sound is%q, want%q", w.SoundRequests[2], wantSound)
		}
		comparisons++
	})
	if comparisons != 48 {
		t.Fatalf("incomplete original satellite contact comparison: %d", comparisons)
	}
}
