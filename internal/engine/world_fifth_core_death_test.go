package engine

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"xenon2/internal/visualassets"
)

func TestFifthCoreDeathNativeFactoriesOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("local fifth core factory reference not supplied")
	}
	f, err := os.Open(filepath.Join(root, "fifth-core-death-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "04820138.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	terrain, err := visualassets.DecodeTerrain(raw)
	if err != nil {
		t.Fatal(err)
	}
	_, referenceAtlas, err := visualassets.DecodeCompoundGuardianArt(5, raw, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	type entry struct{ id, tag, x, y int }
	var world *World
	var created []entry
	lastFamily, lastDamage := -1, -1
	scenarios, entries := 0, 0
	for _, row := range rows[1:] {
		n := func(i int) int {
			value, err := strconv.Atoi(row[i])
			if err != nil {
				t.Fatal(err)
			}
			return value
		}
		family, damage, index := n(0), n(1), n(2)
		if family != lastFamily || damage != lastDamage {
			world = fifthResourceWorld(t)
			world.ScrollY = 416
			if err := world.activateFifthGuardian(visualassets.FixedEncounter{Y: 2336}, family == 1); err != nil {
				t.Fatal(err)
			}
			world.SetRandomState(NewRandomState())
			first := world.nextActorID
			actor := world.fifthMiddleActors[5]
			if family == 1 {
				actor = world.fifthFinalActors[21]
			}
			world.damageFifthGuardian(actor, uint16(damage))
			created = created[:0]
			for _, a := range world.Actors {
				if a.ID > first && a.ActorList == "transient" {
					created = append(created, entry{a.ID, a.part.ResourceTag, int(a.X), int(a.Y)})
				}
			}
			for _, cash := range world.Collectibles {
				if cash.ID > first {
					created = append(created, entry{cash.ID, int(world.Pool.Slot(cash.Binding.Slot).ResourceTag), int(cash.X), int(cash.Y)})
				}
			}
			sort.Slice(created, func(i, j int) bool { return created[i].id < created[j].id })
			lastFamily, lastDamage = family, damage
			scenarios++
		}
		var core int
		var sprite string
		if family == 1 {
			core = int(world.FifthFinal.CoreHealth)
			sprite = world.FifthFinal.Parts[21].Sprite
		} else {
			core = world.FifthMiddle.Parts[5].Health
			sprite = world.FifthMiddle.Parts[5].Sprite
		}
		if core != n(7) || world.Score != n(6) || world.random.A != uint32(n(9)) || world.random.B != uint32(n(10)) {
			t.Fatalf("factory state family%d damage%d: health%d score%d random%+v native%v", family, damage, core, world.Score, world.random, row)
		}
		if index >= 0 {
			if index >= len(created) {
				t.Fatalf("missing created entity%d native%v", index, row)
			}
			got := created[index]
			if got.tag != n(3) || got.x != n(4) || got.y != n(5) {
				t.Fatalf("entity%d Go%+v native%v", index, got, row)
			}
			entries++
			if world.ImmediateSoundRequests[0] != "sampled-effect-03" || n(11) != 0x83 {
				t.Fatal("original explosion factory immediate audio dispatch was lost")
			}
		} else if len(created) != 0 {
			t.Fatal("nonlethal core damage created reward entities")
		}
		if family == 1 && sprite != referenceAtlas.SourceSpriteNames[n(8)] {
			t.Fatalf("core image Go%s native%v", sprite, row)
		}
	}
	if scenarios != 20 {
		t.Fatalf("incomplete core scenarios%d", scenarios)
	}
	t.Logf("Compared %d original core damage cases and %d ordered explosion/cash entities.", scenarios, entries)
}
