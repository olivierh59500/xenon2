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

func TestFirstGuardianWorldCompletionWaitsForExitCash(t *testing.T) {
	w := testWorld(t)
	w.Level.Guardians = &visualassets.Guardians{Visuals: []visualassets.GuardianVisual{{InitialHealth: 30, BodyX: 112, Body: visualassets.TilePatch{Columns: 1, Rows: 1, Tiles: []uint16{0}}}}}
	w.firstGuardianArt = &w.Level.Guardians.Visuals[0]
	state := NewFirstGuardianState(30)
	w.FirstGuardian = &state
	w.firstGuardianActor = &WorldActor{ID: 10, Active: true, ActorList: "moving", firstGuardian: true, part: &visualassets.ActorPart{ResourceTag: 80}}
	w.Actors = append(w.Actors, w.firstGuardianActor)
	for _, name := range []string{"cash-small", "cash-large"} {
		w.commonAnimations[name] = visualassets.NamedActorAnimation{ID: name, Ending: "hold", Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: name}}}}
	}
	w.commonAnimations["explosion-large"] = visualassets.NamedActorAnimation{ID: "explosion-large", Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "large-explosion"}}}}
	w.ScrollY = 48
	w.advanceFirstGuardian()
	if !w.FirstGuardian.Active || len(w.Actors) != 9 {
		t.Fatal("guardian activation must create eight articulated pieces")
	}
	w.strikeFirstGuardian(w.FirstGuardian.WeakPoint(w.ScrollY), 30)
	if !w.FirstGuardian.Defeated || w.PendingExitDrops != 18 || w.ExitReady || len(w.Collectibles) != 18 {
		t.Fatal("guardian death must await its eighteen cash rewards")
	}
	explosions := 0
	for _, actor := range w.Actors {
		if actor.Active && actor.Sprite == "large-explosion" {
			explosions++
		}
	}
	if explosions != 20 {
		t.Fatal("guardian death must emit the source's twenty random explosions")
	}
	for _, coin := range w.Collectibles {
		if coin.Order >= 0 {
			t.Fatal("both source reward coins must retain tail insertion")
		}
		coin.Motion.Mode, coin.Motion.Y = 0, 199
		w.advanceCollectible(coin)
	}
	if w.PendingExitDrops != 0 || !w.ExitReady {
		t.Fatal("expiring all exit coins must release the stage exit")
	}
}

func TestFirstGuardianBodyDecorationNativeOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to local source comparisons")
	}
	bytes, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "000B00E5.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	terrain, err := visualassets.DecodeTerrain(bytes)
	if err != nil {
		t.Fatal(err)
	}
	guardians, _, err := visualassets.DecodeGuardianArt(1, bytes, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	art := guardians.Visuals[0]
	animation := art.BodyAnimations[0]
	file, err := os.Open(filepath.Join(root, "guardian-first-decoration-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	if _, err := reader.Read(); err != nil {
		t.Fatal(err)
	}
	state := NewFirstGuardianState(30)
	maximum := 4607
	patch := readBodyDecoration(art.Body, animation.Column, animation.Row)
	frames := 0
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		frame, _ := strconv.Atoi(row[0])
		scroll, _ := strconv.Atoi(row[1])
		health, _ := strconv.Atoi(row[3])
		state.Health = uint16(health)
		maximum, _, _ = state.Advance(uint64(frame), scroll, maximum)
		if state.Active {
			patch = animation.Frames[(frame&12)>>2]
		}
		words := []uint32{uint32(patch.Tiles[0])<<16 | uint32(patch.Tiles[1]), uint32(patch.Tiles[2])<<16 | uint32(patch.Tiles[3]), uint32(patch.Tiles[4])<<16 | uint32(patch.Tiles[5]), uint32(patch.Tiles[6])<<16 | uint32(patch.Tiles[7])}
		for i, part := range strings.Split(row[18], ":") {
			want, _ := strconv.ParseUint(part, 10, 32)
			if words[i] != uint32(want) {
				t.Fatalf("body decoration differs pass%d rowword%d", frame, i)
			}
		}
		frames++
	}
	if frames != 1200 {
		t.Fatalf("incomplete decoration trace:%d", frames)
	}
}

func readBodyDecoration(body visualassets.TilePatch, column, row int) visualassets.TilePatch {
	patch := visualassets.TilePatch{Columns: 4, Rows: 2}
	for y := range 2 {
		patch.Tiles = append(patch.Tiles, body.Tiles[(row+y)*body.Columns+column:][:4]...)
	}
	return patch
}
