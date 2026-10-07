package app

import (
	"bytes"
	"testing"

	"xenon2/internal/audio"
	"xenon2/internal/shopui"
)

func shopAudioStreams(t *testing.T) (*Game, *audio.Stream) {
	t.Helper()
	g := frontendGame(t)
	if err := g.EnterShop(false); err != nil {
		t.Fatal(err)
	}
	awaitFrontendBoundary(t, g, 160, "merchant entrance fade", func() bool { return g.fade == nil || g.fade.Done })
	g.Config.Mute = false
	reference, err := audio.NewStream(g.Bundle.AudioBank, g.Bundle.Waveforms, 44100)
	if err != nil {
		t.Fatal(err)
	}
	return g, reference
}

func compareShopPCM(t *testing.T, actual, reference *audio.Stream, frames int) {
	t.Helper()
	a, b := make([]byte, frames*4), make([]byte, frames*4)
	if _, err := actual.Read(a); err != nil {
		t.Fatal(err)
	}
	if _, err := reference.Read(b); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("merchant PCM diverged from its source dispatch boundary")
	}
}

func TestShopHeadphoneCueAndTerminationRespectAudioPhase(t *testing.T) {
	g, reference := shopAudioStreams(t)
	// Leave an unfinished audio tick before the UI emits the original direct
	// headphone call. Ownership must change without waiting for that tick.
	compareShopPCM(t, g.stream, reference, 381)
	awaitFrontendBoundary(t, g, 100, "direct headphone sample", func() bool { return g.shop.Entrance == 5 })
	if !g.stream.EffectActive(0) || g.stream.EffectActive(1) || g.stream.EffectActive(2) || g.stream.EffectActive(3) {
		t.Fatal("headphone call did not directly acquire its original voice")
	}
	if err := reference.PlayEffect("shop-sampled-effect-03", 0); err != nil {
		t.Fatal(err)
	}
	compareShopPCM(t, g.stream, reference, 1000)
	awaitFrontendBoundary(t, g, 150, "headphone termination request", func() bool { return g.shop.Phase == shopui.HeadphoneHand && g.shop.PhasePass == 19 })
	if !g.stream.EffectActive(0) {
		t.Fatal("merchant termination stopped its owned voice before the next audio tick")
	}
	reference.QueueStopEffects()
	// There are 383 output frames left in the existing tick. They must retain
	// the current sample; only the following frame dispatches the stop.
	compareShopPCM(t, g.stream, reference, 383)
	if !g.stream.EffectActive(0) {
		t.Fatal("queued merchant stop lost the remaining audio phase")
	}
	compareShopPCM(t, g.stream, reference, 1)
	if g.stream.EffectActive(0) {
		t.Fatal("merchant termination did not dispatch at its next audio tick")
	}
}

func TestShopNavigationCueRemainsInterruptQueued(t *testing.T) {
	g, reference := shopAudioStreams(t)
	g.Config.Mute = true
	awaitFrontendBoundary(t, g, 300, "merchant navigation", func() bool { return g.shop.Phase == shopui.Selling && g.shop.Revealed == len(g.shop.Dialogue) })
	g.Config.Mute = false
	compareShopPCM(t, g.stream, reference, 381)
	advanceFrontend(t, g, inputFrame{rightPressed: true})
	if g.stream.EffectActive(2) {
		t.Fatal("navigation request acquired its voice before the audio interrupt")
	}
	if err := reference.QueueEffect("shop-synthesized-effect-12", 2); err != nil {
		t.Fatal(err)
	}
	compareShopPCM(t, g.stream, reference, 501)
	if g.stream.EffectActive(2) {
		t.Fatal("navigation cue changed ownership during the previous audio tick")
	}
	compareShopPCM(t, g.stream, reference, 1)
	if !g.stream.EffectActive(2) {
		t.Fatal("navigation cue never dispatched on its original voice")
	}
	compareShopPCM(t, g.stream, reference, 1764)
}
