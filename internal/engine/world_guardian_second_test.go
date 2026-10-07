package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"xenon2/internal/visualassets"
)

func testSecondArenaWorld(t *testing.T) *World {
	t.Helper()
	base := testWorld(t).Level
	base.Number = 2
	base.Encounters = &visualassets.Encounters{}
	clip := visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "second-frame"}}, Static: true, Ending: "hold"}
	visual := visualassets.GuardianVisual{ID: "final-guardian", InitialHealth: 30, InitialWorldY: 96, BodyX: 112, Body: visualassets.TilePatch{Columns: 6, Rows: 9, Tiles: make([]uint16, 54)}, MotionParameters: map[string]int{"fire_rate": 10, "shot_speed": 6, "turret_health": 6, "turret_fire_rate": 10, "turret_shot_speed": 6, "minion_transform_frame": 1}, TurnPoints: make([]visualassets.GuardianTurnPoint, 8)}
	for _, name := range []string{"guardian-single-shot", "guardian-radial-shot", "guardian-hatch-minion", "guardian-turret-shot"} {
		visual.Animations = append(visual.Animations, visualassets.NamedActorAnimation{ID: name, Ending: "hold", Animation: clip})
	}
	visual.Animations = append(visual.Animations, visualassets.NamedActorAnimation{ID: "guardian-minion-transform", Ending: "hold", Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "second-frame", Duration: 1}, {Sprite: "second-frame"}}}})
	for i := range 8 {
		visual.Animations = append(visual.Animations, visualassets.NamedActorAnimation{ID: fmt.Sprintf("guardian-turret-%d", i), Ending: "hold", Animation: clip})
	}
	bank := visualassets.SpriteAtlas{Sprites: []visualassets.SpriteRegion{{Name: "second-frame", Width: 16, Height: 16, Collision: &visualassets.CollisionBox{Width: 16, Height: 16}}}}
	base.Guardians = &visualassets.Guardians{Visuals: []visualassets.GuardianVisual{visual}, Atlas: bank}
	base.GuardianParts = &bank
	wave := visualassets.GuardianGroup{ID: "middle-defense-wave"}
	for i := range 12 {
		part := visualassets.GuardianComponent{Index: i, ResourceTag: 280, Health: 5, Score: 50, PathBudget: 8, InitialDelay: -i * 10, Sprite: "second-frame", Animation: clip, HeadingFrames: make([]string, 8), DeathAnimation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "second-frame", Duration: 1}}, Ending: "remove"}}
		if i == 0 {
			part.ResourceTag = 276
		}
		for heading := range 8 {
			part.HeadingFrames[heading] = "second-frame"
		}
		wave.Components = append(wave.Components, part)
	}
	for i := range 16 {
		wave.Launches = append(wave.Launches, visualassets.GuardianLaunch{Index: i, X: 160, WorldY: 2650, GateID: 1, Path: visualassets.Path{ID: i + 1, Commands: []visualassets.PathCommand{{Kind: "curve", Heading: 64, Duration: 100}, {Kind: "end"}}}})
	}
	nodes := visualassets.GuardianGroup{ID: "middle-defense-nodes"}
	for i, xy := range [][2]int{{14, 164}, {4, 164}, {9, 184}} {
		part := visualassets.GuardianComponent{Index: i, InitialX: xy[0] * 16, InitialWorldY: xy[1] * 16, Health: 10}
		for range 6 {
			part.TileFrames = append(part.TileFrames, visualassets.TilePatch{Columns: 1, Rows: 1, Tiles: []uint16{0}})
		}
		nodes.Components = append(nodes.Components, part)
	}
	base.GuardianGroups = []visualassets.GuardianGroup{wave, nodes, {ID: "final-terrain-guardian", DestructibleCells: []visualassets.GuardianTerrainCell{{Quadrant: 1, X: 192, WorldY: 96, RestoredTile: 1}}}}
	w, err := NewWorld(base)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"cash-small", "cash-large", "explosion-small", "explosion-large"} {
		w.commonAnimations[name] = visualassets.NamedActorAnimation{ID: name, Ending: "hold", Animation: clip}
	}
	return w
}
func killSecondDefenseHeads(w *World) {
	for _, actor := range w.Actors {
		if actor.Active && actor.secondSegment != nil && actor.secondPart.Index == 0 {
			w.damageActor(actor, uint16(actor.Health))
		}
	}
}
func replaceSecondDefenseWaves(t *testing.T, w *World) {
	t.Helper()
	for _, actor := range w.Actors {
		if actor.secondSegment != nil {
			actor.Active = false
		}
	}
	w.secondStreamsUpdated = [2]bool{}
	if err := w.advanceSecondDefenseWaves(); err != nil {
		t.Fatal(err)
	}
	killSecondDefenseHeads(w)
}

func TestSecondArenaUnlockRetainsFinalGuardianAndGlobalRewardClock(t *testing.T) {
	w := testSecondArenaWorld(t)
	w.ScrollY, w.MinimumScrollY, w.MaximumScrollY = 2600, 2528, 2800
	if len(w.Actors) != 4 || w.secondDefenseRemaining != 3 {
		t.Fatal("source stage initialization needs its body and three nodes")
	}
	if err := w.advanceSecondDefenseWaves(); err != nil {
		t.Fatal(err)
	}
	if w.secondScheduler.DefenseFlags != 3 || len(w.Actors) != 28 {
		t.Fatal("both middle streams must create twelve parts")
	}
	w.Player.X = 240
	w.advanceSecondNode(w.secondNodes[0])
	if !w.secondNodes[0].Collision.Empty() {
		t.Fatal("active stream flags must protect the side nodes")
	}
	killSecondDefenseHeads(w)
	w.advanceSecondNode(w.secondNodes[0])
	if w.secondNodes[0].Collision.Empty() {
		t.Fatal("destroying the heads must expose the matching side")
	}
	w.damageActor(w.secondNodes[0], 10)
	if w.secondDefenseRemaining != 2 || w.BaseScrollStep != 1 {
		t.Fatal("first node defeat must keep forward travel")
	}
	replaceSecondDefenseWaves(t, w)
	w.Player.X = 80
	w.advanceSecondNode(w.secondNodes[1])
	w.damageActor(w.secondNodes[1], 10)
	if w.secondDefenseRemaining != 1 || w.BaseScrollStep != -1 || !w.secondBackward {
		t.Fatal("second node defeat must begin source backward travel")
	}
	w.ScrollY = 2800
	replaceSecondDefenseWaves(t, w)
	w.advanceSecondNode(w.secondNodes[2])
	w.damageActor(w.secondNodes[2], 10)
	if w.secondDefenseRemaining != 0 || w.MinimumScrollY != 0 || w.BaseScrollStep != 1 || !w.secondMiddleReleased || !w.secondGuardianActor.Active || w.LevelFinished || w.ExitReady || w.PendingExitDrops != 10 {
		t.Fatalf("middle completion must retain final guardian and await its ten coins: %+v", w)
	}
	for _, coin := range w.Collectibles {
		coin.Motion.Mode, coin.Motion.Y = 0, 199
		w.advanceCollectible(coin)
	}
	if w.PendingExitDrops != 0 || !w.ShopReady || w.LevelFinished || w.ExitReady {
		t.Fatal("middle reward exhaustion must request shop without ending the level")
	}
	if err := w.advanceSecondDefenseWaves(); err != nil {
		t.Fatal(err)
	}
	if w.ScrollY != 2544 || w.MaximumScrollY != 2544 {
		t.Fatal("released middle arena must preserve the source camera snap")
	}
}

func TestSecondFinalGuardianWaitsForActualHealthAndTwentyExitCoins(t *testing.T) {
	w := testSecondArenaWorld(t)
	w.ScrollY = 80
	w.Player.X, w.Player.Y = 160, 96
	w.advanceSecondGuardian()
	w.advanceSecondGuardian()
	if w.secondGuardianActor.Collision.Empty() {
		t.Fatal("final controller did not open its source weak point")
	}
	w.damageActor(w.secondGuardianActor, 1)
	if w.secondGuardianActor.Health != 29 || w.LevelFinished || w.PendingExitDrops != 0 {
		t.Fatal("nonlethal body hit must not advance or spawn exit rewards")
	}
	w.damageActor(w.secondGuardianActor, 29)
	if w.secondGuardianActor.Active || !w.LevelFinished || w.ExitReady || w.PendingExitDrops != 20 {
		t.Fatal("lethal final hit must wait for its twenty reward coins")
	}
	for _, coin := range w.Collectibles {
		coin.Motion.Mode, coin.Motion.Y = 0, 199
		w.advanceCollectible(coin)
	}
	if !w.ExitReady || !w.ShopReady || w.PendingExitDrops != 0 {
		t.Fatal("drained final rewards must admit the final shop/level transition")
	}
}

func TestSecondTerrainShotsAndTurretRepair(t *testing.T) {
	w := testSecondArenaWorld(t)
	w.ScrollY = 40
	cell := w.secondTerrainCells.Cells[0]
	w.setSecondMapCell(cell.X/16, cell.WorldY/16, cell.RestoredTile)
	if !w.weaponHitPoint(cell.X, cell.WorldY-w.ScrollY, 1) || w.secondTerrainCells.Intact[0] || w.Level.Terrain.Map[cell.WorldY/16*20+cell.X/16] != 0 {
		t.Fatal("a terrain hit must clear exactly one intact source cell")
	}
	if w.weaponHitPoint(cell.X, cell.WorldY-w.ScrollY, 1) {
		t.Fatal("an already removed terrain cell must not consume another shot")
	}
	rect := CollisionRect{Left: cell.X, Top: cell.WorldY - w.ScrollY, Right: cell.X + 15, Bottom: cell.WorldY - w.ScrollY + 15}
	index := w.secondTerrainCells.RestoreOverlap(rect, w.ScrollY)
	if index != 0 || !w.secondTerrainCells.Intact[0] {
		t.Fatal("turret overlap must restore the original intact flag")
	}
	restored := w.secondTerrainCells.Cells[index]
	w.setSecondMapCell(restored.X/16, restored.WorldY/16, restored.RestoredTile)
	if w.Level.Terrain.Map[cell.WorldY/16*20+cell.X/16] != cell.RestoredTile {
		t.Fatal("restoring terrain must use the source restoration tile")
	}
}

func TestSecondWorldOriginalResourcesOptional(t *testing.T) {
	root := os.Getenv("XENON2_RUNTIME_ASSET_DIR")
	if root == "" {
		t.Skip("set XENON2_RUNTIME_ASSET_DIR to exercise exported original resources")
	}
	var data LevelData
	data.Number = 2
	terrain, paths, encounters := &visualassets.Terrain{}, &visualassets.Paths{}, &visualassets.Encounters{}
	actors, fixed, rules := &visualassets.Actors{}, &visualassets.FixedSprites{}, &visualassets.LevelRules{}
	guardians, ships, common := &visualassets.Guardians{}, &visualassets.ShipArt{}, &visualassets.SpriteAtlas{}
	var groups struct {
		Groups []visualassets.GuardianGroup `json:"groups"`
		Atlas  visualassets.SpriteAtlas     `json:"atlas"`
	}
	for _, item := range []struct {
		name   string
		target any
	}{{"level-2.json", terrain}, {"level-2-paths.json", paths}, {"level-2-encounters.json", encounters}, {"level-2-actors.json", actors}, {"level-2-fixed-sprites.json", fixed}, {"level-2-rules.json", rules}, {"level-2-guardians.json", guardians}, {"level-2-guardian-groups.json", &groups}, {"ships.json", ships}, {"common-actors.json", common}} {
		bytes, err := os.ReadFile(filepath.Join(root, item.name))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(bytes, item.target); err != nil {
			t.Fatal(err)
		}
	}
	data.Terrain, data.Paths, data.Encounters, data.Actors, data.FixedSprites, data.Rules, data.Guardians, data.GuardianGroups, data.GuardianParts, data.Ships, data.Common = terrain, paths, encounters, actors, fixed, rules, guardians, groups.Groups, &groups.Atlas, ships, common
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	w.ScrollY, w.PreviousScrollY, w.RenderScrollY, w.MaximumScrollY, w.VisitedScrollY = 2720, 2720, 2720, 2880, 2880
	w.cursor = RestartEncounterCursor(w.ScrollY)
	w.InvulnerableFrames = 10000
	for pass := range 180 {
		if err := w.Step(Input{}); err != nil {
			t.Fatalf("pass%d: %v", pass, err)
		}
		if !w.secondGuardianActor.Active || w.LevelFinished || w.ExitReady || w.ScrollY < w.MinimumScrollY {
			t.Fatalf("ordinary travel must retain both verified arenas on pass%d", pass)
		}
	}
	if w.Frame != 180 || w.secondScheduler == nil || w.secondMinionConfig == nil || len(w.secondTerrainCells.Cells) != 44 {
		t.Fatal("original resources did not initialize the complete second-stage controllers")
	}
	for index := range 3 {
		for _, actor := range w.Actors {
			if actor.secondSegment != nil {
				actor.Active = false
			}
		}
		w.secondStreamsUpdated = [2]bool{}
		w.ScrollY = 2600
		if index == 2 {
			w.ScrollY = 2800
		}
		if err := w.advanceSecondDefenseWaves(); err != nil {
			t.Fatal(err)
		}
		killSecondDefenseHeads(w)
		w.Player.X = 240
		if index == 1 {
			w.Player.X = 80
		}
		node := w.secondNodes[index]
		w.advanceSecondNode(node)
		if node.Collision.Empty() {
			t.Fatalf("source node%d did not become eligible", index)
		}
		w.damageActor(node, uint16(node.Health))
	}
	if !w.secondMiddleReleased || w.secondDefenseRemaining != 0 || w.MinimumScrollY != 0 || w.PendingExitDrops != 10 || w.LevelFinished {
		t.Fatal("original node data failed to unlock the middle arena")
	}
	t.Logf("Advanced %d complete gameplay passes with original level2 paths, guardians, enemies and weapons", w.Frame)
}
