package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

func TestFourthMiddleGuardianNativeTraceOptional(t *testing.T) {
	groups, atlas, sine := fourthGuardianTestArt(t)
	art := &groups[0]
	state, err := NewFourthMiddleGuardian(art, 2480, nil)
	if err != nil {
		t.Fatal(err)
	}
	random := NewRandomState()
	maximum, cases := 0, 0
	nativeCombatRows(t, "guardian-fourth-middle-trace.csv", func(v []int64) {
		index := int(v[1])
		lookup := func(name string) visualassets.CollisionBox {
			for _, region := range atlas.Sprites {
				if region.Name == name {
					return *region.Collision
				}
			}
			t.Fatalf("missing fourth part %s", name)
			return visualassets.CollisionBox{}
		}
		event, err := state.AdvancePart(index, art, FourthGuardianInput{Frame: uint64(v[0]), ScrollY: int(v[2]), ScrollDelta: int(v[3]), PlayerX: int(v[4]), PlayerY: int(v[5]), MaximumScrollY: maximum}, &sine, lookup, random.Next)
		if err != nil {
			t.Fatal(err)
		}
		maximum = event.MaximumScrollY
		part := state.Parts[index]
		angle := uint32(part.Arc.AngleFixed)
		nativeAngle := angle<<16 | angle>>16
		expectedSprite := atlas.SourceSpriteNames[int(v[17])]
		if maximum != int(v[6]) || part.Arc.X != int32(v[7]) || part.Arc.Y != int32(v[8]) || nativeAngle != uint32(v[9]) || part.Arc.AngularVelocity != int32(v[10]) || part.Arc.AngularAcceleration != int32(v[11]) || part.Arc.Budget != int(v[12]) || part.Counter != int16(v[13]) || part.FireAccumulator != uint8(v[14]) || index == 0 && part.Direction != int16(v[15]) || state.Sprite(index, art) != expectedSprite || part.Collision.Left != int(v[18]) || part.Collision.Top != int(v[19]) || part.Collision.Right != int(v[20]) || part.Collision.Bottom != int(v[21]) || event.ShotCount != int(v[22]) || random.A != uint32(v[27]) || random.B != uint32(v[28]) {
			t.Fatalf("fourth middle %v: part=%+v sprite=%s expected=%s event=%+v random=%+v", v, part, state.Sprite(index, art), expectedSprite, event, random)
		}
		if event.ShotCount != 0 {
			shot := event.Shots[event.ShotCount-1]
			if shot.X != int(v[23]) || shot.Y != int(v[24]) || shot.Direction != uint8(v[25]) || shot.Speed != int(v[26]) {
				t.Fatalf("fourth middle shot %v: %+v", v, shot)
			}
		}
		cases++
	})
	if cases != 1200*20 {
		t.Fatalf("fourth middle coverage: %d states", cases)
	}
}

func TestFourthMiddleGuardianDamageNativeTraceOptional(t *testing.T) {
	groups, _, _ := fourthGuardianTestArt(t)
	state, err := NewFourthMiddleGuardian(&groups[0], 2480, nil)
	if err != nil {
		t.Fatal(err)
	}
	score := 0
	nativeCombatRows(t, "guardian-fourth-middle-damage-trace.csv", func(v []int64) {
		index := int(v[1])
		part := &state.Parts[index]
		part.Collision = CollisionRect{Left: int(v[4]), Top: int(v[5]), Right: int(v[6]), Bottom: int(v[7])}
		x, y := (int(v[4])+int(v[6]))/2, (int(v[5])+int(v[7]))/2
		if v[3] != 0 {
			y = int(v[5])
		}
		event := state.Strike(index, CollisionRect{Left: x, Top: y, Right: x, Bottom: y}, uint16(v[2]))
		score += event.Score
		alive := 0
		for _, p := range state.Parts {
			if !p.Disabled {
				alive++
			}
		}
		if state.Parts[4].Health != uint16(v[8]) || state.Parts[index].Health != uint16(v[9]) || state.OuterTargets != int(v[10]) || score != int(v[11]) || event.CashPairs != int(v[12]) || event.ExplosionCount != int(v[13]) || alive != int(v[17]) {
			t.Fatalf("fourth middle damage %v: core=%d target=%d outer=%d score=%d alive=%d event=%+v", v, state.Parts[4].Health, state.Parts[index].Health, state.OuterTargets, score, alive, event)
		}
		if event.Defeated && (event.MinimumScrollY != int(v[14]) || event.MaximumScrollY != int(v[15]) || event.ForcedScrollY != int(v[16])) {
			t.Fatalf("middle defeat scroll %v: %+v", v, event)
		}
	})
}
