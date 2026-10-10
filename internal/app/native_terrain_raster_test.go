package app

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// The private rasters execute the original background/map routine, including
// its copy, clear and masked blits. They do not use the exported tile atlas to
// construct their expected pixels.
func TestOriginalTerrainRastersMatchProductionGPUOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("local original terrain rasters not supplied")
	}
	root = filepath.Join(root, "terrain-raster")
	file, err := os.Open(filepath.Join(root, "cases.csv"))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(file).ReadAll()
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 61 {
		t.Fatalf("expected sixty native terrain cases, got%d", len(rows)-1)
	}
	g, _ := renderIntegrationGame(t)
	var levels [5]int
	compared := 0
	for _, row := range rows[1:] {
		if len(row) != 5 {
			t.Fatal("malformed terrain raster case")
		}
		var values [5]int
		for i, field := range row {
			values[i], err = strconv.Atoi(field)
			if err != nil {
				t.Fatal(err)
			}
		}
		index, level, camera, background := values[0], values[1], values[2], values[3]
		if index != compared/(ScreenWidth*PlayfieldHeight) || level < 1 || level > 5 || camera < 0 || camera > 4608 || background < 0 || background >= PlayfieldHeight || values[4] < 2 {
			t.Fatalf("invalid native terrain case%v", values)
		}
		native, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("terrain-%03d.bin", index)))
		if err != nil {
			t.Fatal(err)
		}
		if len(native) != ScreenWidth*PlayfieldHeight/2 {
			t.Fatal("native terrain raster is incomplete")
		}
		g.View = SceneFrame{Level: level, CameraY: float64(camera), BackgroundY: float64((PlayfieldHeight - background) % PlayfieldHeight)}
		g.rememberFrameHistory()
		actual := renderIntegrationPixels(t, g, fmt.Sprintf("native-terrain-%03d", index))
		differences := 0
		for y := 0; y < PlayfieldHeight; y++ {
			for x := 0; x < ScreenWidth; x++ {
				color := 0
				for plane := 0; plane < 4; plane++ {
					if native[y*160+plane*40+x/8]&(1<<uint(7-x%8)) != 0 {
						color |= 1 << uint(plane)
					}
				}
				want := g.Bundle.Levels[level-1].Terrain.Palette[color]
				got := actual.NRGBAAt(x, y)
				if [4]uint8{got.R, got.G, got.B, got.A} != want {
					if differences < 3 {
						t.Logf("case%d level%d camera%d background%d pixel%d/%d: Go%v original%v", index, level, camera, background, x, y, got, want)
					}
					differences++
				}
			}
		}
		if differences != 0 {
			t.Errorf("case%d level%d camera%d background%d: %d terrain pixel differences", index, level, camera, background, differences)
		}
		levels[level-1]++
		compared += ScreenWidth * PlayfieldHeight
	}
	if levels != ([5]int{12, 12, 12, 12, 12}) {
		t.Fatalf("incomplete original level coverage: %v", levels)
	}
	t.Logf("Compared %d original background/map pixels across sixty views and all five levels", compared)
}
