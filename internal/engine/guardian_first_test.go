package engine

import "testing"

func TestFirstGuardianNativeTraceOptional(t *testing.T) {
	s := NewFirstGuardianState(30)
	maximum := 4607
	nativeCombatRows(t, "guardian-first-trace.csv", func(v []int64) {
		s.Health = uint16(v[3])
		maximum, _, _ = s.Advance(uint64(v[0]), int(v[1]), maximum)
		if maximum != int(v[2]) || s.BodyWorldY != int(v[4]) || s.BodyVelocity != int(v[5]) || s.BodyTimer != int(v[6]) || s.ExtensionPhase != int(v[7]) || s.SegmentBudget != int(v[8]) || int(s.Heading) != int(v[9]) || s.AngularVelocity != int32(v[10]) || s.AngularAcceleration != int32(v[11]) || s.EyeClock != uint8(v[12]) || s.Active != (v[13] != 0) {
			t.Fatalf("guardian pass %d: got %+v maximum=%d, native=%v", v[0], s, maximum, v)
		}
		if v[14] != 1000 && s.BodyCollision != (CollisionRect{Left: int(v[14]), Top: int(v[15]), Right: int(v[16]), Bottom: int(v[17])}) {
			t.Fatalf("guardian pass %d body collision differs: %+v native=%v", v[0], s.BodyCollision, v[14:])
		}
		if v[14] == 1000 && !s.BodyCollision.Empty() {
			t.Fatalf("inactive guardian pass %d must not collide in the playfield", v[0])
		}
	})
}

func TestFirstGuardianEyeRejectsBodyHits(t *testing.T) {
	s := NewFirstGuardianState(30)
	s.Active = true
	if damaged, _ := s.Strike(CollisionRect{Left: 140, Top: 40, Right: 140, Bottom: 40}, 3, 48); damaged || s.Health != 30 {
		t.Fatal("hitting the broad body outside the eye must not damage it")
	}
	if damaged, dead := s.Strike(CollisionRect{Left: 152, Top: 36, Right: 152, Bottom: 36}, 30, 48); !damaged || !dead || !s.Defeated {
		t.Fatal("the inclusive weak point must accept lethal damage")
	}
}
