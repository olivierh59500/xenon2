package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func originalGuardianTargetWorld(t testing.TB, level int, final bool) *World {
	t.Helper()
	w, err := NewWorld(originalWorldData(t, level))
	if err != nil {
		t.Fatal(err)
	}
	if level == 4 {
		if final {
			w.ScrollY, w.MinimumScrollY, w.MaximumScrollY = 0, 0, 16
		}
		if err := w.activateFourthGuardian(final); err != nil {
			t.Fatal(err)
		}
		actors := w.fourthMiddleActors[:]
		if final {
			actors = w.fourthFinalActors[:]
		}
		for _, actor := range actors {
			if err := w.advanceFourthPart(actor); err != nil {
				t.Fatal(err)
			}
		}
	} else {
		w.ScrollY = 2336
		if err := w.activateFifthGuardian(visualassets.FixedEncounter{Y: 2336}, final); err != nil {
			t.Fatal(err)
		}
		w.advanceFifthGuardian(final)
	}
	return w
}

func targetPoint(bounds CollisionRect) CollisionRect {
	x, y := (bounds.Left+bounds.Right)/2, (bounds.Top+bounds.Bottom)/2
	return CollisionRect{Left: x, Right: x, Top: y, Bottom: y}
}

func TestPresentationFourthGuardianTargetsMatchSourceDamageOptional(t *testing.T) {
	w := originalGuardianTargetWorld(t, 4, false)
	before := forecastDigest(w)
	for index, actor := range w.fourthMiddleActors {
		bounds, target := presentationTargetBounds(w, actor)
		copy := *w.FourthMiddle
		copy.Strike(index, targetPoint(actor.Collision), 1)
		damaged := false
		for part := range copy.Parts {
			damaged = damaged || copy.Parts[part].Health != w.FourthMiddle.Parts[part].Health
		}
		if target != damaged {
			t.Fatalf("middle role%d target%v disagrees with source damage%v", index, target, damaged)
		}
		if index >= 16 && target && (bounds.Top != actor.Collision.Top+1 || bounds.Bottom != actor.Collision.Bottom-1) {
			t.Fatal("satellite aim retained an armored border line")
		}
	}
	if forecastDigest(w) != before {
		t.Fatal("read-only guardian targeting changed the world")
	}
	for index := 15; index < 20; index++ {
		actor := w.fourthMiddleActors[index]
		w.damageFourthGuardian(actor, targetPoint(actor.Collision), w.FourthMiddle.Parts[index].Health)
	}
	for _, index := range []int{4, 5} {
		if _, target := presentationTargetBounds(w, w.fourthMiddleActors[index]); !target {
			t.Fatal("native outer-target destruction did not expose the shared core callback")
		}
	}

	w = originalGuardianTargetWorld(t, 4, true)
	for index, actor := range w.fourthFinalActors {
		_, target := presentationTargetBounds(w, actor)
		copy := *w.FourthFinal
		copy.Strike(index, 1, w.ScrollY)
		damaged := index < 3 && copy.Parts[index].Health != w.FourthFinal.Parts[index].Health
		if target != damaged {
			t.Fatalf("final role%d target%v disagrees with source damage%v", index, target, damaged)
		}
	}
	for _, index := range []int{1, 2} {
		actor := w.fourthFinalActors[index]
		w.damageFourthGuardian(actor, targetPoint(actor.Collision), w.FourthFinal.Parts[index].Health)
		if _, target := presentationTargetBounds(w, actor); target {
			t.Fatal("retained closed eye remained an aim target")
		}
	}
	if err := w.advanceFourthPart(w.fourthFinalActors[0]); err != nil {
		t.Fatal(err)
	}
	core := *w.fourthFinalActors[0]
	core.Visible = false // Its source tile renderer does not control damage.
	if core.Patch == nil {
		t.Fatal("original core fixture omitted the tiled renderer")
	}
	if _, target := presentationTargetBounds(w, &core); !target {
		t.Fatal("exposed tiled core was rejected by sprite visibility")
	}
}

func TestPresentationFifthGuardianTargetsMatchSourceDamageOptional(t *testing.T) {
	w := originalGuardianTargetWorld(t, 5, false)
	before := forecastDigest(w)
	for index, actor := range w.fifthMiddleActors {
		_, target := presentationTargetBounds(w, actor)
		copy := *w.FifthMiddle
		copy.Damage(index, 1)
		damaged := copy.Parts[index].Health != w.FifthMiddle.Parts[index].Health
		if target != damaged {
			t.Fatalf("middle role%d target%v disagrees with source damage%v", index, target, damaged)
		}
	}
	if forecastDigest(w) != before {
		t.Fatal("fifth guardian targeting changed the world")
	}
	w.damageFifthGuardian(w.fifthMiddleActors[1], uint16(w.FifthMiddle.Parts[1].Health))
	if _, target := presentationTargetBounds(w, w.fifthMiddleActors[1]); target {
		t.Fatal("retained middle mount wreck remained a target")
	}

	w = originalGuardianTargetWorld(t, 5, true)
	for index, actor := range w.fifthFinalActors {
		_, target := presentationTargetBounds(w, actor)
		copy := *w.FifthFinal
		copy.DamagePart(w.fifthFinalArt, index, 1)
		damaged := index != 21 && copy.Parts[index].Health != w.FifthFinal.Parts[index].Health
		if target != damaged {
			t.Fatalf("final role%d target%v disagrees with eligible source damage%v", index, target, damaged)
		}
	}
	for index := 3; index < 21; index++ {
		w.damageFifthGuardian(w.fifthFinalActors[index], uint16(w.FifthFinal.Parts[index].Health))
	}
	w.advanceFifthGuardian(true)
	if w.FifthFinal.OuterRemaining != 0 {
		t.Fatal("source callbacks did not destroy eighteen outer parts")
	}
	if _, target := presentationTargetBounds(w, w.fifthFinalActors[21]); !target {
		t.Fatal("source outer-part destruction did not reveal the real core")
	}
	for index := 3; index < 21; index++ {
		if _, target := presentationTargetBounds(w, w.fifthFinalActors[index]); target {
			t.Fatal("destroyed final outer part remained a target")
		}
	}
}
