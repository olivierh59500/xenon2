package audio

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"
)

func immediateSampleStream(t *testing.T) *Stream {
	t.Helper()
	bank, waveforms := testBank()
	bank.Effects = append(bank.Effects, Sequence{
		ID: "sample", Ticks: 3, LoopTick: -1,
		StartEvents: []Event{{Kind: "period", Period: 400}, {Kind: "volume", Volume: 32}, {Kind: "start", Sample: "negative"}, {Kind: "loop", Sample: "negative"}},
		Events:      []Event{{Tick: 2, Kind: "stop"}},
	})
	stream, err := NewStream(bank, waveforms, 10000)
	if err != nil {
		t.Fatal(err)
	}
	return stream
}

func readAudioFrames(t *testing.T, stream *Stream, frames int) []byte {
	t.Helper()
	pcm := make([]byte, frames*4)
	if _, err := stream.Read(pcm); err != nil {
		t.Fatal(err)
	}
	return pcm
}

func requireFirstSample(t *testing.T, pcm []byte) {
	t.Helper()
	left, right := int16(binary.LittleEndian.Uint16(pcm)), int16(binary.LittleEndian.Uint16(pcm[2:]))
	// The first signed PCM byte is -128, at volume32 and the original stereo
	// mixer scale2. This expected value does not use a second replay stream.
	if left != 0 || right != -8192 {
		t.Fatalf("direct initial sample is %d/%d, want 0/-8192", left, right)
	}
}

func TestDirectSampleStartsWithinUnfinishedAudioTick(t *testing.T) {
	stream := immediateSampleStream(t)
	readAudioFrames(t, stream, 37)
	if err := stream.PlayEffect("sample", 2); err != nil {
		t.Fatal(err)
	}
	requireFirstSample(t, readAudioFrames(t, stream, 1))
	if stream.effects[2].age != 0 {
		t.Fatal("direct sample admission advanced its interrupt timer")
	}
	readAudioFrames(t, stream, 162)
	if stream.effects[2].age != 0 {
		t.Fatal("sample timer advanced during the remaining audio phase")
	}
	readAudioFrames(t, stream, 1)
	if stream.effects[2].age != 1 {
		t.Fatal("sample timer did not advance at the next original interrupt")
	}
}

func TestQueuedSampleStartsAndAdvancesOnItsAdmissionInterrupt(t *testing.T) {
	stream := immediateSampleStream(t)
	readAudioFrames(t, stream, 37)
	if err := stream.QueueEffect("sample", 2); err != nil {
		t.Fatal(err)
	}
	pcm := readAudioFrames(t, stream, 163)
	if !bytes.Equal(pcm, make([]byte, len(pcm))) || stream.EffectActive(2) {
		t.Fatal("queued sample played before the next audio interrupt")
	}
	requireFirstSample(t, readAudioFrames(t, stream, 1))
	if stream.effects[2].age != 1 {
		t.Fatal("queued sample omitted the original first interrupt timer advance")
	}
}

func TestDirectSynthesizedEffectKeepsFirstEnvelopeOnNextInterrupt(t *testing.T) {
	stream := immediateSampleStream(t)
	readAudioFrames(t, stream, 37)
	if err := stream.PlayEffect("shot", 2); err != nil {
		t.Fatal(err)
	}
	pcm := readAudioFrames(t, stream, 163)
	if !bytes.Equal(pcm, make([]byte, len(pcm))) {
		t.Fatal("direct synthesized effect applied its first envelope prematurely")
	}
	requireFirstSample(t, readAudioFrames(t, stream, 1))
}

func TestAudioBankValidatesOptionalAdmissionEvents(t *testing.T) {
	bank, waves := testBank()
	encoded, err := json.Marshal(bank)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("start_events")) {
		t.Fatal("legacy bank gained an admission field without any events")
	}
	var legacy Bank
	if err := json.Unmarshal(encoded, &legacy); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Validate(waves); err != nil {
		t.Fatalf("previous schema version1 bank no longer loads: %v", err)
	}
	for _, invalid := range []Event{{Kind: "start", Sample: "missing"}, {Tick: 1, Kind: "volume", Volume: 32}, {Kind: "volume", Volume: 65}} {
		bank.Effects[0].StartEvents = []Event{invalid}
		if err := bank.Validate(waves); err == nil {
			t.Fatalf("invalid admission event accepted: %+v", invalid)
		}
	}
	bank.Effects[0].StartEvents = nil
	bank.Music[0].StartEvents = []Event{{Kind: "volume", Volume: 32}}
	if err := bank.Validate(waves); err == nil {
		t.Fatal("effect-only admission events were accepted for a music score")
	}
}
