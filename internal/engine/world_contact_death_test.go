package engine

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"xenon2/internal/visualassets"
)

func contactDeathWorld(t *testing.T, shield int, heavy, protected, invulnerable, shades bool) (*World, *WorldActor) {
	t.Helper()
	w := testWorld(t)
	w.Level.Number = 2
	w.Level.Encounters = &visualassets.Encounters{}
	w.Level.Ships = &visualassets.ShipArt{Atlas: visualassets.SpriteAtlas{Sprites: []visualassets.SpriteRegion{{Name: "player-ship-2", Collision: &visualassets.CollisionBox{X: -2, Y: -2, Width: 5, Height: 5}}}}}
	w.Equipment.Shield, w.Equipment.Protection = shield, protected
	if invulnerable {
		w.InvulnerableFrames = 80
	}
	if shades {
		w.Equipment.ShadesFrames = 10
	}
	w.MaterializationFrames = 0
	w.movingSpriteBoxes["enemy"] = visualassets.CollisionBox{X: -4, Y: -4, Width: 9, Height: 9}
	for _, name := range []string{"explosion-small", "explosion-large", "cash-small", "cash-large"} {
		w.commonAnimations[name] = visualassets.NamedActorAnimation{Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: name, Duration: 0}}}}
	}
	a := &WorldActor{Active: true, Visible: true, ActorList: "moving", Atlas: "moving", X: 160, Y: 176, PreviousX: 160, PreviousY: 176, Health: 2, Score: 400, Sprite: "enemy", WaveToken: 0x123,
		part: &visualassets.ActorPart{ResourceTag: 200, StrongHealth: heavy, DamageMode: "individual", MotionMode: "follow-leader"}, animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "enemy", Duration: 0}}}}
	a.animationState = NewAnimation(a.animation)
	a.leader = a
	a.Collision = ActorCollisionRect(w.movingSpriteBoxes["enemy"], 160, 176)
	if err := w.bindWorldActor(a); err != nil {
		t.Fatal(err)
	}
	w.Actors = []*WorldActor{a}
	w.WaveBonuses.Entries[0] = WaveBonusEntry{ID: 0x123, Remaining: 1}
	return w, a
}

func TestLethalPlayerContactRetainsEnemyAndRewardState(t *testing.T) {
	w, enemy := contactDeathWorld(t, 16, true, false, false, false)
	beforeRandom := w.RandomState()
	beforePool := enemy.Binding.Slot
	for i := range w.shipTrail {
		w.shipTrail[i] = PlayerMotionState{X: 100 + i*10, Y: 150 + i}
	}
	trail := w.shipTrail
	if err := w.Step(Input{Motion: MotionInput{Right: true}}); err != nil {
		t.Fatal(err)
	}
	if w.PlayerAlive || !enemy.Active || enemy.Health != 2 || w.Score != 0 {
		t.Fatalf("lethal contact: playerAlive=%v enemyActive=%v health=%d score=%d collision=%+v", w.PlayerAlive, enemy.Active, enemy.Health, w.Score, w.playerCollision)
	}
	if w.WaveBonuses.Entries[0] != (WaveBonusEntry{ID: 0x123, Remaining: 1}) || w.RandomState() != beforeRandom || len(w.Collectibles) != 0 {
		t.Fatal("lethal contact consumed a wave reward or its shared random draw")
	}
	if enemy.Binding.Slot != beforePool || w.Pool.Slot(beforePool).ResourceTag != 200 {
		t.Fatal("source moving enemy did not retain its physical slot")
	}
	for _, actor := range w.Actors {
		if actor.ActorList == "transient" {
			t.Fatal("lethal contact created an enemy explosion after the player callback returned")
		}
	}
	if w.Player.ScrollStep != 1 || w.ScrollY != 4607 || w.Player.X != 160 || w.Player.Y != 176 || w.shipTrail != trail {
		t.Fatal("early player return moved the ship, shifted its history or reset the source scroll request")
	}
	if !enemy.Visible {
		t.Fatal("early player return incorrectly skipped the remaining actor callbacks")
	}
	if w.Shadows[0].Counter != 3 {
		t.Fatal("early player return skipped the separately owned shadow callback")
	}
}

func TestPlayerContactDeathNativeBranchOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("local source player-contact comparisons not supplied")
	}
	f, err := os.Open(filepath.Join(root, "player-contact-death-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows[1:] {
		n := func(i int) int {
			v, err := strconv.Atoi(row[i])
			if err != nil {
				t.Fatal(err)
			}
			return v
		}
		w, enemy := contactDeathWorld(t, n(0), n(1) != 0, n(2) != 0, n(3) != 0, n(4) != 0)
		before := w.RandomState()
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
		if w.Equipment.Shield != n(5) || w.PlayerAlive != (n(6) == 0) || enemy.Active != (n(7) == 0) {
			t.Fatalf("contact Go shield%d playerAlive%v enemyAlive%v native%v", w.Equipment.Shield, w.PlayerAlive, enemy.Active, row)
		}
		if n(7) == 0 && (w.Score != 0 || enemy.Health != 2 || w.RandomState() != before || w.WaveBonuses.Entries[0].Remaining != 1) {
			t.Fatalf("early source return changed reward state: %v", row)
		}
		if n(7) != 0 && (w.Score != 400 || w.WaveBonuses.Entries[0].ID != 0 || w.RandomState() == before) {
			t.Fatalf("live/shades contact lost its source enemy-reward branch: %v", row)
		}
	}
	if len(rows)-1 != 112 {
		t.Fatalf("incomplete contact comparisons%d", len(rows)-1)
	}
	t.Logf("Compared %d original contact death, protection, invulnerability and shades branches.", len(rows)-1)
}
