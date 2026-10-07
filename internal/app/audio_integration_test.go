package app

import "testing"

func TestGameRoutesImmediateAndQueuedOriginalEffects(t *testing.T) {
	g := frontendGame(t)
	g.Config.Mute = false
	driver := g.Driver.(*worldDriver)
	w := driver.world
	w.ImmediateSoundRequests[0] = "sampled-effect-03"
	w.SoundRequests[1] = "sampled-effect-05"
	if err := g.consumeDriverAudio(); err != nil {
		t.Fatal(err)
	}
	if w.ImmediateSoundRequests != [4]string{} || w.SoundRequests != [4]string{} {
		t.Fatal("audio requests were retained for duplicate dispatch")
	}
	if !g.stream.EffectActive(0) || g.stream.EffectActive(1) {
		t.Fatal("direct sound and interrupt-queued sound lost their separate boundaries")
	}
	// Audio consumption advances the original PAL tick; no renderer update
	// should be required to dispatch the queued sampled effect.
	buffer := make([]byte, 882*4)
	if _, err := g.stream.Read(buffer); err != nil {
		t.Fatal(err)
	}
	if !g.stream.EffectActive(1) {
		t.Fatal("queued original effect did not acquire its target voice")
	}
	g.deliverEffectActivity()
	if !w.EffectActive[0] || !w.EffectActive[1] || w.EffectActive[2] || w.EffectActive[3] {
		t.Fatal("gameplay did not receive the audio stream's effect ownership")
	}
}
