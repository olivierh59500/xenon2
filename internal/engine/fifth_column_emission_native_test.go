package engine

import "testing"

type fifthNativeColumnBirth struct {
	part  int
	laser FifthGuardianLaser
}

// These records execute the inline factories at 0x56144 and 0x5679e, then
// inspect the initialized tag-272 record. Other factory calls do not count.
func TestFifthColumnEmissionMatchesActualOriginalFactoriesOptional(t *testing.T) {
	groups := fifthGuardianEventTraceArt(t)
	births := make(map[[2]int][]fifthNativeColumnBirth)
	var totals [2]int
	nativeCombatRows(t, "fifth-column-births-trace.csv", func(v []int64) {
		if len(v) != 14 {
			t.Fatalf("native column birth has %d fields, want 14", len(v))
		}
		family, frame, part := int(v[0]), int(v[1]), int(v[2])
		if family < 0 || family > 1 || v[3] != 272 || v[4] != 0x56940 || v[5] != 0x56a10 || v[8] != 0 || v[10] != 0xb9c {
			t.Fatalf("record is not an initialized original projectile-list column: %v", v)
		}
		if part < 0 || part >= len(groups[family].Components) {
			t.Fatal("original column has no guardian owner")
		}
		behavior := groups[family].Components[part].Behavior
		if family == 0 && behavior != "middle-body-controller" || family == 1 && behavior != "final-side-turret" {
			t.Fatalf("original column came from unexpected component %s", behavior)
		}
		laser := FifthGuardianLaser{X: int(v[6]), Y: int(v[7]), Speed: int(v[9])}
		births[[2]int{family, frame}] = append(births[[2]int{family, frame}], fifthNativeColumnBirth{part: part, laser: laser})
		totals[family]++
	})
	if totals != ([2]int{49, 175}) {
		t.Fatalf("incomplete original column births: %v", totals)
	}
	middle, err := NewFifthMiddleGuardianState(&groups[0], -8)
	if err != nil {
		t.Fatal(err)
	}
	final, err := NewFifthFinalGuardianState(&groups[1])
	if err != nil {
		t.Fatal(err)
	}
	random := NewRandomState()
	rows, emitted := 0, 0
	nativeCombatRows(t, "fifth-column-frames-trace.csv", func(v []int64) {
		if len(v) != 7 {
			t.Fatalf("native column frame has %d fields, want 7", len(v))
		}
		family, frame := int(v[0]), int(v[1])
		var event FifthGuardianEvents
		if family == 0 {
			if frame != rows {
				t.Fatal("middle frame order differs")
			}
			scroll := 0x900
			if frame < 300 {
				scroll = 0x920
			} else if frame < 700 {
				scroll = 0x910
			}
			delta := 1
			if frame%11 == 0 {
				delta = 0
			}
			event = middle.Advance(&groups[0], scroll, delta, scroll, frame*3%320, 96+frame%80, &random)
		} else {
			if family != 1 || frame != rows-1500 {
				t.Fatal("final frame order differs")
			}
			if frame == 0 {
				random = NewRandomState()
			}
			scroll, delta := max(0, 416-frame), 0
			if scroll > 0 {
				delta = 1
			}
			event = final.Advance(&groups[1], scroll, delta, 1, 416, frame, frame*3%320, 96+frame%80, &random).FifthGuardianEvents
		}
		want := births[[2]int{family, frame}]
		if len(event.Lasers) != int(v[2]) || len(event.Lasers) != len(want) || event.Sound2 != int(v[3]) || event.Sound1 != int(v[4]) || random.A != uint32(v[5]) || random.B != uint32(v[6]) {
			t.Fatalf("family%d frame%d column/sound/random boundary differs: Go%+v native%v", family, frame, event, v)
		}
		for index, laser := range event.Lasers {
			if laser != want[index].laser {
				t.Fatalf("family%d frame%d emission%d differs: got%+v want%+v", family, frame, index, laser, want[index])
			}
			emitted++
		}
		rows++
	})
	if rows != 2700 || emitted != 224 {
		t.Fatalf("incomplete native column coverage: %d frames, %d emissions", rows, emitted)
	}
	t.Logf("%d original frames and %d actual column births, including four final side turrets", rows, emitted)
}
