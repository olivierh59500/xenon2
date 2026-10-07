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
