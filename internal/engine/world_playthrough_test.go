package engine

import (
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// The external replay contains only public controls and observed state. It
// finishes the first level with real scenery, ordinary combat and shop rules.
func TestFirstLevelPublicInputPlaythroughOptional(t *testing.T) {
	root := os.Getenv("XENON2_RUNTIME_ASSET_DIR")
	if root == "" {
		t.Skip("set XENON2_RUNTIME_ASSET_DIR to local exported resources")
	}
	replay := os.Getenv("XENON2_INPUT_REPLAY")
	explicitReplay := replay != ""
	if replay == "" {
		replay = filepath.Join(root, "..", "..", ".local", "native-playcheck", "grid-7-input.csv")
	}
	file, err := os.Open(replay)
	if os.IsNotExist(err) && !explicitReplay {
		t.Skip("local public-input playthrough replay not supplied")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(header) != 16 || header[0] != "pass" || header[5] != "fire" || header[15] != "money" {
		t.Fatal("unexpected public-input replay schema")
	}
	data := playableOriginalWorldData(t, 1)
	session, err := NewSession(data, 1, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	world := session.ActiveWorld()
	steps, shops, continues, losses := 0, 0, 0, 0
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if world.GameOver {
			if !session.AcceptContinue() {
				t.Fatalf("public replay exhausted continue credits before step%d", steps)
			}
			world = session.ActiveWorld()
			continues++
		}
		if world.ShopReady {
			if world.LevelFinished {
				t.Fatalf("public replay continued beyond its completed stage at step%d", steps)
			}
			shops++
			rules := ShopRules{Level: 1, StockLimit: data.Terrain.MidShopStockLimit}
			for _, item := range []Item{ItemSpeedup, ItemAutofire, ItemHealth1, ItemSuperNashwan} {
				// Unavailable purchases leave the real wallet/loadout unchanged.
				_, _ = rules.Buy(&world.Equipment, &world.Money, item)
			}
			rules.Leave(&world.Equipment)
			world.ResumeShop()
		}
		number := func(index int) int {
			value, err := strconv.Atoi(row[index])
			if err != nil {
				t.Fatal(err)
			}
			return value
		}
		boolean := func(index int) bool {
			value, err := strconv.ParseBool(row[index])
			if err != nil {
				t.Fatal(err)
			}
			return value
		}
		if number(0) != steps || number(1) != int(world.Frame) || number(2) != world.ScrollY || number(3) != world.Player.X || number(4) != world.Player.Y || boolean(11) != world.Ready || number(12) != world.Equipment.Lives || number(13) != world.Equipment.Shield || number(14) != world.Score || number(15) != world.Money {
			t.Fatalf("public input step%d diverged from recorded observations: %v", steps, row)
		}
		input := Input{Fire: boolean(5), Motion: MotionInput{Left: boolean(6), Right: boolean(7), Up: boolean(8), Down: boolean(9)}, Dive: boolean(10)}
		lives := world.Equipment.Lives
		if _, err := session.Advance(input); err != nil {
			t.Fatalf("public input step%d: %v", steps, err)
		}
		world = session.ActiveWorld()
		for range 2 {
			world.AdvancePALTick()
		}
		if world.Equipment.Lives < lives {
			losses++
		}
		steps++
	}
	if steps != 8198 || shops != 1 || continues != 2 || losses != 8 || world.GameOver || !world.LevelFinished || !world.ShopReady || !world.ExitReady || world.PendingExitDrops != 0 || world.FirstGuardian == nil || !world.FirstGuardian.Defeated || world.Equipment.Lives != 1 || world.Equipment.Shield != 7 || world.Score != 5300 || world.Money != 2000 {
		t.Fatalf("public playthrough outcome: steps%d shops%d continues%d losses%d lives%d shield%d score%d cash%d finished%v exit%v", steps, shops, continues, losses, world.Equipment.Lives, world.Equipment.Shield, world.Score, world.Money, world.LevelFinished, world.ExitReady)
	}
	t.Logf("Completed level1 through%d public steps, middle shop, final shop readiness and all exit drops", steps)
}
