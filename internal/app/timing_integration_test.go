package app

import (
	"testing"
	"xenon2/internal/presentation"
)

func TestMeasuredGameplayClockSurvivesAdmissionAndFadeOptional(t *testing.T) {
	g := frontendGame(t)
	g.Config.LogicPALRefreshes = 3
	if err := g.StartSession(1, 1); err != nil {
		t.Fatal(err)
	}
	count := func() int {
		c := g.clock
		n := 0
		for range 180 {
			n += c.Advance()
		}
		return n
	}
	if n := count(); n != 50 {
		t.Fatalf("admission rounded 50/3 Hz: %d passes in 3 s", n)
	}
	g.startLevelFade()
	finishRenderFade(t, g)
	if n := count(); n != 50 {
		t.Fatalf("fade reset selected timing: %d", n)
	}
	g.resetPresentationStars(presentation.LogoDelay)
	c := g.menuClock
	n := 0
	for range 180 {
		n += c.Advance()
	}
	if n != 75 {
		t.Fatal("gameplay profile changed unmeasured presentation clock")
	}
	c = g.palClock
	n = 0
	for range 180 {
		n += c.Advance()
	}
	if n != 150 {
		t.Fatal("gameplay profile changed 50 Hz effects/fades")
	}
}
