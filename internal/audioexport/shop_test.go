package audioexport

import (
	"bytes"
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"xenon2/internal/audio"
)

func TestPrivateOriginalShopEffects(t *testing.T) {
	dir := os.Getenv("XENON2_AUDIO_TEST_DIR")
	if dir == "" {
		t.Skip("local original audio reference not supplied")
	}
	data, err := os.ReadFile(filepath.Join(dir, "analysis", "05c400f8.unpacked"))
	if err != nil {
		t.Fatal(err)
	}
	bank, waves, err := DecodeShopAudio(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(bank.Effects) != 27 || len(bank.Music) != 0 {
		t.Fatal("wrong shop bank catalogue")
	}
	f, err := os.Open(filepath.Join(dir, "analysis", "shop-effects-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	type state struct {
		sample                  string
		period, volume, enabled int
	}
	var current state
	var sequence audio.Sequence
	family, effect, age, index := -1, -1, 0, 0
	for _, row := range rows[1:] {
		group, id, tick := number(t, row[0]), number(t, row[1]), number(t, row[2])
		if family != group || effect != id {
			family, effect = group, id
			current = state{}
			age, index = 0, 0
			position := id
			if group == 1 {
				position += 23
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
				current.sample = e.Sample
				current.enabled = 1
			case "loop":
				current.sample = e.Sample
			case "stop":
				current.enabled = 0
			case "period":
				current.period = int(e.Period)
			case "volume":
				current.volume = int(e.Volume)
			}
			index++
		}
		age++
		enabled := number(t, row[7])
		if current.enabled != enabled {
			t.Fatalf("family %d effect %d tick %d activation Go %d original %d", group, id, tick, current.enabled, enabled)
		}
		if enabled == 0 {
			continue
		}
		volume := number(t, row[6])
		if volume&64 != 0 {
			volume = 64
		} else {
			volume &= 63
		}
		if period := number(t, row[5]); current.period != period || current.volume != volume {
			t.Fatalf("family %d effect %d tick %d period/volume Go %d/%d original %d/%d", group, id, tick, current.period, current.volume, period, volume)
		}
		pointer := number(t, row[3]) - 0x54e00
		size := number(t, row[4]) * 2
		if pointer < 0 || pointer+size > len(data) || !bytes.Equal(waves[current.sample], data[pointer:pointer+size]) {
			t.Fatalf("family %d effect %d tick %d waveform differs", group, id, tick)
		}
	}
	t.Logf("Compared %d original shop audio ticks across all 27 effects.", len(rows)-1)
}
