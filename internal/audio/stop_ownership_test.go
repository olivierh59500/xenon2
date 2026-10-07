package audio

import (
	"bytes"
	"testing"
)

func ownershipStreams(t *testing.T) (*Stream, *Stream) {
	t.Helper()
	bank, waves := testBank()
	// A nonintegral Paula/output-rate ratio makes a spurious DMA restart
	// observable even when its waveform happens to end on an integer boundary.
	bank.Music[0].PhaseIncrement = 0
	bank.Music[0].Events[0].Period = 367
	bank.Music[0].Events[4].Period = 367
	a, err := NewStream(bank, waves, 10000)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewStream(bank, waves, 10000)
	if err != nil {
		t.Fatal(err)
	}
	for _, stream := range []*Stream{a, b} {
		if err := stream.PlayMusic("test"); err != nil {
			t.Fatal(err)
		}
	}
	return a, b
}

func TestStoppingInactiveEffectsDoesNotRestartMusicDMA(t *testing.T) {
	for _, queued := range []bool{false, true} {
		a, b := ownershipStreams(t)
		prefix := make([]byte, 123*4)
		a.Read(prefix)
		b.Read(prefix)
		if queued {
			a.QueueStopEffects()
		} else {
			a.StopEffects()
		}
		got, want := make([]byte, 801*4), make([]byte, 801*4)
		a.Read(got)
		b.Read(want)
		if !bytes.Equal(got, want) {
			t.Fatalf("queued=%v: terminating unowned effect records restarted continuous music PCM", queued)
		}
	}
}

func TestStoppingOneEffectPreservesOtherMusicVoices(t *testing.T) {
	a, b := ownershipStreams(t)
	buffer := make([]byte, 123*4)
	a.Read(buffer)
	b.Read(buffer)
	if err := a.PlayEffect("shot", 1); err != nil {
		t.Fatal(err)
	}
	a.Read(buffer)
	b.Read(buffer)
	a.StopEffects()
	got, want := make([]byte, 401*4), make([]byte, 401*4)
	a.Read(got)
	b.Read(want)
	for offset := 0; offset < len(got); offset += 4 {
		if !bytes.Equal(got[offset:offset+2], want[offset:offset+2]) {
			t.Fatal("right effect termination restarted the unowned left music DMA")
		}
	}
	if a.EffectActive(1) {
		t.Fatal("terminated effect retained ownership")
	}
}
