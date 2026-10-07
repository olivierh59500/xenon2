package engine

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"xenon2/internal/visualassets"
)

func TestSecondGuardianNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local original traces")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "00FA00FE.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	terrain, err := visualassets.DecodeTerrain(data)
	if err != nil {
		t.Fatal(err)
	}
	artwork, _, err := visualassets.DecodeGuardianArt(2, data, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	visual := artwork.Visuals[0]
	file, err := os.Open(filepath.Join(root, "guardian-second-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	if _, err := reader.Read(); err != nil {
		t.Fatal(err)
	}
	var state SecondGuardianState
	var random RandomState
	maximum := 100
	var tiles []uint16
	cases, frames := 0, 0
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		values := make([]int64, 29)
		for i := range values {
			values[i], err = strconv.ParseInt(row[i], 10, 64)
			if err != nil {
				t.Fatal(err)
			}
		}
		if values[1] == 0 {
			state = NewSecondGuardianState()
			random = NewRandomState()
			maximum = 100
			tiles = append([]uint16(nil), visual.Body.Tiles...)
			cases++
		}
		event := state.Advance(SecondGuardianInput{Frame: uint64(values[1]), ScrollY: int(values[2]), MaximumScrollY: maximum, PlayerX: int(values[4]), PlayerY: int(values[5]), ActorCount: int(values[6]), ReverseWhenCrowded: values[7]&0x4000 != 0, FireRate: uint8(visual.MotionParameters["fire_rate"]), ShotSpeed: visual.MotionParameters["shot_speed"]}, &random)
		maximum = event.MaximumScrollY
		if state.BodyWorldY != int(values[8]) || state.Velocity != int(values[9]) || state.WaitTimer != int(values[10]) || state.MotionRemaining != int(values[11]) || state.HatchCountdown != int(values[12]) || state.HatchFrame != int(values[13]) || int(state.FireAccumulator) != int(values[14]) || random.A != uint32(values[15]) || random.B != uint32(values[16]) || maximum != int(values[3]) || event.ShotCount != int(values[21]) || event.SpawnMinion != (values[26] != 0) || event.MovementSound != (values[28] != 0) {
			t.Fatalf("case%d frame%d: state%+v event%+v random%+v expected%v", values[0], values[1], state, event, random, row)
		}
		if values[17] != 1000 && state.BodyCollision != (CollisionRect{Left: int(values[17]), Top: int(values[18]), Right: int(values[19]), Bottom: int(values[20])}) {
			t.Fatalf("collision frame%d: %+v native%v", values[1], state.BodyCollision, row[17:21])
		}
		if values[17] == 1000 && !state.BodyCollision.Empty() {
			t.Fatal("waiting guardian must not collide")
		}
		if event.ShotCount > 0 && (event.ShotX != int(values[22]) || event.ShotY != int(values[23]) || event.ShotSpeed != int(values[24]) || event.ShotDirections[event.ShotCount-1] != uint8(values[25])) {
			t.Fatalf("shot differs: %+v native%v", event, row)
		}
		if event.SpawnMinion && event.MinionHeading != uint8(values[27]) {
			t.Fatalf("minion differs: %+v native%v", event, row)
		}
		for _, animation := range visual.BodyAnimations {
			index := -1
			if animation.ID == "center-hatch" && event.CenterHatchChanged {
				index = event.CenterHatchFrame
			}
			if animation.ID != "center-hatch" && event.ThrustersChanged {
				index = event.ThrusterFrame
			}
			if index < 0 {
				continue
			}
			patch := animation.Frames[index]
			for y := range patch.Rows {
				for x := range patch.Columns {
					tiles[(animation.Row+y)*visual.Body.Columns+animation.Column+x] = patch.Tiles[y*patch.Columns+x]
				}
			}
		}
		for i, index := range []int{38, 42, 46} {
			actual := fmt.Sprintf("%d:%d", uint32(tiles[index])<<16|uint32(tiles[index+1]), uint32(tiles[index+6])<<16|uint32(tiles[index+7]))
			if actual != row[29+i] {
				t.Fatalf("body tiles case%d frame%d patch%d: %s native%s", values[0], values[1], i, actual, row[29+i])
			}
		}
		frames++
	}
	if cases != 4 || frames != 12000 {
		t.Fatalf("incomplete comparison: %d cases, %d passes", cases, frames)
	}
	t.Logf("Compared %d arena cases across %d original controller passes", cases, frames)
}

func TestSecondGuardianEntryAndCrowdedDecision(t *testing.T) {
	state := NewSecondGuardianState()
	random := NewRandomState()
	before := random
	event := state.Advance(SecondGuardianInput{PlayerX: 160, PlayerY: 160, MaximumScrollY: 0}, &random)
	if event.MaximumScrollY != 288 || state.MotionRemaining != 1000 || state.Velocity != 3 || state.WaitTimer != 340 || random != before || !state.BodyCollision.Empty() {
		t.Fatalf("arena entry must defer travel and attacks: %+v %+v", state, event)
	}
	state.MotionRemaining = 0
	state.WaitTimer = 100
	state.BodyWorldY = 144
	state.Advance(SecondGuardianInput{PlayerX: 160, PlayerY: 160, ActorCount: 20, FireRate: 10, ReverseWhenCrowded: true}, &random)
	if state.Velocity != -3 || state.MotionRemaining != 10 || random != before {
		t.Fatalf("crowded decision must preserve its source value without drawing random: %+v", state)
	}
}

func TestSecondDefenseNodesNativeTraceOptional(t *testing.T) {
	var state SecondDefenseNodeState
	var counters [8]int
	frames := 0
	nativeCombatRows(t, "guardian-second-nodes-trace.csv", func(v []int64) {
		if v[2] == 0 {
			state = NewSecondDefenseNodeState(int(v[1]), 9, 184, 50)
			for i := range counters {
				counters[i] = i * 3
			}
		}
		event := state.Advance(SecondDefenseNodeInput{Frame: uint64(v[2]), Remaining: int(v[3]), DefenseFlags: uint8(v[4]), PlayerX: int(v[5]), BackwardScroll: v[6] != 0, MaximumScrollY: 100, ScrollY: 2700}, &counters)
		if state.Phase != int(v[8]) || event.MaximumScrollY != int(v[7]) || uint16(36177+10*event.TileFrame) != uint16(v[13]) {
			t.Fatalf("node clock differs: %+v %+v native%v", state, event, v)
		}
		if v[9] != 1000 && state.Collision != (CollisionRect{Left: int(v[9]), Top: int(v[10]), Right: int(v[11]), Bottom: int(v[12])}) {
			t.Fatalf("node bounds differ: %+v native%v", state.Collision, v)
		}
		if v[9] == 1000 && !state.Collision.Empty() {
			t.Fatal("ineligible node must not collide")
		}
		for i, value := range counters {
			if value != int(v[14+i]) {
				t.Fatalf("node gate%d counter=%d native%d", i, value, v[14+i])
			}
		}
		frames++
	})
	if frames != 1152 {
		t.Fatalf("incomplete comparison: %d passes", frames)
	}
	t.Logf("Compared 18 defense-node cases across %d original passes", frames)
}

func TestSecondDefenseNodeDamageReleasesMiddleArena(t *testing.T) {
	state := NewSecondDefenseNodeState(0, 14, 164, 50)
	if event := state.Strike(50, 3, 3); event.Applied || state.Health != 50 || state.Destroyed {
		t.Fatal("active defense-wave flags must block damage")
	}
	event := state.Strike(50, 2, 0)
	if !event.Destroyed || event.Remaining != 1 || !event.ReverseScroll || event.ReleaseMinimum {
		t.Fatalf("second destroyed node must reverse scrolling: %+v", event)
	}
	state = NewSecondDefenseNodeState(2, 9, 184, 50)
	event = state.Strike(50, 1, 0)
	if !event.Destroyed || event.Remaining != 0 || !event.ReleaseMinimum || !event.ClearWaveActors || event.CashPairs != 5 || event.ExplosionCount != 40 {
		t.Fatalf("last node must release the middle arena: %+v", event)
	}
}
