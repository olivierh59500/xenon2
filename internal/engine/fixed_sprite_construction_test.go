package engine

import (
	"os"
	"path/filepath"
	"testing"

	"xenon2/internal/visualassets"
)

func TestFixedSpriteConstructionNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("local construction traces not supplied")
	}
	var banks [5]*visualassets.FixedSprites
	for level, file := range []string{"000B00E5", "00FA00FE", "02020113", "031F0159", "04820138"} {
		raw, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", file+".decoded"))
		if err != nil {
			t.Fatal(err)
		}
		terrain, err := visualassets.DecodeTerrain(raw)
		if err != nil {
			t.Fatal(err)
		}
		banks[level], err = visualassets.DecodeFixedSprites(level+1, raw, terrain.Palette)
		if err != nil {
			t.Fatal(err)
		}
	}
	cases := 0
	nativeCombatRows(t, "fixed-construction-trace.csv", func(v []int64) {
		bank := banks[v[0]-1]
		kind := bank.Kinds[0]
		variant := int(v[1])
		if kind.VariantSelection == "initial-x-side" {
			variant = 0
			if v[2]-8 > int64(kind.VariantThresholdX) {
				variant = 1
			}
		}
		art := kind.Variants[variant]
		state := NewFixedSpriteState(kind, art, visualassets.FixedEncounter{X: int(v[2]), Y: int(v[3]), State1: 3, State2: 4}, int(v[4]))
		residue := InitializeFixedSpriteResidue(kind, state, ActorResidue{XFraction: 0x1234, YFraction: 0x5678, Counter: 71, Direction: 81, VerticalFraction: 91, Health: 51, StrongHealth: true, PowerOrScore: 101, WaveBonusToken: 111, EmitterClock: 0xa55a})
		if art.ResourceTag != int(v[5]) || residue.X != int16(v[6]) || residue.Y != int16(v[7]) || residue.XFraction != uint16(v[8]) || residue.YFraction != uint16(v[9]) || residue.Counter != int16(v[10]) || residue.Direction != int16(v[11]) || residue.VerticalFraction != uint16(v[12]) || residue.Health != uint16(v[13]) || residue.StrongHealth != (v[14] != 0) || residue.PowerOrScore != uint16(v[15]) || residue.WaveBonusToken != uint16(v[16]) || residue.EmitterClock != uint16(v[17]) || state.Animation.Sprite(art.Animation) != bank.Atlas.SourceSpriteNames[int(v[18])] || state.Animation.Remaining != int(v[19]) || (kind.ActorList == "scenery") != (v[20] == 2) {
			t.Fatalf("fixed constructor %v: state=%+v residue=%+v tag=%d", v, state, residue, art.ResourceTag)
		}
		cases++
	})
	if cases != 30 {
		t.Fatalf("constructor coverage: %d", cases)
	}
}

func TestFixedBeamRenderNativeTraceOptional(t *testing.T) {
	cases := 0
	nativeCombatRows(t, "beam-render-trace.csv", func(v []int64) {
		state := FixedSpriteState{Variant: int(v[0]), Phase: int(v[1]), X: int(v[2]), Y: int(v[3])}
		layout := FixedSpriteBeamLayout(state)
		if layout.TipX != int(v[4]) || layout.TipY != int(v[5]) || layout.ShaftColumns != int(v[8]) || state.Phase != 0 && (layout.ShaftX != int(int16(v[6])) || layout.ShaftY != int(v[7]) || v[9] != 0x88e1) {
			t.Fatalf("beam composition %v: %+v", v, layout)
		}
		cases++
	})
	if cases != 56 {
		t.Fatalf("beam composition coverage: %d", cases)
	}
}

func TestWorldOrdinaryFixedSpritesUseSourceTagsAndRetainedStateOptional(t *testing.T) {
	for level := 1; level <= 5; level++ {
		w, err := NewWorld(originalWorldData(t, level))
		if err != nil {
			t.Fatal(err)
		}
		kind := w.Level.FixedSprites.Kinds[0]
		for _, variant := range kind.Variants {
			w.spawnFixed(visualassets.FixedEncounter{EnemyKind: kind.Kind, Variant: variant.ID, X: 280, Y: w.ScrollY + 120, State1: 3, State2: 4})
			actor := w.Actors[0]
			chosen := variant.ID
			if kind.VariantSelection == "initial-x-side" {
				chosen = 1
			}
			if actor.part.ResourceTag != kind.Variants[chosen].ResourceTag || int(w.Pool.Slot(actor.Binding.Slot).ResourceTag) != actor.part.ResourceTag || actor.Binding.Residue.EmitterClock != 0 {
				t.Fatalf("level%d constructor lost tag or fire state", level)
			}
			if level == 4 {
				actor.fixedState.Phase = 4
				w.composeFixedSprite(actor)
				if actor.Sprite != "" || actor.DrawKind != "assembly" || len(actor.Extras) != 1 || len(actor.TileOverlays) != 4 || actor.Extras[0].X == actor.X {
					t.Fatal("beam shaft/tip composition was omitted")
				}
			}
		}
	}
}

func TestWorldFixedBurstKeepsSourceCallbackAfterListTransferOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("local construction traces not supplied")
	}
	var banks [2]*visualassets.FixedSprites
	for i, file := range []string{"000B00E5", "00FA00FE"} {
		raw, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", file+".decoded"))
		if err != nil {
			t.Fatal(err)
		}
		terrain, err := visualassets.DecodeTerrain(raw)
		if err != nil {
			t.Fatal(err)
		}
		banks[i], err = visualassets.DecodeFixedSprites(i+1, raw, terrain.Palette)
		if err != nil {
			t.Fatal(err)
		}
	}
	var world *World
	var actor *WorldActor
	transientPasses := 0
	nativeCombatRows(t, "fixed-sprite-trace.csv", func(v []int64) {
		if v[0] > 2 {
			return
		}
		bank := banks[v[0]-1]
		kind := &bank.Kinds[0]
		if v[2] == 0 {
			world = testWorld(t)
			world.Level.Number, world.Level.FixedSprites = int(v[0]), bank
			world.fixedKinds[kind.Kind] = kind
			world.ScrollY = int(v[3])
			x := 40
			if v[1] == 1 {
				x = 260
			}
			variant := kind.Variants[v[1]]
			world.spawnFixed(visualassets.FixedEncounter{EnemyKind: kind.Kind, Variant: int(v[1]), X: x - variant.OriginOffsetX, Y: 120 - variant.OriginOffsetY + world.ScrollY, State1: 4, State2: 4})
			actor = world.Actors[0]
		}
		world.ScrollY, world.MaximumScrollY, world.ScrollDelta, world.Player.Y = int(v[3]), int(v[4]), int(v[5]), int(v[6])
		var err error
		if actor.ActorList == "transient" {
			transientPasses++
			err = world.advancePooledProjectiles(Input{})
		} else {
			err = world.advanceActorPhase(ActorPoolMoving, Input{})
		}
		if err != nil {
			t.Fatal(err)
		}
		if int(actor.X) != int(v[7]) || int(actor.Y) != int(v[8]) || actor.Active != (v[15] == 0) || actor.Sprite != bank.Atlas.SourceSpriteNames[int(v[14])] || world.random.A != uint32(v[16]) || world.random.B != uint32(v[17]) {
			t.Fatalf("fixed World phase %v: actor=%+v random=%+v", v, actor, world.random)
		}
	})
	if transientPasses == 0 {
		t.Fatal("source trace never exercised the transferred attack callback")
	}
}
