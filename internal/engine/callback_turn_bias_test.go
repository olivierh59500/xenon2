package engine

import (
	"os"
	"path/filepath"
	"testing"
	"xenon2/internal/visualassets"
)

func secondDecisionResources(t *testing.T) ([]byte, *visualassets.FixedSprites, *visualassets.Paths) {
	t.Helper()
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("local callback decision references not supplied")
	}
	level, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "00FA00FE.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	common, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "XenonII.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	terrain, err := visualassets.DecodeTerrain(level)
	if err != nil {
		t.Fatal(err)
	}
	art, err := visualassets.DecodeFixedSprites(2, level, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := visualassets.DecodePaths(level, common)
	if err != nil {
		t.Fatal(err)
	}
	return level, art, paths
}

func TestSecondCrowdedDirectionFromPrecedingPathNativeOptional(t *testing.T) {
	_, art, paths := secondDecisionResources(t)
	comparisons := 0
	nativeCombatRows(t, "guardian-second-crowded-callback.csv", func(v []int64) {
		w := testWorld(t)
		w.Level.Number, w.Level.Paths = 2, paths
		w.secondCrowdedReverse = uint16(v[4])&0x4000 != 0
		actor := &WorldActor{Active: true, ActorList: "moving", X: 150, Y: 40,
			animation: art.Kinds[0].Variants[0].Animation, animationState: AnimationState{Remaining: int(v[9])},
			part: &visualassets.ActorPart{MotionMode: "path"}, path: &paths.Paths[0],
			motion: PathMotionState{PathID: 1, ProgramCounter: 1, X: int32(150<<16) + int32(v[0]), Y: 40 << 16, AngleFixed: 64 << 16, Remaining: int(v[1]), Budget: int(v[2]), Active: true},
			fire:   EnemyFireState{Rate: uint8(v[3])}}
		if v[3] != 0 {
			actor.fire.Accumulator = 255
		}
		if err := w.advanceMovingActor(actor); err != nil {
			t.Fatal(err)
		}
		if uint32(actor.motion.X) != uint32(v[5]) || w.secondCrowdedReverse != (uint16(v[6])&0x4000 != 0) {
			t.Fatalf("preceding path decision differs: Go%+v reverse%v native%v", actor.motion, w.secondCrowdedReverse, v)
		}
		guardian := SecondGuardianState{BodyWorldY: 144, WaitTimer: 100}
		event := guardian.Advance(SecondGuardianInput{MaximumScrollY: 288, ActorCount: 20, FireRate: 10, ReverseWhenCrowded: w.secondCrowdedReverse}, nil)
		if guardian.Velocity != int(v[7]) || guardian.MotionRemaining != int(v[8]) || event.SpawnMinion {
			t.Fatalf("crowded guardian direction differs: Go%+v native%v", guardian, v)
		}
		comparisons++
	})
	if comparisons != 80 {
		t.Fatalf("incomplete callback chain comparison: %d", comparisons)
	}
	t.Logf("Compared %d original path-to-guardian callback decisions.", comparisons)
}

func TestSecondBounceCallbackDecisionNativeOptional(t *testing.T) {
	_, art, _ := secondDecisionResources(t)
	kind := art.Kinds[0]
	var state FixedSpriteState
	var random RandomState
	comparisons := 0
	nativeCombatRows(t, "fixed-sprite-decision.csv", func(v []int64) {
		if v[3] == 0 {
			variant := int(v[2])
			state = NewFixedSpriteState(kind, kind.Variants[variant], visualassets.FixedEncounter{X: 48 + variant*220, Y: 1128, State1: 4}, 1000)
			random = NewRandomState()
		}
		event := StepFixedSpriteMotion(&state, kind, FixedSpriteInputs{ScrollY: int(v[4]), MaximumScrollY: int(v[5]), ScrollDelta: int(v[6]), PlayerX: 160, PlayerY: int(v[7])}, &random)
		if event.callbackTurnBias.apply(false) != (uint16(v[0])&0x4000 != 0) {
			t.Fatalf("bounce return differs: state%+v event%+v native%v", state, event, v)
		}
		comparisons++
	})
	if comparisons != 1200 {
		t.Fatalf("incomplete bounce comparison: %d", comparisons)
	}
}

func TestSecondPodCallbackDecisionNativeOptional(t *testing.T) {
	_, art, _ := secondDecisionResources(t)
	var state PodCreatureState
	decision := false
	pool := NewActorPool()
	comparisons := 0
	nativeCombatRows(t, "fixed-pod-creature-decision.csv", func(v []int64) {
		if v[3] == 0 {
			clip := art.PodCreatures.Idle[v[2]]
			state = PodCreatureState{X: 100, Y: 100, Variant: int(v[2]), AllocationPhase: int(pool.Slot(int(v[1])).AllocationPhase), Clip: clip, Animation: NewAnimation(clip)}
			decision = false
		}
		playerX := 160
		if v[3] >= 90 {
			playerX = 100
		}
		state.Advance(uint64(v[3]), playerX, art.PodCreatures)
		decision = state.callbackTurnBias.apply(decision)
		if decision != (uint16(v[0])&0x4000 != 0) {
			t.Fatalf("pod return differs: state%+v reverse%v native%v", state, decision, v)
		}
		comparisons++
	})
	if comparisons != 5120 {
		t.Fatalf("incomplete pod comparison: %d", comparisons)
	}
}

func TestSecondHatchCallbackDecisionNativeOptional(t *testing.T) {
	_, art, _ := secondDecisionResources(t)
	w := testWorld(t)
	w.Level.Number, w.Level.FixedSprites = 2, art
	w.Player.X, w.Player.Y, w.ScrollDelta = 160, 176, 1
	w.playerCollision = CollisionRect{Left: 1000, Top: 1000, Right: 1000, Bottom: 1000}
	for _, region := range art.Atlas.Sprites {
		if region.Collision != nil {
			w.movingSpriteBoxes[region.Name] = *region.Collision
		}
	}
	var actor *WorldActor
	comparisons := 0
	nativeCombatRows(t, "fixed-hatch-creature-decision.csv", func(v []int64) {
		if v[2] == 0 {
			clip := art.HatchCreatures.HeadingAnimations[v[1]]
			state := HatchCreatureState{X: 160, Y: 100, Direction: uint8(v[1]), Clip: clip, Animation: NewAnimation(clip)}
			actor = &WorldActor{Active: true, Atlas: "fixed", hatchCreature: &state, Sprite: state.Animation.Sprite(clip)}
			w.secondCrowdedReverse = false
		}
		w.advanceHatchCreature(actor)
		if w.secondCrowdedReverse != (uint16(v[0])&0x4000 != 0) {
			t.Fatalf("hatch return differs: state%+v reverse%v native%v", actor.hatchCreature, w.secondCrowdedReverse, v)
		}
		comparisons++
	})
	if comparisons < 200 {
		t.Fatalf("incomplete hatch comparison: %d", comparisons)
	}
	t.Logf("Compared %d original finite hatch creature return decisions.", comparisons)
}

func TestSecondMinionCallbackDecisionNativeOptional(t *testing.T) {
	comparisons := 0
	nativeCombatRows(t, "guardian-second-minion-decisions.csv", func(v []int64) {
		if uint16(v[0])&0x4000 != 0 {
			t.Fatalf("arena minion has an unexpected reverse callback outcome: %v", v)
		}
		comparisons++
	})
	if comparisons != 14400 {
		t.Fatalf("incomplete minion comparison: %d", comparisons)
	}
}

func TestShadowInitialCrowdedDecisionNativeOptional(t *testing.T) {
	comparisons := 0
	nativeCombatRows(t, "player-shadow-decisions.csv", func(v []int64) {
		state := PlayerShadowState{Counter: int(v[1])}
		player := PlayerMotionState{X: int(v[5]), Y: int(v[6]), Inertia: int(v[2])}
		input := MotionInput{Up: v[3]&1 != 0, Down: v[3]&2 != 0}
		if err := StepPlayerShadow(&state, player, input, v[4] != 0); err != nil {
			t.Fatal(err)
		}
		if state.X != int(v[7]) || (int16(state.X) < 0) != (uint16(v[0])&0x4000 != 0) {
			t.Fatalf("player-list initial decision differs: state%+v native%v", state, v)
		}
		comparisons++
	})
	if comparisons != 416 {
		t.Fatalf("incomplete shadow comparison: %d", comparisons)
	}
}
