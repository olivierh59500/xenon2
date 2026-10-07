package shopui

import (
	"fmt"
	"reflect"
	"testing"

	"xenon2/internal/engine"
)

func TestShopSourceCueDispatchBoundaries(t *testing.T) {
	equipment, money := engine.NewEquipment(), 5000
	s := testShop(&equipment, &money)
	for range 12 {
		s.Advance()
	}
	if got, want := s.TakeCues(), []Cue{{ID: "shop-sampled-effect-03", Channel: 0, Immediate: true}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("headphone direct dispatch: got%v want%v", got, want)
	}
	for value := uint16(1); value <= 3; value++ {
		s.Phase, s.Dialogue, s.Revealed = Selling, []Glyph{{Character: 'A', Word: true}}, 0
		s.random = func() uint16 { return value }
		s.Advance()
		want := []Cue{{ID: fmt.Sprintf("shop-sampled-effect-%02d", value-1), Channel: 0, Immediate: true}}
		if got := s.TakeCues(); !reflect.DeepEqual(got, want) {
			t.Fatalf("speech direct dispatch for choice%d: got%v want%v", value, got, want)
		}
	}
	s.Move(1, 0)
	if got, want := s.TakeCues(), []Cue{{ID: "shop-synthesized-effect-12", Channel: 2}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("navigation queued dispatch: got%v want%v", got, want)
	}
	if err := s.Confirm(); err != nil {
		t.Fatal(err)
	}
	if got, want := s.TakeCues(), []Cue{{ID: "shop-synthesized-effect-13", Channel: 2}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("confirmation queued dispatch: got%v want%v", got, want)
	}
	s.BeginEndingDot()
	if got, want := s.TakeCues(), []Cue{{ID: "shop-synthesized-effect-18", Channel: 2}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ending queued dispatch: got%v want%v", got, want)
	}
}
