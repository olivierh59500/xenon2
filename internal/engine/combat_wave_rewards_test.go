package engine

import "testing"

func TestCombatWaveBonusSharesAndInvalidatesPeriod(t *testing.T) {
	cache := NewWaveBonusCache()
	cache.BeginPass(4)
	first, second := cache.Register(false, 2), cache.Register(false, 3)
	if first != second || cache.NextID != 4 || cache.Entries[0].Remaining != 5 {
		t.Fatalf("shared period: %+v", cache)
	}
	if cache.Defeat(first) {
		t.Fatal("one member cannot award the entire bucket")
	}
	cache.Escape(second)
	if cache.Defeat(first) || cache.Entries[0] != (WaveBonusEntry{}) {
		t.Fatal("an escaped member must invalidate related waves")
	}
}

func TestCombatWaveBonusNativeTraceOptional(t *testing.T) {
	cache := NewWaveBonusCache()
	nativeCombatRows(t, "combat-wave-cache-trace.csv", func(v []int64) {
		result := 0
		switch v[0] {
		case 0:
			cache.BeginPass(uint64(v[1]))
		case 1:
			result = int(cache.Register(v[1] != 0, int(v[2])))
		case 2:
			if cache.Defeat(uint16(v[1])) {
				result = 1
			}
		case 3:
			cache.Escape(uint16(v[1]))
		}
		if result != int(v[3]) || cache.NextID != uint16(v[4]) || cache.NormalID != uint16(v[5]) || cache.HeavyID != uint16(v[6]) {
			t.Fatalf("wave cache %v: got %+v result=%d", v, cache, result)
		}
		for i, entry := range cache.Entries {
			if entry.ID != uint16(v[7+i*2]) || entry.Remaining != uint16(v[8+i*2]) {
				t.Fatalf("wave cache %v: entry %d=%+v", v, i, entry)
			}
		}
	})
}

func TestCombatScrollNativeTraceOptional(t *testing.T) {
	nativeCombatRows(t, "combat-scroll-trace.csv", func(v []int64) {
		scroll := ScrollState{Y: int(v[0]), Minimum: int(v[1]), Maximum: int(v[2]), DeviationPasses: int(v[3])}
		scroll.Advance(int(v[4]), int(v[5]), v[6] != 0)
		if scroll.Y != int(v[7]) || scroll.Maximum != int(v[8]) || scroll.DeviationPasses != int(v[9]) || scroll.ActualStep != int(v[10]) {
			t.Fatalf("scroll %v: got %+v", v, scroll)
		}
	})
}

func TestFourthChildBonusNativeTraceOptional(t *testing.T) {
	cases := 0
	nativeCombatRows(t, "fourth-child-bonus-trace.csv", func(v []int64) {
		cache := WaveBonusCache{NextID: uint16(v[1])}
		for i := range cache.Entries {
			if int(v[2])&(1<<i) != 0 {
				cache.Entries[i] = WaveBonusEntry{ID: uint16(100 + i), Remaining: uint16(2 + i)}
			}
		}
		token := cache.RegisterIndependent()
		if token != uint16(v[3]) || cache.NextID != uint16(v[4]) {
			t.Fatalf("fresh child token %v: %+v token=%d", v, cache, token)
		}
		for i, entry := range cache.Entries {
			if entry.ID != uint16(v[5+i*2]) || entry.Remaining != uint16(v[6+i*2]) {
				t.Fatalf("child bucket %v: %d=%+v", v, i, entry)
			}
		}
		cache.Escape(token)
		for i, entry := range cache.Entries {
			if entry.ID != uint16(v[22+i*2]) || entry.Remaining != uint16(v[23+i*2]) {
				t.Fatalf("escaped child bucket %v: %d=%+v", v, i, entry)
			}
		}
		cases++
	})
	if cases != 28 {
		t.Fatalf("child bonus coverage: %d", cases)
	}
}
