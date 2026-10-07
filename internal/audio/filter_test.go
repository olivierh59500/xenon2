package audio

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func TestA500PermanentFilterImpulseAndSeparateOutputs(t *testing.T) {
	for _, test := range []struct {
		rate int
		want [8]int32
	}{
		{44100, [8]int32{7417, 4320, 2255, 1161, 597, 306, 157, 81}},
		{48000, [8]int32{6677, 4380, 2449, 1328, 715, 385, 207, 111}},
		{96000, [8]int32{2891, 3189, 2709, 2097, 1556, 1130, 811, 579}},
	} {
		f := newA500Filter(test.rate)
		for frame, want := range test.want {
			var input int32
			if frame == 0 {
				input = 16384
			}
			if got := f.sample(0, input); got != want {
				t.Fatalf("rate%d frame%d impulse%d; want%d", test.rate, frame, got, want)
			}
			if got := f.sample(1, 0); got != 0 {
				t.Fatal("left reconstruction tail leaked into the right output")
			}
		}
	}
}

func TestA500PermanentFilterKeepsBassAndAttenuatesHighFrequencies(t *testing.T) {
	measure := func(frequency int) float64 {
		f := newA500Filter(48000)
		var inputEnergy, outputEnergy float64
		for frame := 0; frame < 9600; frame++ {
			input := int32(12000 * math.Sin(2*math.Pi*float64(frame*frequency)/48000))
			output := f.sample(0, input)
			if frame >= 4800 {
				inputEnergy += float64(input) * float64(input)
				outputEnergy += float64(output) * float64(output)
			}
		}
		return math.Sqrt(outputEnergy / inputEnergy)
	}
	if bass, high := measure(1000), measure(12000); bass < .97 || bass > 1 || high < .35 || high > .36 {
		t.Fatalf("unexpected permanent-filter response: bass%.6f high%.6f", bass, high)
	}
}

func TestA500StreamChunkingAndReplayStateRemainIndependentOfFiltering(t *testing.T) {
	bank, waves := testBank()
	whole, _ := NewA500Stream(bank, waves, 44100)
	chunks, _ := NewA500Stream(bank, waves, 44100)
	raw, _ := NewStream(bank, waves, 44100)
	for _, stream := range []*Stream{whole, chunks, raw} {
		if err := stream.PlayMusic("test"); err != nil {
			t.Fatal(err)
		}
		if err := stream.QueueEffect("shot", 2); err != nil {
			t.Fatal(err)
		}
	}
	a, b, c := make([]byte, 44100*4), make([]byte, 44100*4), make([]byte, 44100*4)
	whole.Read(a)
	raw.Read(c)
	for offset := 0; offset < len(b); {
		end := min(len(b), offset+148)
		chunks.Read(b[offset:end])
		offset = end
	}
	if !bytes.Equal(a, b) || bytes.Equal(a, c) {
		t.Fatal("filtering depends on reader chunks or did not affect the output")
	}
	if whole.voices != raw.voices || whole.shadow != raw.shadow || whole.music != raw.music || whole.effects != raw.effects || whole.remaining != raw.remaining || whole.clockRemainder != raw.clockRemainder {
		t.Fatal("reconstruction filtering altered voice registers, dispatch or audio time")
	}
}

func TestA500StreamRetainsOutputDecayWhenMusicStops(t *testing.T) {
	bank, waves := testBank()
	stream, _ := NewA500Stream(bank, waves, 44100)
	stream.PlayMusic("test")
	frame := make([]byte, 4)
	stream.Read(frame)
	stream.StopMusic()
	stream.Read(frame)
	if int16(binary.LittleEndian.Uint16(frame)) <= 0 || int16(binary.LittleEndian.Uint16(frame[2:])) >= 0 {
		t.Fatal("stopping music reset the two physical output tails")
	}
	stream.Read(make([]byte, 1024*4))
	stream.Read(frame)
	if !bytes.Equal(frame, make([]byte, 4)) {
		t.Fatal("silent output did not settle after the reconstruction tail")
	}
}

func BenchmarkA500Stream(b *testing.B) {
	bank, waves := testBank()
	stream, err := NewA500Stream(bank, waves, 44100)
	if err != nil {
		b.Fatal(err)
	}
	if err := stream.PlayMusic("test"); err != nil {
		b.Fatal(err)
	}
	buffer := make([]byte, 4096)
	b.SetBytes(int64(len(buffer)))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		stream.Read(buffer)
	}
}
