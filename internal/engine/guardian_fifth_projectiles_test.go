package engine

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"xenon2/internal/visualassets"
)

func TestFifthGuardianProjectileNativeTraceOptional(t *testing.T) {
	dir := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if dir == "" {
		t.Skip("local fifth projectile reference not supplied")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(dir), "imported", "04820138.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	terrain, err := visualassets.DecodeTerrain(raw)
	if err != nil {
		t.Fatal(err)
	}
	groups, atlas, err := visualassets.DecodeCompoundGuardianArt(5, raw, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(dir, "guardian-fifth-projectile-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	var laser FifthLaserColumnState
	var seeker FifthSeekingState
	for _, row := range rows[1:] {
		num := func(at int) int {
			v, err := strconv.Atoi(row[at])
			if err != nil {
				t.Fatal(err)
			}
			return v
		}
		family, variant, frame := num(0), num(1), num(2)
		group := &groups[0]
		if family == 2 {
			group = &groups[1]
		}
		if frame == 0 {
			laser = FifthLaserColumnState{X: 150, Y: 40, Speed: 10, Active: true}
			if variant == 1 {
				laser.Speed = -10
			}
			if family != 0 {
				clip := group.Components[0].HeadingAnimations[variant]
				seeker = FifthSeekingState{X: 150, Y: 40, Heading: uint8(variant), Active: true, Animation: NewAnimation(clip), Sprite: clip.Frames[0].Sprite}
			}
		}
		delta := 1
		if frame%5 == 0 {
			delta = -1
		}
		if family == 0 {
			laser.Advance(delta, group.MotionParameters["body_laser_speed"])
			if laser.X != num(3) || laser.Y != num(4) || laser.Length != num(5) || laser.Active != (num(9) != 0) {
				t.Fatalf("column Go %+v native %v", laser, row)
			}
			if laser.Active && (laser.Collision.Left != num(10) || laser.Collision.Top != num(11) || laser.Collision.Right != num(12) || laser.Collision.Bottom != num(13)) {
				t.Fatalf("column collision Go %+v native %v", laser, row)
			}
		} else {
			x, y := (frame*7+90)%320, 96+frame%80
			if family == 1 {
				seeker.AdvanceSide(group, x, y, delta)
			} else {
				outer := 18
				if frame >= 70 {
					outer = 0
				}
				seeker.AdvanceMouth(group, x, y, outer)
			}
			if seeker.X != num(3) || seeker.Y != num(4) || seeker.Clock != num(5) || seeker.Heading != uint8(num(6)) || seeker.Animation.Remaining != num(7) || seeker.Sprite != atlas.SourceSpriteNames[num(8)] || seeker.Active != (num(9) != 0) {
				t.Fatalf("seeker Go %+v native %v", seeker, row)
			}
		}
	}
	t.Logf("Compared %d original fifth guardian projectile states.", len(rows)-1)
}

func TestFifthSeekingContactNativeTraceOptional(t *testing.T) {
	dir := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if dir == "" {
		t.Skip("local fifth seeker contact reference not supplied")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(dir), "imported", "04820138.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	terrain, err := visualassets.DecodeTerrain(raw)
	if err != nil {
		t.Fatal(err)
	}
	groups, atlas, err := visualassets.DecodeCompoundGuardianArt(5, raw, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	regions := map[string]visualassets.SpriteRegion{}
	for _, region := range atlas.Sprites {
		regions[region.Name] = region
	}
	f, err := os.Open(filepath.Join(dir, "fifth-seeking-contact-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows[1:] {
		n := func(i int) int {
			value, err := strconv.Atoi(row[i])
			if err != nil {
				t.Fatal(err)
			}
			return value
		}
		mouth := n(0) == 1
		group := &groups[n(0)]
		heading := uint8(n(1))
		clip := group.Components[0].HeadingAnimations[heading]
		state := FifthSeekingState{X: 150, Y: 40, Clock: n(6), Heading: heading, Animation: NewAnimation(clip), Sprite: clip.Frames[0].Sprite, Active: true}
		input := FixedProjectileInputs{ScrollDelta: 1, PlayerX: 200, PlayerY: 100, CanHitPlayer: n(5) == 0, Invulnerable: n(4) != 0, PlayerBounds: CollisionRect{Left: n(20), Top: n(21), Right: n(22), Bottom: n(23)}}
		event := state.AdvanceContact(group, mouth, input, 18, func(name string) visualassets.CollisionBox { return *regions[name].Collision }, func(name string) visualassets.SpriteRegion { return regions[name] })
		if state.X != n(7) || state.Y != n(8) || state.Clock != n(9) || state.Sprite != atlas.SourceSpriteNames[n(10)] || state.Active != (n(11) != 0) {
			t.Fatalf("state Go%+v native%v", state, row)
		}
		if event.Explosion != (n(12) != 0) || event.PlayerDamage != n(15) || event.Explosion && (event.ExplosionX != n(13) || event.ExplosionY != n(14)) {
			t.Fatalf("contact Go%+v native%v", event, row)
		}
		if n(0) == 1 || n(2) == 0 {
			if event.Collision.Left != n(16) || event.Collision.Top != n(17) || event.Collision.Right != n(18) || event.Collision.Bottom != n(19) {
				t.Fatalf("bounds Go%+v native%v", event, row)
			}
		}
	}
	if len(rows)-1 != 512 {
		t.Fatal("incomplete original contact comparisons")
	}
	t.Logf("Compared %d original side/mouth contact, invulnerability, dive and expiry states.", len(rows)-1)
}
