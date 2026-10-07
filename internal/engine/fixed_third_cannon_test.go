package engine

import (
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"xenon2/internal/visualassets"
)

func TestThirdCompoundCannonNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local original traces")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "02020113.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	terrain, err := visualassets.DecodeTerrain(data)
	if err != nil {
		t.Fatal(err)
	}
	bank, err := visualassets.DecodeFixedSprites(3, data, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	art := bank.Third.Cannon
	file, err := os.Open(filepath.Join(root, "fixed-third-cannon-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	if _, err := reader.Read(); err != nil {
		t.Fatal(err)
	}
	var state ThirdCannonState
	var random RandomState
	var tiles []uint16
	passes := 0
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		v := make([]int64, 15)
		for i := range v {
			v[i], err = strconv.ParseInt(row[i], 10, 64)
			if err != nil {
				t.Fatal(err)
			}
		}
		if v[1] == 0 {
			state = NewThirdCannon(visualassets.FixedEncounter{X: 136, Y: 1704}, art)
			state.Stage = int(v[0])
			random = NewRandomState()
			tiles = append([]uint16(nil), art.Base.Tiles...)
		}
		event := state.Advance(1600, 4607, art, &random)
		if state.Phase != int(v[2]) || state.FireAccumulator != uint8(v[3]) || random.A != uint32(v[4]) || random.B != uint32(v[5]) || event.ShotCount != int(v[6]) || event.Collision != (CollisionRect{Left: int(v[11]), Top: int(v[12]), Right: int(v[13]), Bottom: int(v[14])}) {
			t.Fatalf("cannon stage%d frame%d: %+v event%+v random%+v native%v", v[0], v[1], state, event, random, row)
		}
		if event.ShotCount > 0 && (event.X != int(v[7]) || event.Y != int(v[8]) || event.Speed != int(v[9]) || event.Directions[event.ShotCount-1] != uint8(v[10])) {
			t.Fatalf("cannon emission differs: %+v native%v", event, row)
		}
		if event.WriteFrame {
			patch := art.FirstFrames[event.Frame]
			offset := 17
			if state.Stage == 1 {
				patch = art.SecondFrames[event.Frame]
				offset = 1
			}
			for y := range 2 {
				for x := range 2 {
					tiles[offset+y*4+x] = patch.Tiles[y*2+x]
				}
			}
		}
		fields := strings.Split(row[15], ":")
		for i, field := range fields {
			value, err := strconv.ParseUint(field, 10, 16)
			if err != nil {
				t.Fatal(err)
			}
			if tiles[i] != uint16(value) {
				t.Fatalf("cannon stage%d frame%d tile%d=%d native%d", v[0], v[1], i, tiles[i], value)
			}
		}
		passes++
	}
	if passes != 2000 {
		t.Fatalf("incomplete cannon proof: %d passes", passes)
	}
}
