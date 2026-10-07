package engine

import (
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"xenon2/internal/visualassets"
)

// The optional reference contains execution of the original callbacks. The
// production helper receives semantic artwork and never executes native code.
func TestFixedSpriteMotionNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local original traces")
	}
	file, err := os.Open(filepath.Join(root, "fixed-sprite-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	names := []string{"000B00E5", "00FA00FE", "02020113", "031F0159", "04820138"}
	var banks [5]*visualassets.FixedSprites
	for index, name := range names {
		data, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", name+".decoded"))
		if err != nil {
			t.Fatal(err)
		}
		terrain, err := visualassets.DecodeTerrain(data)
		if err != nil {
			t.Fatal(err)
		}
		banks[index], err = visualassets.DecodeFixedSprites(index+1, data, terrain.Palette)
		if err != nil {
			t.Fatal(err)
		}
	}
	reader := csv.NewReader(file)
	if _, err := reader.Read(); err != nil {
		t.Fatal(err)
	}
	var state FixedSpriteState
	var random RandomState
	cases, frames := 0, 0
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		values := make([]int64, len(row))
		for i, v := range row {
			values[i], err = strconv.ParseInt(v, 10, 64)
			if err != nil {
				t.Fatal(err)
			}
		}
		level, variant, frame := int(values[0]), int(values[1]), int(values[2])
		bank := banks[level-1]
		kind := bank.Kinds[0]
		if frame == 0 {
			original := kind.Variants[variant]
			x, y := 40, 120
			if variant == 1 {
				x = 260
			}
			if level == 3 {
				x = 120
				if variant == 1 {
					x = 200
				}
			}
			if level == 5 {
				y = 32
			}
			event := visualassets.FixedEncounter{X: x - original.OriginOffsetX, Y: y - original.OriginOffsetY + int(values[3]), State1: 4, State2: 4}
			state = NewFixedSpriteState(kind, original, event, int(values[3]))
			random = NewRandomState()
			cases++
		}
		result := StepFixedSpriteMotion(&state, kind, FixedSpriteInputs{ScrollY: int(values[3]), MaximumScrollY: int(values[4]), ScrollDelta: int(values[5]), PlayerX: 160, PlayerY: int(values[6])}, &random)
		velocity := state.VelocityX
		if level <= 2 || level == 5 {
			velocity = state.VelocityY
		}
		if level == 4 {
			velocity = state.PhaseDirection
		}
		phase := state.Phase
		if level <= 2 {
			phase = state.VelocityY
		}
		expectedSprite := bank.Atlas.SourceSpriteNames[int(values[14])]
		sprite := state.Animation.Sprite(FixedSpriteAnimation(state, kind))
		removed := values[15] != 0
		if state.X != int(values[7]) || state.Y != int(values[8]) || velocity != int(values[9]) || state.Distance != int(values[10]) || phase != int(values[11]) || int(state.FireAccumulator) != int(values[12]) || state.Animation.Remaining != int(values[13]) || sprite != expectedSprite || state.Removed != removed || random.A != uint32(values[16]) || random.B != uint32(values[17]) || result.ShotCount != int(values[18]) {
			t.Fatalf("level %d variant %d frame %d: state %+v sprite %s events %+v rng %+v expected %v (%s)", level, variant, frame, state, sprite, result, random, row, expectedSprite)
		}
		if result.ShotCount > 0 && (result.ShotX != int(values[19]) || result.ShotY != int(values[20]) || level <= 2 && (result.ShotDirections[result.ShotCount-1] != int(values[21]) || result.ShotSpeed != int(values[22]))) {
			t.Fatalf("shot level%d frame%d: %+v expected %v", level, frame, result, row)
		}
		frames++
	}
	if cases != 10 || frames < 4000 {
		t.Fatalf("incomplete reference: %d cases, %d passes", cases, frames)
	}
	t.Logf("Compared %d fixed actor cases across %d original callback passes", cases, frames)
}

func TestFixedSpriteClipOrderAndBeamContact(t *testing.T) {
	held := visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "held", Duration: 3}}}
	for _, test := range []struct {
		behavior string
		margin   int
		strict   bool
	}{{"scroll-bounce-attack", 208, true}, {"horizontal-sweeper", 208, true}, {"extending-beam", 392, true}, {"vertical-oscillator", 400, false}} {
		kind := visualassets.FixedSpriteKind{Behavior: test.behavior, MotionParameters: map[string]int{"clip_margin": 208, "clip_bottom": test.margin}, Variants: []visualassets.FixedSpriteVariant{{ID: 0, Animation: held}}}
		boundary := test.margin
		state := FixedSpriteState{Y: boundary, Animation: NewAnimation(held)}
		StepFixedSpriteMotion(&state, kind, FixedSpriteInputs{ScrollDelta: 1}, nil)
		if test.behavior == "scroll-bounce-attack" {
			if state.Removed || state.Y != boundary+1 || state.Animation.Remaining != 2 {
				t.Fatalf("bounce clip runs before camera delta: %+v", state)
			}
		} else if !state.Removed || state.Animation.Remaining != 3 {
			t.Fatalf("clip must stop before animator: %s %+v", test.behavior, state)
		}
	}
	kind := visualassets.FixedSpriteKind{Behavior: "extending-beam", ContactDamage: 6, MotionParameters: map[string]int{"clip_bottom": 392, "maximum_phase": 6, "phase_spacing": 16, "fire_rate": 8}, Variants: []visualassets.FixedSpriteVariant{{ID: 1, Animation: held}}}
	random := NewRandomState()
	before := random
	state := FixedSpriteState{X: 240, Y: 100, Variant: 1, FireAccumulator: 248, Animation: NewAnimation(held)}
	event := StepFixedSpriteMotion(&state, kind, FixedSpriteInputs{}, &random)
	if state.Phase != 1 || state.PhaseDirection != 1 || event.ContactRectangle != [4]int{224, 108, 256, 124} || event.ContactDamage != 6 || random == before {
		t.Fatalf("first beam extension/contact: %+v %+v", state, event)
	}
	before = random
	state.Phase = 5
	StepFixedSpriteMotion(&state, kind, FixedSpriteInputs{}, &random)
	if state.Phase != 6 || state.PhaseDirection != -1 || random != before {
		t.Fatalf("active beam extends without another random draw: %+v", state)
	}
}
