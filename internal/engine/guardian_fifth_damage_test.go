package engine

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"xenon2/internal/visualassets"
)

func TestFifthGuardianDamageNativeTraceOptional(t *testing.T) {
	dir := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if dir == "" {
		t.Skip("local fifth damage reference not supplied")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(dir), "imported", "04820138.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	terrain, err := visualassets.DecodeTerrain(raw)
	if err != nil {
		t.Fatal(err)
	}
	groups, _, err := visualassets.DecodeCompoundGuardianArt(5, raw, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(dir, "guardian-fifth-damage-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows[1:] {
		num := func(at int) int {
			v, err := strconv.Atoi(row[at])
			if err != nil {
				t.Fatal(err)
			}
			return v
		}
		family, index, damage := num(0), num(1), uint16(num(2))
		var event FifthGuardianEvents
		var health, core, remaining int
		var flash bool
		if family == 0 {
			state, err := NewFifthMiddleGuardianState(&groups[0], -8)
			if err != nil {
				t.Fatal(err)
			}
			event = state.Damage(index, damage)
			health = state.Parts[index].Health
			flash = state.Flash
		} else {
			state, err := NewFifthFinalGuardianState(&groups[1])
			if err != nil {
				t.Fatal(err)
			}
			event = state.DamagePart(&groups[1], index, damage)
			health = state.Parts[index].Health
			core = int(state.CoreHealth)
			remaining = state.OuterRemaining
			flash = state.Flash
		}
		if health != num(3) || core != num(4) || remaining != num(5) || event.Score != num(6) || event.Cash != num(7) || event.Explosions != num(8) || flash != (num(10) != 0) {
			t.Fatalf("damage callback Go %d/%d/%d score %d cash %d explosions %d flash %v native %v", health, core, remaining, event.Score, event.Cash, event.Explosions, flash, row)
		}
	}
	t.Logf("Compared %d original fifth guardian damage cases.", len(rows)-1)
}
