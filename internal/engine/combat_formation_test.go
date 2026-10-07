package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"xenon2/internal/visualassets"
)

func TestCombatFormationNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local original traces")
	}
	common, err := os.ReadFile(filepath.Join(root, "..", "XenonII-unpacked.bin"))
	if err != nil {
		t.Fatal(err)
	}
	var allPaths [5]*visualassets.Paths
	var actors [5]visualassets.Actors
	for level, name := range []string{"000B00E5", "00FA00FE", "02020113", "031F0159", "04820138"} {
		data, err := os.ReadFile(filepath.Join(root, name+".unpacked"))
		if err != nil {
			t.Fatal(err)
		}
		allPaths[level], err = visualassets.DecodePaths(data, common)
		if err != nil {
			t.Fatal(err)
		}
		name := "level-" + string(rune('1'+level)) + "-actors.json"
		data, err = os.ReadFile(filepath.Join(root, "..", "..", "assets", "runtime", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &actors[level]); err != nil {
			t.Fatal(err)
		}
	}
	var wave visualassets.Wave
	var kind visualassets.WaveActor
	var path *visualassets.Path
	var seeds []uint8
	nativeCombatRows(t, "combat-formation-trace.csv", func(v []int64) {
		if int(v[6]) == 2 && int(v[7]) == 0 {
			wave = visualassets.Wave{EnemyKind: int(v[2]), Spacing: int(v[3]), Count: int(v[4]), PathID: int(v[5]), MotionBudget: 7}
			for _, candidate := range actors[v[1]-1].Kinds {
				if candidate.Kind == wave.EnemyKind {
					kind = candidate
					break
				}
			}
			path = &allPaths[v[1]-1].Paths[wave.PathID-1]
			random := RandomState{A: 0x12345678, B: 0x6abcdef1}
			seeds = make([]uint8, wave.Count*len(kind.Parts))
			for i := range seeds {
				seeds[i] = uint8(random.Next())
			}
		}
		part := kind.Parts[v[7]]
		config, err := FormationMotion(wave, int(v[6]), int(v[7]), part)
		if err != nil {
			t.Fatal(err)
		}
		motion, err := NewPathMotion(path, config)
		if err != nil {
			t.Fatal(err)
		}
		health, bonusID := 2, 1
		if part.StrongHealth {
			health, bonusID = 4, 2
			if kind.StrongHealthOverride != 0 {
				health = kind.StrongHealthOverride
			}
		}
		seed := seeds[int(v[6])*len(kind.Parts)+int(v[7])]
		if uint32(motion.X) != uint32(v[8]) || uint32(motion.Y) != uint32(v[9]) || motion.Remaining != int(v[10]) || motion.Budget != int(v[11]) || health != int(v[12]) || part.Score != int(v[13]) || v[14] != 102 || seed != uint8(v[15]) || bonusID != int(v[16]) || part.ResourceTag != int(v[17]) {
			t.Fatalf("formation %v: got motion %+v health=%d bonus=%d part=%+v seed=%d", v, motion, health, bonusID, part, seed)
		}
	})
}
