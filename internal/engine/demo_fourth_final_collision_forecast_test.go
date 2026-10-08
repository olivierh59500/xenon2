package engine

import "testing"

// Arranged original final arena, not an earned campaign state. Native factories,
// animation prefixes, full eye/core health and ordinary callbacks are untouched.
func fourthFinalOrderScene(t *testing.T, shotY int) *World {
	t.Helper()
	equipment := NewEquipment()
	equipment.ApplyItem(ItemSpeedup)
	equipment.ApplyItem(ItemSpeedup)
	data := playableOriginalWorldData(t, 4)
	data.InitialEquipment = &equipment
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	w.Frame = 6661
	w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = 42, 0, 58, 58
	w.ScrollDelta = 1
	w.cursor = EncounterCursor{MovingHighWater: 1, FixedHighWater: 0}
	w.Player = PlayerMotionState{X: 134, Y: 170, Inertia: -6, SpeedTier: 2, ScrollStep: 1}
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	if err = w.activateFourthGuardian(true); err != nil {
		t.Fatal(err)
	}
	if w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
		t.Fatal("arranged source pose is covered")
	}
	w.spawnFourthShot(FourthGuardianShot{X: 128, Y: shotY, Direction: 4, Speed: w.fourthFinalArt.MotionParameters["eye_shot_speed"], Animation: "eye-shot-4"}, w.fourthFinalArt)
	return w
}

func TestFourthFinalNativeClipUsesCachedPlayerPrefixOptional(t *testing.T) {
	w := fourthFinalOrderScene(t, 144)
	before := forecastIsolationDigest(w)
	shot := w.Projectiles[0]
	motion := MotionInput{Left: true}
	player := w.Player
	player.Advance(motion, MotionContext{ScrollY: w.ScrollY, VisitedScrollY: w.MaximumScrollY, BaseScrollStep: w.BaseScrollStep})
	rect := thirdMiddlePlayerBounds(w, player)
	x, y, active := demoProjectilePosition(w, shot, 1, w.ScrollDelta)
	oldRisk := active && x >= rect.Left-5 && x <= rect.Right+5 && y >= rect.Top-5 && y <= rect.Bottom+5
	if oldRisk {
		t.Fatal("old center-plus-five counterexample disappeared")
	}
	var native WorldForecast
	if err := native.Load(w); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		native.AdvancePALTick()
	}
	result, err := native.Advance(Input{Motion: motion})
	if err != nil {
		t.Fatal(err)
	}
	if result.Shield != w.Equipment.Shield-4 || !result.Alive {
		t.Fatal("native fullclip did not damage cached ship prefix")
	}
	q := native.State().Projectiles
	t.Logf("NATIVE_CLIP sourcePrefix%+v movedPrefix%+v predictedCenter%d,%d oldRisk%v shield%d->%d sprite%s sourceBox%+v remainingShots%d", thirdMiddlePlayerBounds(w, w.Player), rect, x, y, oldRisk, w.Equipment.Shield, result.Shield, shot.Sprite, w.movingSpriteBoxes[shot.Sprite], len(q))
	if forecastIsolationDigest(w) != before {
		t.Fatal("native clone changed source")
	}
}

func TestFourthFinalAnalyticalMotionMissesOriginalProjectileClipOptional(t *testing.T) {
	w := fourthFinalOrderScene(t, 115)
	before := forecastIsolationDigest(w)
	pilot := DemoPilot{}
	input, handled := pilot.FourthFinalInput(w)
	if !handled {
		t.Fatal("original final not admitted")
	}
	var old, right WorldForecast
	if err := old.Load(w); err != nil {
		t.Fatal(err)
	}
	if err := right.Load(w); err != nil {
		t.Fatal(err)
	}
	oldMin, rightMin := w.Equipment.Shield, w.Equipment.Shield
	for range 6 {
		for range 3 {
			old.AdvancePALTick()
			right.AdvancePALTick()
		}
		a, err := old.Advance(input)
		if err != nil {
			t.Fatal(err)
		}
		b, err := right.Advance(Input{Motion: MotionInput{Right: true}, Fire: input.Fire})
		if err != nil {
			t.Fatal(err)
		}
		oldMin, rightMin = min(oldMin, a.Shield), min(rightMin, b.Shield)
	}
	t.Logf("SOURCE_POLICY_CLIP oldInput%+v oldMin%d rightMin%d oldEnd%+v rightEnd%+v", input, oldMin, rightMin, old.State().Player, right.State().Player)
	if oldMin >= w.Equipment.Shield || rightMin != w.Equipment.Shield || fourthAdmissionTerrainUnsafe(right.State()) {
		t.Fatal("one native safe motion did not separate old approximate risk")
	}
	if w.FourthFinal.Parts[0].Health != 100 || w.FourthFinal.Parts[1].Health != 50 || w.FourthFinal.Parts[2].Health != 50 || forecastIsolationDigest(w) != before {
		t.Fatal("read-only policy/forecast changed native source gates")
	}
}
