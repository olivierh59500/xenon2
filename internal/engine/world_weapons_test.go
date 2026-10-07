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

func TestWeaponRuntimeGuardedNativeTraceOptional(t *testing.T) {
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
	nativeCombatRows(t, "combat-guarded-weapon-trace.csv", func(v []int64) {
		if v[1] == 0 {
			runtime, err = NewWeaponRuntime(common)
			if err != nil {
				t.Fatal(err)
			}
			equipment = NewEquipment()
			equipment.ApplyItem(Item(v[0]))
		}
		ctx := WeaponContext{Equipment: &equipment, ShipX: int(v[2]), ShipY: int(v[3]), PreviousShipX: int(v[4]), PreviousShipY: int(v[5]), TrailX: int(v[6]), TrailY: int(v[7]), Held: v[8] != 0, Materializing: v[9] != 0, MaterializationFrames: int(v[9]), ShipCenterX: int(v[10]), ShipCenterY: int(v[11])}
		if err := runtime.AdvanceEquipment(ctx); err != nil {
			t.Fatal(err)
		}
		mount := runtime.mounts[5]
		mode := mount.Mine.Mode
		if Item(v[0]) == ItemElectroBall {
			mode = mount.Electro.Mode
		}
		if mount.X != int(v[12]) || mount.Y != int(v[13]) || mode != int(v[14]) || mount.Visible != (v[15] != 0) {
			t.Fatalf("guarded weapon %v: mount=%+v", v, mount)
		}
	})
}

func TestWeaponRuntimeCannonSupportNativeTraceOptional(t *testing.T) {
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
	runtime, err := NewWeaponRuntime(common)
	if err != nil {
		t.Fatal(err)
	}
	equipment := NewEquipment()
	equipment.ApplyItem(ItemCannon)
	nativeCombatRows(t, "combat-cannon-support-trace.csv", func(v []int64) {
		ctx := WeaponContext{Equipment: &equipment, ShipX: int(v[1]), ShipY: int(v[2]), Materializing: v[3] != 0, MaterializationFrames: int(v[3]), ShipCenterX: int(v[4]), ShipCenterY: int(v[5])}
		if err := runtime.AdvanceEquipment(ctx); err != nil {
			t.Fatal(err)
		}
		m := runtime.mounts[1]
		if m.SupportAnimation.Frame != int(v[6]) || m.SupportX != int(v[7]) || m.SupportY != int(v[8]) || m.SupportVisible != (v[9] != 0) {
			t.Fatalf("cannon support %v: frame=%d position=%d,%d visible=%v", v, m.SupportAnimation.Frame, m.SupportX, m.SupportY, m.SupportVisible)
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

func TestWeaponRuntimeMinePowerupPreservesCursorAndDroppedPower(t *testing.T) {
	common := &visualassets.SpriteAtlas{Animations: []visualassets.NamedActorAnimation{{ID: "mine-small-active", Ending: "loop", Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "a", Duration: 2}, {Sprite: "b", Duration: 2}}}}}}
	runtime, err := NewWeaponRuntime(common)
	if err != nil {
		t.Fatal(err)
	}
	equipment := NewEquipment()
	equipment.ApplyItem(ItemMineSmall)
	ctx := WeaponContext{Equipment: &equipment, ShipX: 160, ShipY: 100, Held: true}
	if err := runtime.AdvanceEquipment(ctx); err != nil {
		t.Fatal(err)
	}
	runtime.mounts[5].Mine.X, runtime.mounts[5].Mine.Y, runtime.mounts[5].Mine.Mode = 80, 40, 2
	equipment.ApplyItem(ItemMineSmall)
	if err := runtime.AdvanceEquipment(ctx); err != nil {
		t.Fatal(err)
	}
	if len(runtime.projectiles) != 1 {
		t.Fatal("upgraded cursor did not plant")
	}
	mine := runtime.projectiles[0]
	if mine.Mine.X != 80 || mine.Mine.Y != 40 || mine.Mine.Tier != 1 || mine.Render.Sprite != runtime.mounts[5].Animation.Sprite(runtime.mounts[5].AnimationData.Animation) {
		t.Fatalf("upgraded mine lost cursor, power or current animation: %+v", mine)
	}
	runtime.mounts[5].Mine.Tier = 0
	ctx.Held = false
	var area CollisionRect
	var damage uint16
	ctx.HitRect = func(r CollisionRect, d uint16, all bool) bool { area, damage = r, d; return false }
	random := NewRandomState()
	ctx.NextRandom = random.Next
	for range 4 {
		if err := runtime.AdvanceProjectile(mine.Render.ID, ctx); err != nil {
			t.Fatal(err)
		}
	}
	if area.Right-area.Left != 47 || damage != 6 {
		t.Fatalf("planted mine did not retain upgraded damage: %+v %d", area, damage)
	}
}

func TestWeaponRuntimeSmallShotPreservesPendingSound(t *testing.T) {
	runtime, err := NewWeaponRuntime(&visualassets.SpriteAtlas{})
	if err != nil {
		t.Fatal(err)
	}
	equipment := NewEquipment()
	requests := [4]string{"", "", "synthesized-effect-17", ""}
	ctx := WeaponContext{Equipment: &equipment, ShipX: 160, ShipY: 100, Pulse: true, SoundVoiceIfEmpty: func(voice int, effect string) {
		if requests[voice] == "" {
			requests[voice] = effect
		}
	}}
	if err := runtime.AdvanceEquipment(ctx); err != nil {
		t.Fatal(err)
	}
	if requests[2] != "synthesized-effect-17" {
		t.Fatal("ordinary fire replaced the pending death sound")
	}
	requests[2] = ""
	if err := runtime.AdvanceEquipment(ctx); err != nil {
		t.Fatal(err)
	}
	if requests[2] != "sampled-effect-09" {
		t.Fatal("ordinary fire did not queue its sound")
	}
}

func TestWeaponRuntimeBombEffectsPrecedeDamage(t *testing.T) {
	runtime, err := NewWeaponRuntime(&visualassets.SpriteAtlas{})
	if err != nil {
		t.Fatal(err)
	}
	projectile := runtime.add("bomb", "", 160, 100, 0)
	projectile.Bomb = BombState{X: 160, Y: 100, Timer: 1}
	id := projectile.Render.ID
	random := NewRandomState()
	draws, effects, immediate, hits := 0, 0, false, 0
	ctx := WeaponContext{NextRandom: func() uint32 { draws++; return random.Next() }, SoundVoice: func(voice int, effect string) { effects++ }, ImmediateSoundVoice: func(voice int, effect string) { immediate = voice == 0 && effect == "sampled-effect-03" }}
	ctx.HitRect = func(area CollisionRect, damage uint16, all bool) bool {
		if draws != 8 || effects != 2 || !immediate || len(runtime.projectiles) != 5 {
			t.Fatal("bomb damage happened before its source-ordered effects")
		}
		hits++
		return true
	}
	if err := runtime.AdvanceProjectile(id, ctx); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatal("bomb did not query its damage area")
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
