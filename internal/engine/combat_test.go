package engine

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"xenon2/internal/visualassets"
)

func TestCombatInclusiveEdgesAndLethalThreshold(t *testing.T) {
	r := ActorCollisionRect(visualassets.CollisionBox{X: -5, Y: -4, Width: 10, Height: 8}, 20, 30)
	if r != (CollisionRect{15, 26, 24, 33}) || !r.Contains(24, 33) || !r.Intersects(CollisionRect{24, 33, 24, 33}) || r.Intersects(CollisionRect{25, 33, 25, 33}) {
		t.Fatalf("inclusive actor rectangle: %+v", r)
	}
	if damage := ApplyShieldDamage(8, 8, false, false); damage.Shield != 0 || !damage.Destroyed {
		t.Fatalf("zero shield must destroy the ship: %+v", damage)
	}
	if damage := ApplyShieldDamage(8, 3, true, false); damage.Shield != 7 || damage.Lost != 1 {
		t.Fatalf("protection uses floor division: %+v", damage)
	}
	if damage := ApplyShieldDamage(8, 127, false, true); damage.Shield != 8 || damage.Applied || damage.Destroyed {
		t.Fatalf("suppressed damage: %+v", damage)
	}
	if damage := ApplyEnemyDamage(1, 3); damage.Health != 65534 || !damage.Destroyed {
		t.Fatalf("lethal unsigned subtraction: %+v", damage)
	}
}

func TestCombatFormationAndQueuedFire(t *testing.T) {
	wave := visualassets.Wave{Count: 5, Spacing: 32, MotionBudget: 7}
	config, err := FormationMotion(wave, 2, 3, visualassets.ActorPart{Score: 42})
	if err != nil || config.Delay != 106 || config.StartXOffset != 0 || config.Budget != 7 {
		t.Fatalf("following formation: %+v error=%v", config, err)
	}
	wave.Spacing = 132
	config, err = FormationMotion(wave, 2, 3, visualassets.ActorPart{Score: 42})
	if err != nil || config.Delay != 42 || config.StartXOffset != 64 {
		t.Fatalf("side-by-side formation: %+v error=%v", config, err)
	}
	clock := NewFireCadence(NewEquipment())
	clock.QueueTrigger()
	for frame := 0; frame < 17; frame++ {
		if pulse := clock.TakePulse(); pulse != (frame%8 == 0) {
			t.Fatalf("firing frame %d: pulse=%v", frame, pulse)
		}
		if err := clock.Tick(true); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCombatSmallWeaponPatternAndNoFallback(t *testing.T) {
	storage := make([]SmallShot, 0, 2)
	shots, err := AppendSmallWeaponShots(storage, WeaponSlot{Item: ItemDoubleShot, Tier: 2}, 160, 176)
	if err != nil || len(shots) != 2 || shots[0].X != 155 || shots[1].X != 166 || shots[0].Y != 164 || shots[0].VelocityY != -9 || shots[0].Damage != 3 {
		t.Fatalf("double-shot pattern: %+v error=%v", shots, err)
	}
	if _, err := AppendSmallWeaponShots(storage, WeaponSlot{Item: ItemCannon}, 160, 176); err == nil {
		t.Fatal("a cannon must use its dedicated emitter")
	}
	shot := SmallShot{X: 311, Y: 10, VelocityX: 1}
	if shot.Advance(false) {
		t.Fatal("ordinary bullets leave at X = 312")
	}
}

func nativeCombatRows(t *testing.T, name string, visit func([]int64)) {
	t.Helper()
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local original traces")
	}
	file, err := os.Open(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 2 {
		t.Fatalf("native trace %s has no comparison rows", name)
	}
	for _, row := range rows[1:] {
		values := make([]int64, len(row))
		for i, field := range row {
			values[i], err = strconv.ParseInt(field, 10, 64)
			if err != nil {
				t.Fatal(err)
			}
		}
		visit(values)
	}
}

func TestCombatAimNativeTraceOptional(t *testing.T) {
	nativeCombatRows(t, "combat-aim-trace.csv", func(v []int64) {
		if direction := AimDirection(int(v[0]), int(v[1])); direction != uint8(v[2]) {
			t.Fatalf("aim %v: got %d", v, direction)
		}
	})
}

func TestCombatCollisionNativeTraceOptional(t *testing.T) {
	nativeCombatRows(t, "combat-collision-trace.csv", func(v []int64) {
		r := CollisionRect{0, 0, 9, 9}
		other := CollisionRect{int(v[0]), int(v[1]), int(v[0]) + 9, int(v[1]) + 9}
		if intersects := r.Intersects(other); intersects != (v[2] != 0) {
			t.Fatalf("collision %v: got %v", v, intersects)
		}
	})
}

func TestCombatShieldNativeTraceOptional(t *testing.T) {
	nativeCombatRows(t, "combat-shield-trace.csv", func(v []int64) {
		damage := ApplyShieldDamage(int(v[0]), int(v[1]), v[2] != 0, v[3] != 0)
		if damage.Shield != int(v[4]) || damage.Destroyed != (v[5] != 0) {
			t.Fatalf("shield %v: got %+v", v, damage)
		}
	})
}

func TestCombatFireClockNativeTraceOptional(t *testing.T) {
	var clock FireCadence
	nativeCombatRows(t, "combat-fire-clock-trace.csv", func(v []int64) {
		if v[2] == 0 {
			clock = FireCadence{Remaining: int(v[0]), Period: int(v[0]), Advance: int(v[1])}
		}
		if v[4] != 0 {
			clock.QueueTrigger()
		}
		if pulse := clock.TakePulse(); pulse != (v[5] != 0) {
			t.Fatalf("fire pulse %v: got %v", v, pulse)
		}
		if err := clock.Tick(v[3] != 0); err != nil {
			t.Fatal(err)
		}
		if clock.Remaining != int(v[6]) || clock.Pending != (v[7] != 0) {
			t.Fatalf("fire clock %v: got %+v", v, clock)
		}
	})
}

func TestCombatDirectionalNativeTraceOptional(t *testing.T) {
	var shot DirectionalProjectile
	nativeCombatRows(t, "combat-directional-trace.csv", func(v []int64) {
		if v[2] == 0 {
			shot = DirectionalProjectile{X: 160 << 16, Y: 100 << 16, Direction: uint8(v[0]), Speed: 6}
		}
		alive, err := shot.Advance(int(v[1]))
		if err != nil {
			t.Fatal(err)
		}
		if uint32(shot.X) != uint32(v[3]) || uint32(shot.Y) != uint32(v[4]) || alive != (v[5] != 0) {
			t.Fatalf("directional %v: got %+v alive=%v", v, shot, alive)
		}
	})
}

func TestCombatEnemyFireNativeTraceOptional(t *testing.T) {
	var fire EnemyFireState
	var random RandomState
	nativeCombatRows(t, "combat-enemy-fire-trace.csv", func(v []int64) {
		if v[5] == 0 {
			fire = EnemyFireState{Rate: uint8(v[1]), Accumulator: uint8(v[2])}
			random = RandomState{A: 0x12345678, B: 0x6abcdef1}
		}
		shot, fired, err := fire.Tick(random.Next, int(v[3]), int(v[4]))
		if err != nil {
			t.Fatal(err)
		}
		if fire.Accumulator != uint8(v[6]) || fired != (v[7] != 0) || fired && (shot.Direction != uint8(v[8]) || shot.Speed != int(v[9])) || random.A != uint32(v[10]) || random.B != uint32(v[11]) {
			t.Fatalf("enemy fire %v: got %+v shot=%+v fired=%v random=%+v", v, fire, shot, fired, random)
		}
	})
}

func TestCombatSmallEmittersNativeTraceOptional(t *testing.T) {
	common, err := os.ReadFile(filepath.Join(os.Getenv("XENON2_NATIVE_TRACE_DIR"), "..", "XenonII-unpacked.bin"))
	if os.Getenv("XENON2_NATIVE_TRACE_DIR") == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local original traces")
	}
	if err != nil {
		t.Fatal(err)
	}
	images := map[uint32]string{}
	for variant, root := range []int{0x27ce, 0x27c2, 0x27da, 0x27e6} {
		for tier := range 3 {
			at := root + tier*4
			address := uint32(common[at])<<24 | uint32(common[at+1])<<16 | uint32(common[at+2])<<8 | uint32(common[at+3])
			images[address] = smallShotImages[variant][tier]
		}
	}
	var shots []SmallShot
	nativeCombatRows(t, "combat-small-emit-trace.csv", func(v []int64) {
		if v[2] == 0 {
			shots, err = AppendSmallWeaponShots(shots[:0], WeaponSlot{Item: Item(v[0]), Tier: int(v[1])}, 160, 176)
			if err != nil {
				t.Fatal(err)
			}
		}
		shot := shots[len(shots)-1-int(v[2])]
		if shot.X != int(v[3]) || shot.Y != int(v[4]) || shot.VelocityX != int(v[5]) || shot.VelocityY != int(v[6]) || shot.Damage != uint16(v[7]) || shot.SpriteName != images[uint32(v[8])] {
			t.Fatalf("small emitter %v: got %+v", v, shot)
		}
	})
}

func TestCombatCashNativeTraceOptional(t *testing.T) {
	var cash CashMotion
	nativeCombatRows(t, "combat-cash-trace.csv", func(v []int64) {
		if v[5] == 0 {
			cash = CashMotion{X: int(v[1]), Y: int(v[2]), Mode: int(v[3]), Direction: uint16(v[4])}
		}
		alive := cash.Advance()
		if cash.X != int(v[6]) || cash.Y != int(v[7]) || cash.Mode != int(v[8]) || cash.Direction != uint16(v[9]) || alive != (v[10] != 0) {
			t.Fatalf("cash %v: got %+v alive=%v", v, cash, alive)
		}
	})
}

func TestCombatLaserBeamNativeTraceOptional(t *testing.T) {
	var beam LaserBeamState
	nativeCombatRows(t, "combat-laser-beam-trace.csv", func(v []int64) {
		if v[1] == 0 {
			beam = NewLaserBeamState(int(v[0]))
		}
		alive, rectangle, damage := beam.Advance(int(v[2]), int(v[3]), false)
		hasRect := !rectangle.Empty()
		if beam.X != int(v[4]) || beam.Y != int(v[5]) || beam.Length != int(v[6]) || alive != (v[7] != 0) || hasRect != (v[8] != 0) || hasRect && (rectangle.Left != int(v[9]) || rectangle.Top != int(v[10]) || rectangle.Right != int(v[11]) || rectangle.Bottom != int(v[12]) || damage != uint16(v[13])) {
			t.Fatalf("laser beam %v: got %+v alive=%v rectangle=%+v damage=%d", v, beam, alive, rectangle, damage)
		}
	})
}

func TestCombatLaserMountNativeTraceOptional(t *testing.T) {
	mount := NewLaserMountState()
	nativeCombatRows(t, "combat-laser-mount-trace.csv", func(v []int64) {
		fired := mount.Tick(v[1] != 0, v[2] != 0)
		if mount.Cooldown != int(v[3]) || fired != (v[4] != 0) {
			t.Fatalf("laser mount %v: got %+v emitted=%v", v, mount, fired)
		}
	})
}
