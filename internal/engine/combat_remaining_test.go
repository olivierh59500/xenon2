package engine

import "testing"

func TestCombatElectroNativeTraceOptional(t *testing.T) {
	ball := ElectroBallState{X: 160, Y: 201}
	nativeCombatRows(t, "combat-electro-trace.csv", func(v []int64) {
		query := ball.Advance(int(v[1]), int(v[2]), int(v[3]), int(v[4]), int(v[5]), int(v[6]), v[7] != 0, v[8] != 0)
		if v[8] != 0 {
			return
		} // The original materialization renderer changes the displayed anchor separately.
		if ball.X != int(v[9]) || ball.Y != int(v[10]) || ball.Mode != int(v[11]) || query != (v[12] != 0) {
			t.Fatalf("electro %v: got %+v query=%v", v, ball, query)
		}
	})
}

func TestCombatMineNativeTraceOptional(t *testing.T) {
	var mine MineState
	var context MineContext
	nativeCombatRows(t, "combat-mine-trace.csv", func(v []int64) {
		if v[1] == 0 {
			mine = MineState{Tier: int(v[0])}
			context = MineContext{LastX: -100}
		}
		bits := int(v[3])
		spawn, _, _, _ := mine.Advance(160, 100, 40, 60, MotionInput{Up: bits&1 != 0, Down: bits&2 != 0, Left: bits&4 != 0, Right: bits&8 != 0}, v[2] != 0, v[4] != 0, false, &context)
		if v[4] != 0 {
			return
		}
		if mine.X != int(v[5]) || mine.Y != int(v[6]) || mine.Mode != int(v[7]) || context.Count != int(v[8]) || context.LastX != int(v[9]) || context.LastY != int(v[10]) || (spawn != nil) != (v[11] != 0) {
			t.Fatalf("mine %v: got %+v context=%+v spawned=%v", v, mine, context, spawn != nil)
		}
	})
}

func TestCombatBombMountNativeTraceOptional(t *testing.T) {
	mount := NewBombMountState()
	nativeCombatRows(t, "combat-bomb-mount-trace.csv", func(v []int64) {
		created := mount.Tick(v[1] != 0, v[2] != 0, false)
		if mount.Cooldown != int(v[3]) || created != (v[4] != 0) {
			t.Fatalf("bomb mount %v: got %+v created=%v", v, mount, created)
		}
	})
}

func TestCombatBombNativeTraceOptional(t *testing.T) {
	bomb := NewBombState(160, 100)
	nativeCombatRows(t, "combat-bomb-trace.csv", func(v []int64) {
		alive, area, damage := bomb.Advance(false)
		if bomb.X != int(v[1]) || bomb.Y != int(v[2]) || bomb.Timer != int(v[3]) || alive != (v[4] != 0) || !area.Empty() != (v[5] != 0) || !area.Empty() && (area.Left != int(v[6]) || area.Top != int(v[7]) || area.Right != int(v[8]) || area.Bottom != int(v[9]) || damage != uint16(v[10])) {
			t.Fatalf("bomb %v: got %+v alive=%v area=%+v damage=%d", v, bomb, alive, area, damage)
		}
	})
}

func TestCombatHomingNativeTraceOptional(t *testing.T) {
	var missile HomingMissileState
	targets := []WeaponTarget{{ID: 1, ResourceTag: 200, Active: true, Bounds: CollisionRect{Left: 200, Top: 20, Right: 220, Bottom: 40}}}
	nativeCombatRows(t, "combat-homing-trace.csv", func(v []int64) {
		if v[1] == 0 {
			missile = NewHomingMissileState(160, 100, uint8(v[0]))
			missile.TargetID = 1
		}
		alive := missile.Advance(targets, nil, false)
		if missile.X != int(v[2]) || missile.Y != int(v[3]) || missile.TurnTimer != int(v[4]) || missile.Direction != uint8(v[5]) || alive != (v[6] != 0) {
			t.Fatalf("homing %v: got %+v alive=%v", v, missile, alive)
		}
	})
}

func TestCombatHomingTargetNativeTraceOptional(t *testing.T) {
	tags := [10]int{100, 200, 204, 184, 80, 84, 208, 212, 216, 220}
	nativeCombatRows(t, "combat-homing-target-trace.csv", func(v []int64) {
		var storage [10]WeaponTarget
		count, test := int(v[3]), int(v[0])
		for i := range count {
			x, y := 100+i*8, 20+i*7
			if i == 2 && test%2 == 0 {
				x = -50
			}
			if i == 6 && test%3 == 0 {
				y = 220
			}
			storage[i] = WeaponTarget{ID: i + 1, ResourceTag: tags[i], Active: true, Bounds: CollisionRect{Left: x, Top: y, Right: x + 15, Bottom: y + 15}}
		}
		random := RandomState{A: uint32(v[1]), B: uint32(v[2])}
		target, found := SelectHomingTarget(storage[:count], random.Next)
		id := 0
		if found {
			id = target.ID
		}
		if id != int(v[4]) || random.A != uint32(v[5]) || random.B != uint32(v[6]) {
			t.Fatalf("homing selector %v: selected=%d random=%+v", v, id, random)
		}
	})
}
