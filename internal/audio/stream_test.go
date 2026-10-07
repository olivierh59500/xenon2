package audio

import (
	"bytes"
	"encoding/binary"
	"sync"
	"testing"
)

func testBank() (*Bank, map[string][]byte) {
	b := &Bank{Version: SchemaVersion, TickRate: TickRate, Clock: PALClock, Samples: []Sample{{ID: "positive", Bytes: 4}, {ID: "negative", Bytes: 2}}, Music: []Sequence{{ID: "test", Ticks: 4, LoopTick: 1, PhaseIncrement: 20, Events: []Event{{Tick: 0, Channel: 0, Kind: "period", Period: 354}, {Tick: 0, Channel: 0, Kind: "volume", Volume: 64}, {Tick: 0, Channel: 0, Kind: "loop", Sample: "positive"}, {Tick: 0, Channel: 0, Kind: "start", Sample: "positive"}, {Tick: 0, Channel: 1, Kind: "period", Period: 354}, {Tick: 0, Channel: 1, Kind: "volume", Volume: 64}, {Tick: 0, Channel: 1, Kind: "loop", Sample: "negative"}, {Tick: 0, Channel: 1, Kind: "start", Sample: "negative"}, {Tick: 2, Channel: 0, Kind: "volume", Volume: 32}}}}, Effects: []Sequence{{ID: "shot", Ticks: 3, LoopTick: -1, Events: []Event{{Tick: 0, Channel: 0, Kind: "period", Period: 400}, {Tick: 0, Channel: 0, Kind: "volume", Volume: 32}, {Tick: 0, Channel: 0, Kind: "loop", Sample: "negative"}, {Tick: 0, Channel: 0, Kind: "start", Sample: "negative"}, {Tick: 2, Channel: 0, Kind: "stop"}}}}}
	return b, map[string][]byte{"positive": {127, 0, 128, 0}, "negative": {128, 127}}
}

func TestStreamSignedSamplesAndAmigaStereo(t *testing.T) {
	b, w := testBank()
	s, err := NewStream(b, w, 10000)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.PlayMusic("test"); err != nil {
		t.Fatal(err)
	}
	p := make([]byte, 4)
	if _, err = s.Read(p); err != nil {
		t.Fatal(err)
	}
	if int16(binary.LittleEndian.Uint16(p)) != 16256 || int16(binary.LittleEndian.Uint16(p[2:])) != -16384 {
		t.Fatalf("unexpected signed stereo frame %v", p)
	}
}

func TestStreamIsIndependentOfOutputBufferSize(t *testing.T) {
	b, w := testBank()
	a, _ := NewStream(b, w, 10001)
	bstream, _ := NewStream(b, w, 10001)
	if err := a.PlayMusic("test"); err != nil {
		t.Fatal(err)
	}
	if err := bstream.PlayMusic("test"); err != nil {
		t.Fatal(err)
	}
	whole := make([]byte, 40000)
	a.Read(whole)
	parts := make([]byte, len(whole))
	offset := 0
	for offset < len(parts) {
		size := 172
		if size > len(parts)-offset {
			size = len(parts) - offset
		}
		bstream.Read(parts[offset : offset+size])
		offset += size
	}
	if !bytes.Equal(whole, parts) {
		t.Fatal("PCM depends on reader chunk size")
	}
}

func TestStreamControlsCanRunAlongsideAudio(t *testing.T) {
	b, w := testBank()
	s, _ := NewStream(b, w, 44100)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		p := make([]byte, 1024)
		for i := 0; i < 100; i++ {
			s.Read(p)
		}
	}()
	for i := 0; i < 100; i++ {
		if err := s.PlayMusic("test"); err != nil {
			t.Fatal(err)
		}
		if err := s.PlayEffect("shot", i%4); err != nil {
			t.Fatal(err)
		}
		s.StopEffects()
		s.StopMusic()
	}
	wg.Wait()
}

func TestBankRejectsMalformedSampleReferences(t *testing.T) {
	b, w := testBank()
	b.Music[0].Events[3].Sample = "missing"
	if _, err := NewStream(b, w, 44100); err == nil {
		t.Fatal("unknown sample reference accepted")
	}
}

func BenchmarkStream(b *testing.B) {
	bank, w := testBank()
	s, _ := NewStream(bank, w, 44100)
	s.PlayMusic("test")
	buffer := make([]byte, 4096)
	b.ReportAllocs()
	b.SetBytes(int64(len(buffer)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Read(buffer)
	}
}

func TestQueuedEffectWaitsForTickAndLastRequestWins(t *testing.T) {
	bank, waves := testBank()
	alternate := bank.Effects[0]
	alternate.ID = "alternate"
	alternate.Events = append([]Event(nil), alternate.Events...)
	for i := range alternate.Events {
		if alternate.Events[i].Kind == "start" || alternate.Events[i].Kind == "loop" {
			alternate.Events[i].Sample = "positive"
		}
	}
	bank.Effects = append(bank.Effects, alternate)
	stream, err := NewStream(bank, waves, 44100)
	if err != nil {
		t.Fatal(err)
	}
	stream.Read(make([]byte, 4))
	if err = stream.QueueEffect("shot", 2); err != nil {
		t.Fatal(err)
	}
	if err = stream.QueueEffect("alternate", 2); err != nil {
		t.Fatal(err)
	}
	before := make([]byte, 881*4)
	stream.Read(before)
	if !bytes.Equal(before, make([]byte, len(before))) {
		t.Fatal("queued effect changed the current musical tick")
	}
	frame := make([]byte, 4)
	stream.Read(frame)
	if left, right := int16(binary.LittleEndian.Uint16(frame)), int16(binary.LittleEndian.Uint16(frame[2:])); left != 0 || right != 8128 {
		t.Fatalf("last queued effect was not dispatched: %d/%d", left, right)
	}
}

func TestEffectActivityAndQueuedGlobalStopPreserveNewRequests(t *testing.T) {
	bank, waves := testBank()
	stream, err := NewStream(bank, waves, 10000)
	if err != nil {
		t.Fatal(err)
	}
	if err = stream.PlayMusic("test"); err != nil {
		t.Fatal(err)
	}
	stream.tick()
	if err = stream.PlayEffect("shot", 1); err != nil {
		t.Fatal(err)
	}
	stream.tick()
	if !stream.EffectActive(1) || stream.EffectActive(4) || stream.EffectActive(-1) {
		t.Fatal("incorrect native voice ownership")
	}
	if err = stream.QueueEffect("shot", 2); err != nil {
		t.Fatal(err)
	}
	if stream.EffectActive(2) {
		t.Fatal("queued sound became active before dispatch")
	}
	stream.QueueStopEffects()
	if !stream.EffectActive(1) {
		t.Fatal("queued stop changed ownership before its tick")
	}
	stream.tick()
	if stream.EffectActive(1) || !stream.EffectActive(2) {
		t.Fatal("global stop discarded the newer dispatch or retained old voice ownership")
	}
	if stream.voices[1].current != stream.shadow[1].next {
		t.Fatal("terminated voice did not restore the music reload buffer")
	}
	stream.tick()
	stream.tick()
	if stream.EffectActive(2) {
		t.Fatal("finished effect retained ownership")
	}
}
