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

func TestDefaultGameplayUsesMeasuredA500Cadence(t *testing.T) {
	g := frontendGame(t)
	if err := g.StartSession(1, 1); err != nil {
		t.Fatal(err)
	}
	clock, passes := g.clock, 0
	for range 180 {
		passes += clock.Advance()
	}
	if passes != 50 {
		t.Fatalf("default gameplay ran %d passes in three seconds; want 50", passes)
	}
	g.Config.LogicPALRefreshes = 2
	clock, passes = g.newLogicClock(), 0
	for range 180 {
		passes += clock.Advance()
	}
	if passes != 75 {
		t.Fatal("explicit source maximum no longer provides 25 passes per second")
	}
}

func TestActualAttractCreditPairsKeepMeasuredSixSecondIntervals(t *testing.T) {
	g := frontendGame(t)
	g.BeginAttract()
	var pairUpdates []int
	previous := 0
	for update := 0; update < 60*30; update++ {
		advanceFrontend(t, g, inputFrame{})
		if g.director.Phase == presentation.Credits && g.director.CreditPair != previous {
			previous = g.director.CreditPair
			pairUpdates = append(pairUpdates, update+1)
			if len(pairUpdates) == 4 {
				break
			}
		}
	}
	if len(pairUpdates) != 4 {
		t.Fatal("real attract flow did not traverse four credit pairs")
	}
	for pair := 1; pair < len(pairUpdates); pair++ {
		if interval := pairUpdates[pair] - pairUpdates[pair-1]; interval != 360 {
			t.Fatalf("credit pair %d elapsed %d display updates; want six seconds at 60 Hz", pair, interval)
		}
	}
	menu, pal := g.menuClock, g.palClock
	menuPasses, palPasses := 0, 0
	for range 180 {
		menuPasses += menu.Advance()
		palPasses += pal.Advance()
	}
	if menuPasses != 75 || palPasses != 150 {
		t.Fatal("credit pacing changed menu or PAL fade clocks")
	}
	advanceFrontend(t, g, inputFrame{confirm: true})
	awaitFrontendBoundary(t, g, 180, "skip paced credits to menu", func() bool { return g.Screen == TitleScreen })
	t.Logf("Real attract credit pair onsets: %v; every subsequent interval is six seconds", pairUpdates)
}
