package engine

import "testing"

func newFixedSpriteDamageReferenceWorld(t *testing.T, data LevelData, record, profile, amount, frame int) (*World, int, int) {
	t.Helper()
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	first := w.Pool.FreeFirst()
	for slot := first; slot != NoActorSlot; slot = w.Pool.Slot(slot).freeNext {
		w.Pool.Slot(slot).Residue = waveConstructorResidueFixture(slot)
	}
	w.WaveBonuses.BeginPass(0)
	r := data.Encounters.Fixed[record]
	w.ScrollY, w.MaximumScrollY, w.ScrollDelta = r.Y-108, 4607, 0
	w.Player.X, w.Player.Y = 160, 176
	w.spawnFixed(r)
	actor := w.Actors[0]
	if actor.fixedKind == nil || actor.fixedKind.Behavior == "fourth-crawler" {
		t.Fatal("original fixed record did not select a common sprite constructor")
	}
	target := actor.Binding.Slot
	for pass := range frame {
		w.Frame = uint64(pass)
		w.advanceFixedSprite(actor)
		w.finishActorUpdate(actor)
		if !actor.Active {
			t.Fatalf("original fixed sprite expired before damage at pass %d", pass)
		}
	}
	for index := 0; profile != 0 && w.Pool.FreeFirst() != NoActorSlot; index++ {
		if profile == 1 {
			w.spawnEnemyShot(80+index%20*8, 48+index%12*8, EnemyShot{Direction: uint8(index & 7), Speed: 4 + index%5})
		} else if _, err := w.reserveWorldActor(200, ActorPoolMoving, true); err != nil {
			t.Fatal(err)
		}
	}
	w.Score, w.SoundRequests = 0, [4]string{}
	w.damageActor(actor, uint16(amount))
	if w.poolError != nil {
		t.Fatal(w.poolError)
	}
	return w, first, target
}

func TestEveryCommonFixedSpriteDamageUnderPressureNativeTraceOptional(t *testing.T) {
	data, _, _ := nativeWaveReferenceData(t)
	var w *World
	previous := [5]int{-1, -1, -1, -1, -1}
	seen := make(map[[5]int]bool)
	nextSlot, cases, rows := ActorPoolCapacity, 0, 0
	var levels [5]int
	nativeCombatRows(t, "fixed-sprite-damage.csv", func(v []int64) {
		if len(v) != 60 {
			t.Fatalf("fixed-sprite damage trace has %d columns, want 60", len(v))
		}
		key := [5]int{int(v[0]), int(v[1]), int(v[2]), int(v[3]), int(v[58])}
		if key != previous {
			if nextSlot != ActorPoolCapacity || seen[key] || key[0] < 1 || key[0] > 5 || key[1] < 0 || key[1] >= len(data[key[0]-1].Encounters.Fixed) || key[2] < 0 || key[2] > 2 || key[3] != 1 && key[3] != 127 || key[4] != 0 && key[4] != 1 && key[4] != 16 && key[4] != 64 {
				t.Fatalf("invalid or incomplete fixed-sprite damage case %v after %v", key, previous)
			}
			seen[key] = true
			var first, target int
			w, first, target = newFixedSpriteDamageReferenceWorld(t, data[key[0]-1], key[1], key[2], key[3], key[4])
			if target != int(v[4]) || first != int(v[6]) || w.Pool.FreeFirst() != int(v[7]) || w.ScrollY != int(v[59]) {
				t.Fatalf("fixed-sprite case %v target/free/scroll differ: target %d free %d scroll %d original %v", key, target, w.Pool.FreeFirst(), w.ScrollY, v)
			}
			if w.random != (RandomState{A: uint32(v[34]), B: uint32(v[35])}) || w.Score != int(v[36]) || w.SoundRequests[1] != nativeRewardSound(v[37]) || w.SoundRequests[2] != nativeRewardSound(v[38]) || w.WaveBonuses.NextID != uint16(v[39]) || w.WaveBonuses.NormalID != uint16(v[40]) || w.WaveBonuses.HeavyID != uint16(v[41]) {
				t.Fatalf("fixed-sprite case %v RNG %+v score %d sound %v cache %+v, original %v", key, w.random, w.Score, w.SoundRequests, w.WaveBonuses, v[34:42])
			}
			for index, bucket := range w.WaveBonuses.Entries {
				if bucket != (WaveBonusEntry{ID: uint16(v[42+index*2]), Remaining: uint16(v[43+index*2])}) {
					t.Fatalf("fixed-sprite case %v original reward bucket %d differs", key, index)
				}
			}
			previous, nextSlot = key, first
			cases++
			levels[key[0]-1]++
		}
		if int(v[5]) != nextSlot {
			t.Fatalf("fixed-sprite case %v skipped slot %d for %d", key, nextSlot, v[5])
		}
		nextSlot++
		compareDamageNativePoolRow(t, key, w, v)
		rows++
	})
	if cases != 1032 || rows != 157824 || nextSlot != ActorPoolCapacity || levels != [5]int{264, 96, 456, 0, 216} {
		t.Fatalf("incomplete fixed-sprite callback coverage: cases %d rows %d levels %v", cases, rows, levels)
	}
	t.Logf("Compared %d fixed-sprite damage cases and %d physical-slot states", cases, rows)
}

func TestFifthPracticeRejectsOutcomeChangedByCorrectedProjectileResidueOptional(t *testing.T) {
	w := fifthPracticeSourceFixture(t)
	p := PresentationPilot{PALRefreshes: 3}
	for index := range 1149 {
		input, ok := p.fifthPracticedOpeningInput(w)
		if !ok || input != fifthPracticeControl(fifthOpeningControls[index]) {
			t.Fatalf("unchanged practice prefix rejected at %d", index)
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(input); err != nil {
			t.Fatal(err)
		}
	}
	before := forecastIsolationDigest(w)
	// Preserve the obsolete post-pass outcome independently of the renewed
	// recording. Its delayed health pickup must still fail the same validator.
	const obsoleteHealthOutcome uint64 = 0x3a4f9835846f45c2
	input := fifthPracticeControl(fifthOpeningControls[1149])
	stale := fifthOpeningPractice{world: w}
	if stale.acceptsNext(w, input, obsoleteHealthOutcome, false) {
		t.Fatal("obsolete fifth-stage outcome admitted after corrected projectile publication")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("rejecting the obsolete recording changed the live game")
	}
	actual := stale.forecast.State()
	if actual.Frame != 1151 || actual.Equipment.Shield != 39 || fifthPracticeMarker(actual) == obsoleteHealthOutcome || fifthPracticeMarker(actual) != fifthOpeningMarkers[1150] || w.Frame != 1150 || w.Equipment.Shield != 23 {
		t.Fatal("the real earlier health collection no longer distinguishes the obsolete outcome")
	}
	if renewed, ok := p.fifthPracticedOpeningInput(w); !ok || renewed != input || p.fifthPractice == nil {
		t.Fatal("renewed ordinary control did not admit the corrected outcome")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("renewed validation changed the live game")
	}
}
