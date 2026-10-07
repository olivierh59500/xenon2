package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"xenon2/internal/visualassets"
)

func TestWeaponRuntimeNativeFamiliesOptional(t *testing.T) {
	root := os.Getenv("XENON2_RUNTIME_ASSET_DIR")
	if root == "" {
		t.Skip("set XENON2_RUNTIME_ASSET_DIR to exported original artwork")
	}
	data, err := os.ReadFile(filepath.Join(root, "common-actors.json"))
	if err != nil {
		t.Fatal(err)
	}
	common := &visualassets.SpriteAtlas{}
	if err := json.Unmarshal(data, common); err != nil {
		t.Fatal(err)
	}
	for _, item := range []Item{ItemForwardShot, ItemDoubleShot, ItemRearShot, ItemSideShot, ItemCannon, ItemMissileLauncher, ItemLaser, ItemDrone, ItemFlamer, ItemElectroBall, ItemMineSmall, ItemMineLarge, ItemBomb, ItemHomingMissile} {
		t.Run(string(rune('A'+item)), func(t *testing.T) {
			runtime, err := NewWeaponRuntime(common)
			if err != nil {
				t.Fatal(err)
			}
			equipment := NewEquipment()
			if item != ItemForwardShot {
				equipment.ApplyItem(item)
			}
			random := NewRandomState()
			ids := 0
			hits := 0
			ctx := WeaponContext{Equipment: &equipment, ShipX: 160, ShipY: 100, PreviousShipX: 159, PreviousShipY: 101, TrailX: 155, TrailY: 105, Held: true, Pulse: true, NextRandom: random.Next, NextID: func() int { ids++; return ids }, Targets: []WeaponTarget{{ID: 1, Active: true, ResourceTag: 200, Bounds: CollisionRect{Left: 200, Top: 20, Right: 220, Bottom: 40}}}, HitPoint: func(x, y int, d uint16) bool { hits++; return false }, HitRect: func(r CollisionRect, d uint16, multiple bool) bool { hits++; return false }}
			for frame := 0; frame < 80; frame++ {
				ctx.Held = frame < 45
				ctx.Pulse = frame%8 == 0
				ctx.Motion.Up = frame < 30
				ctx.Motion.Right = frame >= 30 && frame < 40
				if err := runtime.AdvanceEquipment(ctx); err != nil {
					t.Fatal(err)
				}
				if err := runtime.AdvanceProjectiles(ctx); err != nil {
					t.Fatal(err)
				}
				if err := runtime.AdvanceSparks(ctx); err != nil {
					t.Fatal(err)
				}
				runtime.Compact()
			}
			if item != ItemDrone && item != ItemElectroBall && item != ItemMineSmall && item != ItemMineLarge && ids == 0 {
				t.Fatalf("item %d never emitted", item)
			}
			state := runtime.RenderState(nil)
			for _, v := range state {
				if v.Kind != "laser" && v.Kind != "spark" && v.Sprite == "" {
					t.Fatalf("item %d has unnamed artwork: %+v", item, v)
				}
			}
			_ = hits
		})
	}
}

func TestWeaponRuntimeCannonLauncherNativeAnimationOptional(t *testing.T) {
	root := os.Getenv("XENON2_RUNTIME_ASSET_DIR")
	if root == "" {
		t.Skip("set XENON2_RUNTIME_ASSET_DIR to exported original artwork")
	}
	data, err := os.ReadFile(filepath.Join(root, "common-actors.json"))
	if err != nil {
		t.Fatal(err)
	}
	common := &visualassets.SpriteAtlas{}
	if err := json.Unmarshal(data, common); err != nil {
		t.Fatal(err)
	}
	var runtime *WeaponRuntime
	var equipment Equipment
	nativeCombatRows(t, "combat-cannon-mount-trace.csv", func(v []int64) {
		item := Item(v[0])
		if v[1] == 0 {
			runtime, err = NewWeaponRuntime(common)
			if err != nil {
				t.Fatal(err)
			}
			equipment = NewEquipment()
			equipment.ApplyItem(item)
		}
		before := len(runtime.projectiles)
		ctx := WeaponContext{Equipment: &equipment, ShipX: 160, ShipY: 176, TrailX: 160, TrailY: 176, Pulse: v[2] != 0, Materializing: v[3] != 0, SkipSmallWeapons: true}
		if err := runtime.AdvanceEquipment(ctx); err != nil {
			t.Fatal(err)
		}
		if len(runtime.projectiles)-before != int(v[6]) {
			t.Fatalf("runtime emissions differ: %v", v)
		}
		idle, fire := "cannon-idle", "cannon-fire"
		if item == ItemMissileLauncher {
			idle, fire = "launcher-idle", "launcher-fire"
		}
		expected := runtime.animations[idle].Animation.Frames[0].Sprite
		if v[4] > 0 {
			expected = runtime.animations[fire].Animation.Frames[v[4]-1].Sprite
		}
		actual := runtime.mounts[1].Animation.Sprite(runtime.mounts[1].AnimationData.Animation)
		if actual != expected {
			t.Fatalf("runtime mount %v: sprite=%q expected=%q", v, actual, expected)
		}
	})
}

func TestWeaponRuntimeSparksHaveIndependentPoolAndPhase(t *testing.T) {
	runtime, err := NewWeaponRuntime(&visualassets.SpriteAtlas{})
	if err != nil {
		t.Fatal(err)
	}
	equipment := NewEquipment()
	equipment.ApplyItem(ItemDrone)
	allocated := 0
	ctx := WeaponContext{Equipment: &equipment, ShipX: 160, ShipY: 100, TrailX: 150, TrailY: 80, Held: true, NextID: func() int { allocated++; return allocated }}
	if err := runtime.AdvanceEquipment(ctx); err != nil {
		t.Fatal(err)
	}
	if allocated != 0 || len(runtime.ProjectileIDs(nil)) != 0 {
		t.Fatal("drone consumed the actor pool")
	}
	before := runtime.RenderState(nil)
	if err := runtime.AdvanceProjectiles(ctx); err != nil {
		t.Fatal(err)
	}
	after := runtime.RenderState(nil)
	if len(before) != len(after) {
		t.Fatal("ordinary phase removed a spark")
	}
	for i := range before {
		if before[i].X != after[i].X || before[i].Y != after[i].Y {
			t.Fatal("ordinary phase moved a spark")
		}
	}
	if err := runtime.AdvanceSparks(ctx); err != nil {
		t.Fatal(err)
	}
	after = runtime.RenderState(nil)
	if len(after) != 4 || after[0].X == before[0].X && after[0].Y == before[0].Y {
		t.Fatal("dedicated phase did not move four sparks")
	}
}

func TestWeaponRuntimeMergedIDsAndCapacity(t *testing.T) {
	runtime, err := NewWeaponRuntime(&visualassets.SpriteAtlas{})
	if err != nil {
		t.Fatal(err)
	}
	equipment := NewEquipment()
	random := NewRandomState()
	id := 100
	c := WeaponContext{Equipment: &equipment, ShipX: 160, ShipY: 100, Held: true, Pulse: true, NextRandom: random.Next, NextID: func() int { id++; return id }}
	if err := runtime.AdvanceEquipment(c); err != nil {
		t.Fatal(err)
	}
	ids := runtime.ProjectileIDs(nil)
	if len(ids) != 1 || ids[0] != 101 {
		t.Fatalf("merged creation IDs: %v", ids)
	}
	c.HitPoint = func(x, y int, damage uint16) bool { return true }
	if err := runtime.AdvanceProjectile(101, c); err != nil {
		t.Fatal(err)
	}
	runtime.Compact()
	if len(runtime.ProjectileIDs(nil)) != 0 {
		t.Fatal("merged projectile removal did not compact")
	}
}

func BenchmarkWeaponRuntimeEquipment(b *testing.B) {
	runtime, _ := NewWeaponRuntime(&visualassets.SpriteAtlas{})
	equipment := NewEquipment()
	random := NewRandomState()
	c := WeaponContext{Equipment: &equipment, ShipX: 160, ShipY: 100, NextRandom: random.Next}
	if err := runtime.AdvanceEquipment(c); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := runtime.AdvanceEquipment(c); err != nil {
			b.Fatal(err)
		}
	}
}
