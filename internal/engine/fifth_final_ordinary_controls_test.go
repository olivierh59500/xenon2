package engine

import (
	"encoding/base64"
	"testing"

	"xenon2/internal/visualassets"
)

// This fixture starts inside the original final arena with a hypothetical
// purchased loadout and five ships. Only ordinary controls run after admission:
// no shots, enemy health, completion gates or reward counters are arranged.
// It establishes isolated weapon capability, not a carried campaign victory.
func TestOriginalFifthFinalOrdinaryControlsDefeatAndReachMerchantOptional(t *testing.T) {
	data := playableOriginalWorldData(t, 5)
	equipment := NewEquipment()
	for _, item := range []Item{
		ItemExtraLife, ItemExtraLife, ItemHomingMissile,
		ItemSideShot, ItemSideShot, ItemSideShot,
		ItemLaser, ItemLaser, ItemLaser, ItemLaser,
		ItemAutofire, ItemAutofire, ItemAutofire,
		ItemSpeedup, ItemSpeedup, ItemProtection,
	} {
		if !equipment.ApplyItem(item) {
			t.Fatalf("cannot prepare ordinary item %d", item)
		}
	}
	for equipment.ApplyItem(ItemPowerup) {
	}
	data.InitialEquipment = &equipment
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	w.Ready, w.MaterializationFrames = false, 0
	w.Player.X, w.Player.Y = 160, 176
	var final visualassets.FixedEncounter
	for _, record := range data.Encounters.Fixed {
		if record.EnemyKind == 6 {
			final = record
			break
		}
	}
	if final.EnemyKind != 6 {
		t.Fatal("original final encounter is absent")
	}
	w.spawnFixed(final)
	// Earlier encounters precede this isolated arena; their streams are omitted.
	w.Level.Encounters = &visualassets.Encounters{}
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	if w.FifthFinal == nil || w.FifthFinal.OuterRemaining != 18 || w.FifthFinal.CoreHealth != 20 || w.Equipment.Lives != 5 || w.Coverage == nil {
		t.Fatal("fixture did not retain native guardian health, gates and terrain")
	}
	for i, part := range w.FifthFinal.Parts {
		if part.Health != w.fifthFinalArt.Components[i].Health || part.Destroyed {
			t.Fatalf("component %d did not retain its source constructor", i)
		}
	}
	controls, err := base64.StdEncoding.DecodeString(fifthFinalOrdinaryControls)
	if err != nil || len(controls) != 2594 {
		t.Fatalf("ordinary control recording: %d bytes, error %v", len(controls), err)
	}
	defeatInput := 0
	deaths := 0
	for i, code := range controls {
		if code&0xa0 != 0 {
			t.Fatalf("input %d uses an unsupported command", i)
		}
		if code&64 != 0 && (!w.Ready || code != 80) {
			t.Fatalf("input %d skips PAL ticks outside its READY acknowledgement", i)
		}
		if code&64 == 0 {
			for range 3 {
				w.AdvancePALTick()
			}
		}
		input := Input{Motion: MotionInput{
			Up: code&1 != 0, Down: code&2 != 0,
			Left: code&4 != 0, Right: code&8 != 0,
		}, Fire: code&16 != 0}
		alive := w.PlayerAlive
		if err := w.Step(input); err != nil {
			t.Fatalf("input %d: %v", i, err)
		}
		if alive && !w.PlayerAlive {
			deaths++
		}
		if w.Cheats.Enabled() || w.InvulnerableFrames != 0 || w.Equipment.SuperFrames != 0 || w.Equipment.ShadesFrames != 0 || w.ContinueCredits != 2 || w.GameOver {
			t.Fatalf("input %d used assistance outside the prepared regular loadout", i)
		}
		if w.FifthFinal.OuterRemaining > 0 && w.FifthFinal.CoreHealth != 20 {
			t.Fatalf("input %d bypassed the core's original defense gate", i)
		}
		if defeatInput == 0 && w.FifthFinal.Defeated {
			defeatInput = i + 1
			if defeatInput != 2531 || w.FifthFinal.OuterRemaining != 0 || !w.LevelFinished || w.PendingExitDrops != 20 || len(w.Collectibles) != 20 || w.ShopReady || w.ExitReady || !w.PlayerAlive {
				t.Fatalf("input %d lost the source victory/reward boundary", i)
			}
			for index := 3; index <= 20; index++ {
				if !w.FifthFinal.Parts[index].Destroyed {
					t.Fatalf("final core was defeated before component %d", index)
				}
			}
		}
	}
	if defeatInput == 0 || deaths != 3 || !w.PlayerAlive || w.Equipment.Lives != 2 || w.Equipment.Shield != 3 || w.Score != 7500 || w.Money != 1500 || w.Frame != 2591 || w.RandomState() != (RandomState{A: 534033797, B: 2235120218}) {
		t.Fatalf("ordinary fight diverged: defeat input %d, deaths %d, lives %d, HP %d, score %d, cash %d, frame %d, random %+v", defeatInput, deaths, w.Equipment.Lives, w.Equipment.Shield, w.Score, w.Money, w.Frame, w.RandomState())
	}
	if !w.ShopReady || !w.ExitReady || !w.LevelFinished || w.PendingExitDrops != 0 {
		t.Fatal("ordinary reward collection did not open the final merchant")
	}
	for _, actor := range w.fifthFinalActors {
		slot := w.Pool.Slot(actor.Binding.Slot)
		if actor.Active || slot.allocated && slot.EntityID == actor.ID {
			t.Fatal("defeated guardian still owns a live physical slot")
		}
	}
	t.Logf("Replayed %d ordinary inputs: all eighteen defenses and core defeated, 1500 cash collected, final merchant ready; %d ships lost from five prepared ships.", len(controls), deaths)
}

// Each decoded byte holds Up/Down/Left/Right in bits 0..3 and Fire in bit 4.
// Bit 6 marks the three READY acknowledgements, which consume no gameplay tick.
// The recording contains controls only, with no original artwork or program.
const fifthFinalOrdinaryControls = "EBAQEBAQEBAQEBAQEBAQEBAQEBAUEBUVERgYGBgZERgZGhoaEBAQEBAQFhQVFBQYGBgaGhAQEBAQFBAQEBAQEBAQEBAQEBAQ" +
	"EBAQEBAQEBAQEBAQEBAQEBgYEBgQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAUEBAQEBgQGBAY" +
	"EBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBUYFhYaEhIQEBgWFBgUFBQUFBAUFBQUFhYWFBQUFhQSEBoRGBISEBISEhESEhAS" +
	"EBIVEhASEhASGhIREhIQEhASEhEUEhIQEhoSERISEBIQEhIRFBISEBoSEhEUEhIQEhoSERQSEhAaEhIRFBISEBIaEBASEhIR" +
	"EhISEhIQEBASEBIRFBIaEBISFRQSFBoaGRASEBAWFhISEhISEhIaGBAYEBAQEBAQGBgYGBgQGBgYEBAQEBAUEBgYGBgQEBAQ" +
	"EBAQEBAQEBAQEBgQGBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAYEBAQEBAQEBAREBAQEBASEBAQEBAQEBAQEBAQEBAQ" +
	"EBAQEBAQEBISEhISEhASEBIQEhASEBAQEBAQEBASEBAQEhISERASEhISEhISEhISEhISAAAAAAAAAAAAAAAAAAAAAFAAEBAQ" +
	"EBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQ" +
	"EBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQ" +
	"EBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBgYGBAYGBgYEBAQEBAQEBAQGBgQGBAYEBAQEBAQ" +
	"EBAQEBAQEBAQEBAQEBgQEBAQEBAQEBAQEBoSEhIQEBASEBAQEBAQEBAQEBIQEBAQEBAQEBAQEBAQEhAQEBAQEBgQEBAQEBAQ" +
	"EBAQEBAQEBAREBESEhISEhASEhIQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQGBAQEBAQEBIQEhASEBIQEhASEBIQEhASEBIQ" +
	"EhISEBAVEBAQEBAQEBAQERAQEhASEBIQEhASEBIQEhASEBIQEhASEBIQEhASERISEhISEhISEhIQEBIQEhUUFBAQEBAQEBUY" +
	"GBAQEBkQFhQUEBgaGRAYEBAQEBgSFhAQEhASEBIQEBAQGBAQEBAQEBAQEhASEBIQEhASEBIUEhASEBIQEhAQEBASEhAQEhIQ" +
	"EBAREBIQEhEQEhIREhIQEBISEhESEBASEhISERIZEBISEhISEhISERASEhIQERISEhISEhISEhISEBAQEBYQEBAQEBAQEBAQ" +
	"EBgQERAQEREREREQEBAQEBAQEBISEgAAAAAAAAAAAAAAAAAAAABQAREQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQ" +
	"EBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQ" +
	"EBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAU" +
	"EBkZGhkaEhkaEBAQEBAQEBAQFBkaEBAZGhgQEBAQEBAQEBAQEBAQEBAQEBgQEBAQEBAQEBAQEBgQEBAQEBAQEBAQEBAQGBAQ" +
	"EBAQEBAQEBAQEBAQEBAQEBIQEBAQEBARERAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQGBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQ" +
	"EBAQEBAQEBAQEBAQEBAQEBAQERAQEBAUEBAQEBAQEBEQEBAQEBAQEBAQEBgQEBAQEBAQEBAQEBAQEBAQEBAQEBEQEBAQEhIQ" +
	"EBIQERERERISEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBIWEBAQEBAQEBAQEBAQEBAQEBEREhESERISEBAQEBAQEBAQEBAQEBAQ" +
	"EBAQERESERISEBEREhESERIaAAAAAAAAAAAAAAAAAAAAAFABERAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQ" +
	"EBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQ" +
	"EBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBQQGRka" +
	"GRoSGRoQEBAQEBAQEBAUGRoQEBkaGBAQEBAQEBAQEBAQEBAQEBAQGBAQEBAQEBAQEBAQGBAQEBAQEBAQEBAQEBAYEBASERER" +
	"EREQEBIQEBAQEBAQEBAQEBAQEBAQEBARERAQEBAQEBAQEhISGBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQ" +
	"EBAQEBAQEBEQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQERASEBIQEBEQEhER" +
	"ERASEhIQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBESERERERISEhIREREREhISEhERERESEhARERES" +
	"EhISERERERISEhISEhAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBEREBARERAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQ" +
	"EBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEREREBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQ" +
	"EBAQEBAQEBAQEBASEBAQEBAQEBEUEBAQEBAQEBAVEBAQEBAQGBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQ" +
	"EBAQEBAQEBAQEBAQERAQEBAQEBAQEBAUEBAQEBAQEBgQEBgQEBAQEBAQEBAQEBEREhISEhISEhISEhISEhISEhISEhISEhIS" +
	"EhISEhISEhISEhISEhISEhISEhISEhISEhISEhISEhISEhISEhISEhISEhISEhISEhISEhISEhISEhISEhISEhISEhISEhIS" +
	"EhISEhISEhISEhISEhISEhISEhISEhISEhISEhISEhISEhIQEBAQEhISEBAQEhISEBAQEhISEBAQEhISEBAQEhISEBAQEhIS" +
	"EBAQEhISEhISEhIFBQUFBQUFBQUFBQUFBQUEAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAA="
