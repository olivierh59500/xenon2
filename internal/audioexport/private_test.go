package audioexport

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"xenon2/internal/audio"
)

func TestPrivateOriginalEffectVoiceTrace(t *testing.T) {
	dir := os.Getenv("XENON2_AUDIO_TEST_DIR")
	if dir == "" {
		t.Skip("local original audio reference not supplied")
	}
	data, err := os.ReadFile(filepath.Join(dir, "XenonII-unpacked.bin"))
	if err != nil {
		t.Fatal(err)
	}
	bank, waveforms, err := DecodeMusic(data)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(dir, "analysis", "effects-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	type effectState struct {
		sample                  string
		period, volume, enabled int
	}
	state := effectState{}
	family, effect, age, index := -1, -1, 0, 0
	var sequence audio.Sequence
	for _, row := range rows[1:] {
		f, n, tick := number(t, row[0]), number(t, row[1]), number(t, row[2])
		if f != family || n != effect {
			family, effect = f, n
			state = effectState{}
			age, index = 0, 0
			position := n + 18
			if family == 1 {
				position = n
			}
			sequence = bank.Effects[position]
		}
		if age == sequence.Ticks && sequence.LoopTick >= 0 {
			age = sequence.LoopTick
			index = 0
			for index < len(sequence.Events) && sequence.Events[index].Tick < age {
				index++
			}
		}
		for index < len(sequence.Events) && sequence.Events[index].Tick == age {
			e := sequence.Events[index]
			switch e.Kind {
			case "start":
				state.sample = e.Sample
				state.enabled = 1
			case "loop":
				state.sample = e.Sample
			case "stop":
				state.enabled = 0
			case "volume":
				state.volume = int(e.Volume)
			case "period":
				state.period = int(e.Period)
			}
			index++
		}
		age++
		enabled := number(t, row[7])
		period, volume := number(t, row[5]), number(t, row[6])
		if volume&64 != 0 {
			volume = 64
		} else {
			volume &= 63
		}
		if state.enabled != enabled {
			t.Fatalf("effect family %d index %d tick %d enabled Go %d original %d", family, effect, tick, state.enabled, enabled)
		}
		if enabled == 0 {
			continue
		}
		if state.period != period || state.volume != volume {
			t.Fatalf("effect family %d index %d tick %d period/volume Go %d/%d original %d/%d", family, effect, tick, state.period, state.volume, period, volume)
		}
		pointer, words := number(t, row[3]), number(t, row[4])
		if pointer < 0 || pointer+words*2 > len(data) {
			t.Fatal("original effect waveform leaves source")
		}
		if !bytes.Equal(waveforms[state.sample], data[pointer:pointer+words*2]) {
			t.Fatalf("effect family %d index %d tick %d waveform differs", family, effect, tick)
		}
	}
	t.Logf("Compared %d original effect ticks across all 41 effects.", len(rows)-1)
}

// The source and oracle trace remain local; ordinary builds do not need them.
func TestPrivateOriginalMusicVoiceTrace(t *testing.T) {
	compareMusic(t, "audio-trace.csv", 0)
}

func TestPrivateOriginalMenuMusicVoiceTrace(t *testing.T) {
	compareMusic(t, "menu-audio-trace.csv", 1)
}

func compareMusic(t *testing.T, traceFile string, sequenceIndex int) {
	dir := os.Getenv("XENON2_AUDIO_TEST_DIR")
	if dir == "" {
		t.Skip("local original audio reference not supplied")
	}
	data, err := os.ReadFile(filepath.Join(dir, "XenonII-unpacked.bin"))
	if err != nil {
		t.Fatal(err)
	}
	bank, waveforms, err := DecodeMusic(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(bank.Samples) < 39 || len(bank.Effects) != 41 {
		t.Fatal("incomplete sample catalogue")
	}
	encoded, err := json.Marshal(bank)
	if err != nil {
		t.Fatal(err)
	}
	var portable audio.Bank
	if err = json.Unmarshal(encoded, &portable); err != nil {
		t.Fatal(err)
	}
	if err = portable.Validate(waveforms); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(dir, "analysis", traceFile))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	type registers struct{ pointer, words, period, volume, enabled int }
	var voices [4]registers
	sources := map[string]int{"silence": 0x2aba0}
	cursor := 0x1bb80
	for i := 0; i < 20; i++ {
		sources[bank.Samples[i].ID] = cursor + 6
		cursor += 6 + int(binaryLong(data, cursor))
	}
	sequence := bank.Music[sequenceIndex]
	eventIndex, age, phase := 0, 0, byte(0)
	for tick := 0; tick < (len(rows)-1)/4; tick++ {
		next := uint16(phase) + uint16(sequence.PhaseIncrement)
		phase = byte(next)
		if next <= 255 {
			if age == sequence.Ticks {
				age = sequence.LoopTick
				eventIndex = 0
				for eventIndex < len(sequence.Events) && sequence.Events[eventIndex].Tick < age {
					eventIndex++
				}
			}
			for eventIndex < len(sequence.Events) && sequence.Events[eventIndex].Tick == age {
				e := sequence.Events[eventIndex]
				v := &voices[e.Channel]
				switch e.Kind {
				case "start":
					v.pointer = sources[e.Sample]
					v.words = len(waveforms[e.Sample]) / 2
					v.enabled = 1
				case "loop":
					v.pointer = sources[e.Sample]
					v.words = len(waveforms[e.Sample]) / 2
				case "stop":
					v.enabled = 0
				case "period":
					v.period = int(e.Period)
				case "volume":
					v.volume = int(e.Volume)
				}
				eventIndex++
			}
			age++
		}
		for ch := 0; ch < 4; ch++ {
			row := rows[1+tick*4+ch]
			got := voices[ch]
			want := registers{number(t, row[2]), number(t, row[3]), number(t, row[4]), number(t, row[5]), number(t, row[6])}
			if want.volume&64 != 0 {
				want.volume = 64
			} else {
				want.volume &= 63
			}
			if got != want {
				t.Fatalf("tick %d channel %d: Go %+v, original %+v", tick, ch, got, want)
			}
		}
	}
	t.Logf("Compared %d original musical ticks and four voices.", (len(rows)-1)/4)
}

func binaryLong(b []byte, at int) uint32 {
	return uint32(b[at])<<24 | uint32(b[at+1])<<16 | uint32(b[at+2])<<8 | uint32(b[at+3])
}
func number(t *testing.T, s string) int {
	t.Helper()
	v, err := strconv.Atoi(s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
