package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

// This isolated original-art fixture replays the native body controller into a
// reachable pose with hypothetical Side Shot equipment. It does not establish
// a carried campaign admission or retain earlier projectile birth events.
func fifthFinalSideAimFixture(t testing.TB, bodyY int) *World {
	t.Helper()
	data := originalWorldData(t, 5)
	e := NewEquipment()
	if !e.ApplyItem(ItemSideShot) {
		t.Fatal("cannot initialize isolated Side Shot capability")
	}
	data.InitialEquipment = &e
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	w.ScrollY, w.PreviousScrollY, w.RenderScrollY = 416, 416, 416
	w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = 0, 416, 416
	w.ScrollDelta = 1
	if err := w.activateFifthGuardian(visualassets.FixedEncounter{TriggerY: 416, Y: 416}, true); err != nil {
		t.Fatal(err)
	}
	for w.FifthFinal.Parts[0].Y < bodyY-2 {
		w.Frame++
		w.ScrollY--
		w.FifthFinal.Advance(w.fifthFinalArt, w.ScrollY, 1, 1, 416, int(w.Frame), 304, 176, &w.random)
	}
	for range 2 {
		w.Frame++
		w.ScrollY--
		w.advanceFifthGuardian(true)
	}
	w.Ready, w.MaterializationFrames = false, 0
	w.Player.X, w.Player.Y = 304, 176
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	w.cursor = RestartEncounterCursor(w.ScrollY)
	return w
}

func TestFifthGuardianAimRecognizesFirstSideVolleyWhilePrimaryMissesOptional(t *testing.T) {
	for _, test := range []struct {
		name          string
		bodyY         int
		maximumPasses int
	}{
		{"near defense", -120, 18},
		{"opposite flank", -16, 36},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := fifthFinalSideAimFixture(t, test.bodyY)
			before := forecastIsolationDigest(w)
			var native WorldForecast
			if err := native.Load(w); err != nil {
				t.Fatal(err)
			}
			var volleyFrame uint64
			var ids [2]int
			count, impactPass := 0, -1
			primaryUseful := false
			for pass := 0; pass < test.maximumPasses; pass++ {
				for range 3 {
					native.AdvancePALTick()
				}
				_, err := native.AdvanceObserved(Input{Fire: true}, func(e WeaponPointImpact) {
					if e.Kind != "small-shot" || e.ProjectileID <= w.nextActorID {
						return
					}
					_, useful := presentationFirstPointImpact(native.State(), e.X, e.Y)
					if e.OwnerSlot == 0 {
						primaryUseful = primaryUseful || useful
					}
					if e.OwnerSlot != 6 {
						return
					}
					if volleyFrame == 0 {
						volleyFrame = native.State().Frame
					}
					if native.State().Frame == volleyFrame && count < len(ids) {
						ids[count] = e.ProjectileID
						count++
					}
					for _, id := range ids[:count] {
						if id == e.ProjectileID && useful {
							impactPass = pass
						}
					}
				})
				if err != nil {
					t.Fatal(err)
				}
				if impactPass >= 0 {
					break
				}
			}
			damaged := false
			for index := 3; index < 21; index++ {
				damaged = damaged || native.State().FifthFinal.Parts[index].Health < w.FifthFinal.Parts[index].Health
			}
			if impactPass < 0 || !damaged || primaryUseful {
				t.Fatal("fixture did not execute a first-side-volley damage callback while primary misses")
			}
			if test.maximumPasses > 18 && impactPass < 18 {
				t.Fatal("opposite-flank fixture no longer crosses the original eighteen-pass horizon")
			}
			var aim WorldForecast
			useful, supported := presentationGuardianShotOpportunity(w, MotionInput{}, &aim, 3)
			if !supported || !useful {
				t.Fatal("actual first side volley did not justify the guardian trigger")
			}
			if forecastIsolationDigest(w) != before {
				t.Fatal("side-weapon aiming modified the live source")
			}
		})
	}
}

func TestFifthGuardianAimDoesNotCreditLaterSideVolleysOptional(t *testing.T) {
	w := fifthFinalSideAimFixture(t, -145)
	before := forecastIsolationDigest(w)
	var aim WorldForecast
	useful, supported := presentationGuardianShotOpportunity(w, MotionInput{}, &aim, 3)
	if !supported || useful {
		t.Fatal("a later side volley was credited to the current trigger")
	}
	laterDamage := false
	for index := 3; index < 21; index++ {
		laterDamage = laterDamage || aim.State().FifthFinal.Parts[index].Health < w.FifthFinal.Parts[index].Health
	}
	if !laterDamage {
		t.Fatal("fixture no longer includes the later volley that must not justify the first trigger")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("rejected side volley modified live state")
	}
}

func BenchmarkFifthGuardianSideAim(b *testing.B) {
	for _, fixture := range []struct {
		name   string
		bodyY  int
		useful bool
	}{
		{"near", -120, true}, {"opposite", -16, true}, {"later volley", -145, false},
	} {
		b.Run(fixture.name, func(b *testing.B) {
			w := fifthFinalSideAimFixture(b, fixture.bodyY)
			var forecast WorldForecast
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				useful, supported := presentationGuardianShotOpportunity(w, MotionInput{}, &forecast, 3)
				if !supported || useful != fixture.useful {
					b.Fatal("side volley benchmark changed its native callback outcome")
				}
			}
		})
	}
}
