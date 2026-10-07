package visualassets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFifthPersistentTileRemapDoesNotAliasOptional(t *testing.T) {
	root := os.Getenv("XENON2_PRIVATE_DECODED_DIR")
	if root == "" {
		t.Skip("local fifth resources not supplied")
	}
	data, err := os.ReadFile(filepath.Join(root, "04820138.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	tiles, _, err := DecodeFixedTiles(5, data)
	if err != nil {
		t.Fatal(err)
	}
	var ordinary, persistent *FixedTileKind
	for i := range tiles.Kinds {
		if tiles.Kinds[i].Kind == 3 {
			ordinary = &tiles.Kinds[i]
		}
		if tiles.Kinds[i].Kind == 9 {
			persistent = &tiles.Kinds[i]
		}
	}
	if ordinary == nil || persistent == nil {
		t.Fatal("missing aiming turret variants")
	}
	before := persistent.Variants[0].Frames[0].Tiles[0]
	ordinary.Variants[0].Frames[0].Tiles[0] = 0
	if persistent.Variants[0].Frames[0].Tiles[0] != before {
		t.Fatal("shared frame data would be remapped twice")
	}
	before = persistent.InitialChanges[0].After.Tiles[0]
	ordinary.InitialChanges[0].After.Tiles[0] = 0
	if persistent.InitialChanges[0].After.Tiles[0] != before {
		t.Fatal("shared neighbour patches would be remapped twice")
	}
}
