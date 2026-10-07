package app

import (
	"bytes"
	"testing"
	"xenon2/internal/audio"
	"xenon2/internal/engine"
	"xenon2/internal/presentation"
)

func TestGameplayMusicPCMContinuesThroughCollisionDeathAndContinue(t *testing.T) {
	g := menuAdmittedGame(t, 1, 1)
	g.Config.Mute = false
	g.selectMusic()
	w := g.Driver.(*worldDriver).world
	w.Equipment.Lives = 1
	w.Score, w.DisplayScore = 500, 500
	reference, err := audio.NewA500Stream(g.Bundle.AudioBank, g.Bundle.Waveforms, 44100)
	if err != nil {
		t.Fatal(err)
	}
	if err := reference.PlayMusic("megablast-main"); err != nil {
		t.Fatal(err)
	}
	actual, expected := make([]byte, 735*4), make([]byte, 735*4)
	seenDeath, seenInitials, seenContinue, seenReady := false, false, false, false
	for pass := 0; pass < 5000; pass++ {
		controls := inputFrame{}
		if g.Screen == LevelScreen {
			controls.gameMotion = engine.MotionInput{Left: true, Up: true}
		}
		if g.director.Phase == presentation.Initials || g.director.Phase == presentation.ContinueHold {
			controls.confirm = true
			controls.anyKey = true
		}
		if seenContinue && g.readyRunning && g.director.Phase == presentation.ReadyMessage && g.director.Data.MessageSteps[g.director.Step] == 17 {
			seenReady = true
			controls.confirm = true
			controls.anyKey = true
		}
		advanceFrontend(t, g, controls)
		seenDeath = seenDeath || !w.PlayerAlive
		seenInitials = seenInitials || g.director.Phase == presentation.Initials
		seenContinue = seenContinue || g.director.Phase == presentation.ContinueHold
		if _, err := g.stream.Read(actual); err != nil {
			t.Fatal(err)
		}
		if _, err := reference.Read(expected); err != nil {
			t.Fatal(err)
		}
		// Gameplay effects use the right voices. Compare the unaffected left
		// music output so source death sounds cannot hide a replay phase reset.
		for offset := 0; offset < len(actual); offset += 4 {
			if !bytes.Equal(actual[offset:offset+2], expected[offset:offset+2]) {
				t.Fatalf("continuous left music PCM changed at display%d sample%d phase%s screen%d", pass, offset/4, g.director.Phase, g.Screen)
			}
		}
		if seenReady && g.Screen == LevelScreen && !g.View.Ready && !g.backdropOnly {
			if !seenDeath || !seenInitials || !seenContinue {
				t.Fatal("audio comparison did not traverse the actual collision/score/continue route")
			}
			t.Logf("Continuous original music PCM matched %d output frames through death, initials, continue and READY.", (pass+1)*735)
			return
		}
	}
	t.Fatal("bounded collision/continue route did not resume gameplay")
}

func TestInitialReadyAndShopReloadKeepNativeMusicAdmissionBoundaries(t *testing.T) {
	g := frontendGame(t)
	g.Config.Mute = false
	advanceFrontend(t, g, inputFrame{menuConfirm: true, anyKey: true})
	awaitFrontendBoundary(t, g, 800, "initial READY without gameplay music", func() bool {
		return g.readyRunning && g.director.Phase == presentation.ReadyMessage && g.director.Data.MessageSteps[g.director.Step] == 17
	})
	if g.gameMusicRunning || g.soundtrack != "" {
		t.Fatal("initial READY started the gameplay score before its source admission")
	}
	advanceFrontend(t, g, inputFrame{confirm: true, anyKey: true})
	awaitFrontendBoundary(t, g, 240, "initial gameplay replay admission", func() bool { return g.Screen == LevelScreen && !g.View.Ready })
	if !g.gameMusicRunning || g.soundtrack != "megablast-main" {
		t.Fatal("initial gameplay fade lost the original pre-fade music start")
	}
	g.stopGameplayMusic()
	g.beginLoading(1, true, headerReload)
	awaitFrontendBoundary(t, g, 240, "shop reload fade", func() bool { return g.Screen == LevelScreen && g.fade != nil && !g.fade.Done })
	if g.gameMusicRunning || g.soundtrack != "" {
		t.Fatal("reloaded gameplay score began before source fade completion")
	}
	awaitFrontendBoundary(t, g, 240, "shop reload replay admission", func() bool { return g.fade == nil || g.fade.Done })
	if !g.gameMusicRunning || g.soundtrack != "megablast-main" {
		t.Fatal("shop reload never restarted the source gameplay score")
	}
}
