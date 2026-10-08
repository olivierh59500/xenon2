package engine

import (
	"encoding/base64"
	"testing"
)

// Recorded ordinary controls reproduce the actual Side-equipped fourth entry's
// three published contacts independently of the current presentation pilot.
// Bits encode Up,Down,Left,Right,Fire,Dive; no simulation state or game code.
const fourthOpeningContactControls = "AQEBBQUFBAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAQEBABAAEBAWFhYSEhAACQkGFhAQ" +
	"EAACAAAAAhICFAQJCQkJGBgIEBAAFRUZCRkREREQEAICEhIFFQUBAQQGBAICAgoaEgISEBAQ" +
	"EBAQFAAAAAAAEBAQEQAQEBAAABAQGBgUEBAQABAQGRkSAgIREREaCQkBAAAAABAQEBEWFhEV" +
	"AAACAgIAAAAAAgIKCBgWFgYGBgQEAQgAAAIKCAkJCQkZGBgIGBAQGQEVFRYGAgIAAAEBAQgI" +
	"AQEAAAAAAgICEgIFEhQUBQEBAQEBAAQAAAACAgICAhIQEBgYEBUQAAAICBAQEAAAAQEBAQkA" +
	"AgkCAAAaGhoQEBAQEgISEgkWGAgIAgAAAAAAAQEBAQIAEBAQEBAVGhEFAAAAEBAREBASEhIC" +
	"AgIAEBAQEAAFBQAAAAAEBBAQEBAQEBAAAAEBAAAQEREREREQEAEBAQEAAAAAAAAAAhAaAQoK" +
	"CAgQEQUAAAAAAQEEBAUFEBISEhAQABAWFhYUABAQEBASEhAQAAACAhIQEhIUEBQSAQUAABAQ" +
	"EBAQEBABAQEBAQEBAQEBAQECAgICAgIAAAAAAQEBAQEBAQIBAQEBAQEBAgICAgICAgISEhIS" +
	"ERAZEAAAAAAYGBAAGAgQABAAEAAQABAYCgkKBhgQEAgKEhICCgUFBQIEAgAAAAACAgIICAgI" +
	"CgACCgICBhoKCAkJAAECAgYGAgICBgQBBQQAAAQCCAgKBhYRAQoAARAAAAoAAQ=="

// The two source entry profiles retain their actual inventory and READY RNG.
// Direct entry does not claim that this test replayed the preceding campaign.
func fourthGuardRecordedEntry(t testing.TB, side bool) *World {
	t.Helper()
	equipment := NewEquipment()
	for _, item := range []Item{ItemPowerup, ItemCannon, ItemRearShot, ItemSpeedup, ItemSpeedup, ItemAutofire, ItemAutofire} {
		if !equipment.ApplyItem(item) {
			t.Fatal("recorded fourth entry inventory rejected")
		}
	}
	equipment.Lives = 1
	equipment.Primary.Serial, equipment.Mounts[0].Serial, equipment.Rear.Serial, equipment.NextWeaponSerial = 18, 19, 20, 20
	entry := RandomState{A: 3635312503, B: 963527034}
	afterReady := RandomState{A: 3510710189, B: 3328633438}
	if side {
		if !equipment.ApplyItem(ItemSideShot) || equipment.Rear.Item != ItemNone {
			t.Fatal("native Side install did not replace rear")
		}
		equipment.Primary.Serial, equipment.Mounts[0].Serial, equipment.Side.Serial, equipment.NextWeaponSerial = 19, 20, 21, 21
		entry = RandomState{A: 1963864266, B: 3858137998}
		afterReady = RandomState{A: 911379102, B: 3838764668}
	}
	data := playableOriginalWorldData(t, 4)
	data.InitialEquipment, data.InitialRandom = &equipment, &entry
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	if w.Frame != 0 || w.ScrollY != 4608 || w.MaximumScrollY != 4607 || w.MaterializationFrames != 8 || w.RandomState() != entry {
		t.Fatal("recorded source initializer differs")
	}
	w.ContinueCredits = 2
	w.Ready = true
	// Source star initialization/READY reconstruction: the recorded entry is
	// after144 constructor draws and one star advance (38/42 reseed draws).
	// Fifty further READY star passes precede normal gameplay-star restoration.
	w.SetRandomState(afterReady)
	w.ResetBackgroundStars()
	w.PrimeBackgroundStars(2)
	if err = w.Step(Input{Fire: true}); err != nil {
		t.Fatal(err)
	}
	if w.Ready || w.Frame != 0 {
		t.Fatal("READY admission advanced gameplay")
	}
	return w
}

func fourthRecordedInput(bits byte) Input {
	return Input{Motion: MotionInput{Up: bits&1 != 0, Down: bits&2 != 0, Left: bits&4 != 0, Right: bits&8 != 0}, Fire: bits&16 != 0, Dive: bits&32 != 0}
}

func TestFourthOpeningPublishedContactPrecedesInputOptional(t *testing.T) {
	w := fourthGuardRecordedEntry(t, true)
	inputs, err := base64.StdEncoding.DecodeString(fourthOpeningContactControls)
	if err != nil || len(inputs) != 586 {
		t.Fatal("recorded ordinary contact controls are invalid")
	}
	checked := 0
	for _, bits := range inputs {
		for range 3 {
			w.AdvancePALTick()
		}
		if err = w.Step(fourthRecordedInput(bits)); err != nil {
			t.Fatal(err)
		}
		expectedLoss, ok := map[uint64]int{503: 8, 514: 8, 586: 16}[w.Frame]
		if !ok {
			continue
		}
		before := forecastIsolationDigest(w)
		prefix := thirdMiddlePlayerBounds(w, w.Player)
		var first *WorldActor
		var order [ActorPoolCapacity]*WorldActor
		for _, actor := range w.orderedMovingActors(&order) {
			if actor.Active && actor.ActorList == "moving" && actor.Collision.Intersects(prefix) {
				first = actor
				break
			}
		}
		if first == nil || first.part == nil || !fourthOpeningTerminalContactUnsafe(w) {
			t.Fatalf("recorded frame%d has no published damaging contact", w.Frame)
		}
		damage := ApplyShieldDamage(w.Equipment.Shield, ContactDamage(first.part.StrongHealth), w.Equipment.Protection, false)
		if damage.Lost != expectedLoss {
			t.Fatalf("recorded source contact damage differs atframe%d: %d", w.Frame, damage.Lost)
		}
		for _, motion := range demoDirections {
			var next WorldForecast
			if err = next.Load(w); err != nil {
				t.Fatal(err)
			}
			for range 3 {
				next.AdvancePALTick()
			}
			result, e := next.Advance(Input{Motion: motion})
			if e != nil || result.Shield > damage.Shield {
				t.Fatalf("input escaped pre-movement contact atframe%d motion%+v result%+v err%v", w.Frame, motion, result, e)
			}
		}
		for _, mode := range []string{"dive", "shades", "invulnerable", "pending-cash", "infinite-energy", "screen-clear"} {
			var suppressed WorldForecast
			if err = suppressed.Load(w); err != nil {
				t.Fatal(err)
			}
			q := suppressed.State()
			switch mode {
			case "screen-clear":
				q.ScreenClearFrames = 31
			case "dive":
				q.Dive.Phase = 1
			case "shades":
				q.Equipment.ShadesFrames = 1
			case "invulnerable":
				q.InvulnerableFrames = 1
			case "pending-cash":
				q.PendingExitDrops = 1
			case "infinite-energy":
				q.Cheats.InfiniteEnergy = true
			}
			if fourthOpeningTerminalContactUnsafe(q) {
				t.Fatalf("suppressed contact rejected atframe%d mode%s", w.Frame, mode)
			}
		}
		var hidden WorldForecast
		if err = hidden.Load(w); err != nil {
			t.Fatal(err)
		}
		forecastIsolationActor(t, hidden.State(), first.ID).Visible = false
		if !fourthOpeningTerminalContactUnsafe(hidden.State()) {
			t.Fatal("render visibility filtered source contact")
		}
		for range 3 {
			hidden.AdvancePALTick()
		}
		result, e := hidden.Advance(Input{})
		if e != nil || result.Shield > damage.Shield {
			t.Fatal("hidden source collider stopped damaging")
		}
		for _, level := range []int{1, 2, 3, 5} {
			q := *w
			q.Level.Number = level
			if fourthOpeningTerminalContactUnsafe(&q) {
				t.Fatalf("contact filter escaped level%d", level)
			}
		}
		var protected WorldForecast
		if err = protected.Load(w); err != nil {
			t.Fatal(err)
		}
		protected.State().Equipment.Protection = true
		if !fourthOpeningTerminalContactUnsafe(protected.State()) {
			t.Fatal("protection incorrectly suppressed nonzero source contact damage")
		}
		if forecastIsolationDigest(w) != before {
			t.Fatal("contact forecast changed recorded source")
		}
		checked++
	}
	if checked != 3 {
		t.Fatal("not all recorded contacts were replayed")
	}
}

func TestFourthGuardKeepsRecordedRearProfileAliveThroughNativeTailOptional(t *testing.T) {
	w := fourthGuardRecordedEntry(t, false)
	// This strategy check follows the current connected campaign's READY
	// state. The older entry above remains available to recorded contact tests.
	w.Score, w.DisplayScore = 130480, 130480
	w.SetRandomState(RandomState{A: 344191127, B: 4035761264})
	w.ResetBackgroundStars()
	w.PrimeBackgroundStars(2)
	p := PresentationPilot{PALRefreshes: 3}
	admitted := false
	for pass := 0; pass < 4500; pass++ {
		input := p.NormalInput(w)
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(input); err != nil {
			t.Fatal(err)
		}
		if w.Frame == 1 && (w.Player.X != 160 || w.Player.Y != 171 || w.ScrollY != 4607 || w.RandomState() != (RandomState{A: 2364670790, B: 3602040142})) {
			t.Fatal("current fourth-entry first pass differs from the connected route")
		}
		if !w.PlayerAlive || w.GameOver || w.Cheats.Enabled() || w.Equipment.Lives != 1 || w.ContinueCredits != 2 {
			t.Fatalf("recorded fourth profile lost its ship atframe%d", w.Frame)
		}
		if w.FourthMiddle == nil {
			continue
		}
		if !admitted {
			admitted = true
			if w.Equipment.Shield < 23 || w.FourthMiddle.OuterTargets != 5 || w.FourthMiddle.Parts[4].Health != 175 || w.FourthMiddle.Parts[15].Health != 20 {
				t.Fatal("native middle admission lost source health/reserve")
			}
		}
		if w.FourthMiddle.Parts[15].Disabled {
			if w.Equipment.Shield < 23 || w.FourthMiddle.Parts[15].Health != 0 || w.FourthMiddle.OuterTargets != 4 || w.PendingExitDrops != 0 || w.ShopReady {
				t.Fatal("native tail boundary changed health/gates/rewards")
			}
			if w.Frame != 2542 || w.ScrollY != 2176 || w.Equipment.Shield != 27 || w.RandomState() != (RandomState{A: 34059241, B: 4155321188}) {
				t.Fatal("current connected tail endpoint differs")
			}
			t.Logf("native middle/tail frame%d shield%d sameShip1 credits2", w.Frame, w.Equipment.Shield)
			return
		}
	}
	t.Fatal("recorded fourth profile did not reach native tail boundary")
}
