package app

import (
	"encoding/base64"
	"os"
	"testing"

	"xenon2/internal/engine"
	"xenon2/internal/presentation"
	"xenon2/internal/shopui"
)

// The ordinary intro earns all preceding victories, inventory and merchants.
// Recorded controls then take over through the real input boundary; no world
// state, damage, completion gate or resource is assigned by this continuation.
func TestCurrentFiveStageReferenceCompletesEndingAndStartsNextRoundOptional(t *testing.T) {
	if os.Getenv("XENON2_HUMAN_PRESENTATION_CHECK") == "" {
		t.Skip("enable the complete five-stage reference journey explicitly")
	}
	g := verifyPresentationFifthFinalAdmission(t)
	codes, err := base64.StdEncoding.DecodeString(currentFifthFinalValidationControls)
	if err != nil || len(codes) != 1099 {
		t.Fatalf("current final controls: length%d error%v", len(codes), err)
	}
	for i, code := range codes {
		if code&0xa0 != 0 || code&64 != 0 && code != 80 {
			t.Fatalf("control%d has unsupported bits", i)
		}
	}
	cursor, deaths := 0, 0
	victory, merchant, nextRound := false, false, false
	ending := map[shopui.Phase]bool{}
	for update := 0; update < 60*360; update++ {
		d := g.Driver.(*worldDriver)
		w := d.world
		if d.diagnostic || w.Cheats.Enabled() || w.GameOver || d.session.PlayerCount != 1 || w.Equipment.Lives < 2 {
			t.Fatalf("reference journey changed normal rules or reserves: F%d ships%d", w.Frame, w.Equipment.Lives)
		}
		if len(w.UnimplementedFixedEncounters) != 0 || w.Level.Number == 5 && w.FifthFinal.OuterRemaining > 0 && w.FifthFinal.CoreHealth != 20 {
			t.Fatal("reference journey skipped an encounter or bypassed the original core gate")
		}
		if w.Level.Number == 1 {
			if !victory || !merchant || cursor != len(codes) || deaths != 1 || d.session.Difficulty != 2 || d.session.Current != 0 || d.session.Completed != ([2]bool{}) || w.ContinueCredits != 3 || w.Equipment.Lives != 2 || w.Equipment.Shield != 9 || w.Score != 284750 || w.Money != 0 || w.Equipment.Primary.Item != engine.ItemForwardShot || w.Equipment.Primary.Tier != 0 || w.Equipment.Mounts != ([4]engine.WeaponSlot{}) || w.Equipment.Rear.Item != engine.ItemNone || w.Equipment.Side.Item != engine.ItemNone {
				t.Fatal("native ending did not admit the next round with the earned score, ships and original weapon reset")
			}
			for _, phase := range []shopui.Phase{shopui.MerchantEnding, shopui.EndingFade, shopui.EndingDot, shopui.EndingDotFade, shopui.EndingWait} {
				if !ending[phase] {
					t.Fatalf("native ending omitted%s", phase)
				}
			}
			nextRound = true
			if g.Screen == LevelScreen && !w.Ready && !g.backdropOnly && w.Frame > 0 && (g.fade == nil || g.fade.Done) {
				renderIntegrationPixels(t, g, "current-five-stage-next-round-gameplay")
				t.Logf("Current default-intro journey completes all five stages, collects all1500 final cash, shows every ending phase and starts round2: score%d two ships three continues; one ship lost, no continue spent", w.Score)
				return
			}
		} else if w.Level.Number != 5 || w.ContinueCredits != 2 {
			t.Fatal("final encounter or ending entered an unrelated stage or consumed a continue")
		}
		in := inputFrame{}
		before, ready, alive := w.Frame, w.Ready, w.PlayerAlive
		code := byte(0)
		if cursor < len(codes) {
			code = codes[cursor]
			in = inputFrame{gameMotion: engine.MotionInput{Up: code&1 != 0, Down: code&2 != 0, Left: code&4 != 0, Right: code&8 != 0}, fire: code&16 != 0, anyKey: g.DemoActive()}
		}
		if w.Ready && g.Screen == PresentationScreen && g.director.Phase == presentation.ReadyMessage && g.director.Data.MessageSteps[g.director.Step] == 17 {
			in.confirm = true
			if nextRound {
				renderIntegrationPixels(t, g, "current-five-stage-next-round-ready")
			}
		}
		if g.Screen == ShopScreen && g.shop != nil {
			if g.shop.Phase == shopui.Selling || g.shop.Phase == shopui.Buying || !g.shop.Ending {
				t.Fatal("fifth completion exposed a sale or purchase instead of the native ending")
			}
			if !ending[g.shop.Phase] && g.shop.Phase == shopui.MerchantEnding {
				renderIntegrationPixels(t, g, "current-five-stage-merchant-ending")
			}
			ending[g.shop.Phase] = true
		}
		advanceFrontend(t, g, in)
		w = g.Driver.(*worldDriver).world
		if cursor < len(codes) && (w.Frame != before || ready && !w.Ready) {
			if code&64 != 0 && (!ready || w.Frame != before || w.Ready) {
				t.Fatal("READY acknowledgement advanced combat or bypassed its normal director")
			}
			cursor++
		}
		if alive && !w.PlayerAlive {
			deaths++
		}
		if w.Level.Number == 5 && w.FifthFinal != nil && w.FifthFinal.Defeated && !victory {
			if w.Frame != 6562 || w.ScrollY != 206 || w.Equipment.Shield != 9 || w.Equipment.Lives != 2 || deaths != 1 || w.Score != 284750 || w.Money != 850 || w.PendingExitDrops != 20 || !w.LevelFinished || w.ShopReady || w.ExitReady || w.FifthFinal.OuterRemaining != 0 || w.FifthFinal.CoreHealth != 0 || w.RandomState() != (engine.RandomState{A: 3660352986, B: 3935538876}) {
				t.Fatalf("native final victory differs: F%d C%d HP%d ships%d score%d cash%d", w.Frame, w.ScrollY, w.Equipment.Shield, w.Equipment.Lives, w.Score, w.Money)
			}
			for index := 3; index <= 20; index++ {
				if !w.FifthFinal.Parts[index].Destroyed {
					t.Fatal("core victory bypassed an original outer defense")
				}
			}
			victory = true
			renderIntegrationPixels(t, g, "current-five-stage-final-victory")
		}
		if g.Screen == ShopScreen && !merchant {
			if !victory || cursor != len(codes) || w.Frame != 6605 || w.ScrollY != 163 || w.Equipment.Shield != 9 || w.Equipment.Lives != 2 || w.Money != 2350 || w.Score != 284750 || w.PendingExitDrops != 0 || !w.ShopReady || !w.ExitReady || !w.LevelFinished || !g.shopFinal || !g.shop.Ending {
				t.Fatal("final merchant omitted the complete native reward drain or ending gate")
			}
			merchant = true
		}
	}
	t.Fatalf("bounded complete journey did not start the next round: input%d/%d victory%v merchant%v round%v", cursor, len(codes), victory, merchant, nextRound)
}

// These bytes contain only ordinary direction/fire controls. Bit 6 marks the
// single READY acknowledgement after a normal ship loss; no game data is stored.
const currentFifthFinalValidationControls = "BRUBGQURARkFGQUQCBAAEAQYABAAEAAQABAAEAAQABAAEAAYCBQEEAAQCBgAGQERARACEQgZChgJGgoRChYGEAAQAhAJGgEa" +
	"BBgFFAgUBRAAEAASABgJGAoYABEGGQYZABAGGAEQBhgEGAAQARAGGQkRARkBEQEZAREJGQEYCBgEEAoVAhoIGAoSABkAFgEa" +
	"ChgAEAgYChACEAgQAhgIEAgSCBgBEgoQCBAKEAASAhoJGQkZAREBGQERBBUAGgoSABUFEggaABIBEQkZARUBFQgUCBIGFgoW" +
	"BhYAFAkUBBIAEAoaChoIEgQaARACEAAQBhkAEQAWCRIAEAESARAAEQERCRkAEAAQABAAEAAQABAIGgARABEBEQkaAhAAEAEa" +
	"ARAKEgoRCBgKGAoQAhAIEAAYAhAAGAoYAhgIEAgYCBgCEAgYABoIEAoSAhIAGAgQABAAGAkRCRkJGQkRABkEFQQYChICEAAQ" +
	"ABIAEAESAhoCEgISAhICEgISAhICEgISAhICEgISAhICEgISAhICEgISAhICEgISAhICEgISAhICEgISAhICEgISAhICEgIS" +
	"AhICEgISAhICEgISAhICEgISAhICEgISAhICEgISAhICEgISAhIAAAAAAAAAAAAAAAAAAAAAUAUVBRUFFQEVBRUBEQUZBREJ" +
	"FQkVCRUBEQEYBRgCFAQYABAAEgEUCRgEEAQQABAFEQURAREAEAAQABAAEAAQABAAEAIQChUEEAAWBBAKEAQUABYEEAIRARYG" +
	"FAASBhQCEgkZARUBEQURCRQIFAgQBBQEEAQQCBgAFAIWABAAEAAYBBACEAASAhUFFQURABAAEAAQABAAEAAQABAAEAAQABAA" +
	"EAAQABAAEAAQABAAEAAQABAAEAAQABAAEAAQABAAEAAQABAAEAAQABAAEAAQABAAEAIUBBQEFgAQABACEgAUBBQBEgAUBhAA" +
	"EgIQABgJEQURBREBGAAUABgEEAAQABAAEAAQABAAEAQaABAAEgQUBBQCFgQQBBUGEAQQBhAAEAASAhgJEQURARUJEAQYABQA" +
	"EgAUChAEGQASAhYGEgAQABAAEAAQABAAEAAQABAAEAAQABAAEAAQABAAEAAQABAAEAAQABAAEAAQABAAEAAQABAAEAAQABAA" +
	"EAAQABAAEAAQABgAGAgYCBgIGAgYCBgIGAoaCBoIGAgYChkKGAgaCBoCEgIWChICEgISAhICEgoSAhoCEgoaAhICEgIaChoC" +
	"EgISChIKEgIaAhoKEgoaChIKEgIaChIKEgoSAhICGgISChIKGgoaChoKEgIaAhIKEgoSAhICEgoaChICGgoaAhIKGgISChoK" +
	"GgIaAhIKGgoSAhICGgoSChICEgISAhoKEgoaAhICFgYSAhICEgIQAhoIGAgaChAABQUFBQUFBQUFBQUFBgUFAQEFCQkJCAkI" +
	"CAkJAgAFBAYEBgQEBgYFBQQEAA=="
