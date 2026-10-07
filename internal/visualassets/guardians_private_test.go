package visualassets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrivateGuardianTablesOptional(t *testing.T) {
	directory := os.Getenv("XENON2_PRIVATE_DECODED_DIR")
	if directory == "" {
		t.Skip("local decoded originals are optional")
	}
	files := []string{"000B00E5", "00FA00FE", "02020113", "031F0159", "04820138"}
	for index, name := range files {
		t.Run(name, func(t *testing.T) {
			data := mustReadPrivate(t, filepath.Join(directory, name+".decoded"))
			base, err := DecodeTerrain(data)
			if err != nil {
				t.Fatal(err)
			}
			visuals, extra, err := DecodeGuardianArt(index+1, data, base.Palette)
			if err != nil {
				t.Fatal(err)
			}
			groups, atlas, err := DecodeCompoundGuardianArt(index+1, data, base.Palette)
			if err != nil {
				t.Fatal(err)
			}
			extra = append(extra, GuardianGroupTileCodes(groups)...)
			expanded, err := DecodeTerrainWithTiles(data, extra)
			if err != nil {
				t.Fatal(err)
			}
			if visuals != nil {
				if err := RemapGuardianTiles(visuals, expanded.SourceTileIDs); err != nil {
					t.Fatal(err)
				}
			}
			if err := RemapGuardianGroupTiles(groups, expanded.SourceTileIDs); err != nil {
				t.Fatal(err)
			}
			names := map[string]bool{}
			for _, sprite := range atlas.Sprites {
				names[sprite.Name] = true
			}
			for _, group := range groups {
				for _, part := range group.Components {
					for _, frame := range part.Animation.Frames {
						if !names[frame.Sprite] {
							t.Fatalf("missing guardian frame %s", frame.Sprite)
						}
					}
					for _, overlay := range part.Overlays {
						for _, name := range overlay.Frames {
							if !names[name] {
								t.Fatalf("missing overlay %s", name)
							}
						}
					}
				}
			}
			if index == 1 && (len(groups) != 2 || len(groups[0].Components) != 12 || len(groups[0].Launches) != 16 || len(groups[1].DestructibleCells) != 44) {
				t.Fatal("second guardian formats differ")
			}
			if index == 2 && (len(groups) != 1 || len(groups[0].Components) != 17 || groups[0].Path == nil || len(groups[0].Path.Commands) != 10) {
				t.Fatal("third guardian formats differ")
			}
			if index == 3 && (len(groups) != 2 || len(groups[0].Components) != 20 || len(groups[1].Components) != 19) {
				t.Fatal("fourth guardian formats differ")
			}
			if index == 4 && (len(groups) != 2 || len(groups[0].Components) != 10 || len(groups[1].Components) != 22 || visuals.Visuals[0].Body.Columns != 15 || visuals.Visuals[0].Body.Rows != 18) {
				t.Fatal("fifth guardian formats differ")
			}
		})
	}
}
