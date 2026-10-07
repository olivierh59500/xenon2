package app

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"xenon2/internal/engine"
)

// inputFrame separates device sampling from one display-rate state update.
// Recorded checks can supply the same controls without OS keyboard injection.
type inputFrame struct {
	escape, music, pause, anyKey                      bool
	confirm, menuConfirm, firePressed, divePressed    bool
	left, right                                       bool
	leftPressed, rightPressed, upPressed, downPressed bool
	fire, mousePressed                                bool
	mouseX, mouseY                                    int
	referenceLevel                                    int
	referenceShop                                     bool
	cheatMenu                                         bool
	cheatItem                                         engine.Item
	cheatEnergy                                       int
	gameMotion                                        engine.MotionInput
}

func sampleInput() inputFrame {
	pressed := inpututil.IsKeyJustPressed
	i := inputFrame{
		escape: pressed(ebiten.KeyEscape), music: pressed(ebiten.KeyM), pause: pressed(ebiten.KeyP),
		anyKey:      len(inpututil.AppendJustPressedKeys(nil)) != 0,
		confirm:     pressed(ebiten.KeyEnter) || pressed(ebiten.KeySpace) || pressed(ebiten.KeyControl),
		menuConfirm: pressed(ebiten.KeyEnter) || pressed(ebiten.KeySpace),
		firePressed: pressed(ebiten.KeySpace) || pressed(ebiten.KeyControl), divePressed: pressed(ebiten.KeyAlt),
		left: ebiten.IsKeyPressed(ebiten.KeyArrowLeft), right: ebiten.IsKeyPressed(ebiten.KeyArrowRight),
		leftPressed: pressed(ebiten.KeyArrowLeft), rightPressed: pressed(ebiten.KeyArrowRight),
		upPressed: pressed(ebiten.KeyArrowUp), downPressed: pressed(ebiten.KeyArrowDown),
		fire:         ebiten.IsKeyPressed(ebiten.KeySpace) || ebiten.IsKeyPressed(ebiten.KeyControl),
		mousePressed: inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
	}
	i.mouseX, i.mouseY = ebiten.CursorPosition()
	i.gameMotion = engine.MotionInput{Left: i.left || ebiten.IsKeyPressed(ebiten.KeyA), Right: i.right || ebiten.IsKeyPressed(ebiten.KeyD), Up: ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW), Down: ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyS)}
	for n, key := range []ebiten.Key{ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3, ebiten.KeyDigit4, ebiten.KeyDigit5} {
		if pressed(key) {
			i.referenceLevel = n + 1
		}
	}
	i.referenceShop = pressed(ebiten.KeyF2)
	i.cheatMenu = pressed(ebiten.KeyF3)
	for _, binding := range []struct {
		key  ebiten.Key
		item engine.Item
	}{
		{ebiten.KeyF1, engine.ItemSpeedup}, {ebiten.KeyF2, engine.ItemHealth1},
		{ebiten.KeyF4, engine.ItemAutofire}, {ebiten.KeyF5, engine.ItemSuperNashwan},
		{ebiten.KeyF6, engine.ItemHealth2}, {ebiten.KeyF7, engine.ItemRearShot},
		{ebiten.KeyF8, engine.ItemMineSmall}, {ebiten.KeyF9, engine.ItemSideShot},
		{ebiten.KeyF10, engine.ItemExtraLife},
		{ebiten.KeyNumpad0, engine.ItemFlamer}, {ebiten.KeyNumpad1, engine.ItemElectroBall},
		{ebiten.KeyNumpad2, engine.ItemPowerup}, {ebiten.KeyNumpad3, engine.ItemMineLarge},
		{ebiten.KeyNumpad4, engine.ItemDoubleShot}, {ebiten.KeyNumpad5, engine.ItemCannon},
		{ebiten.KeyNumpad6, engine.ItemDive}, {ebiten.KeyNumpad7, engine.ItemMissileLauncher},
		{ebiten.KeyNumpad8, engine.ItemLaser}, {ebiten.KeyNumpad9, engine.ItemDrone},
	} {
		if pressed(binding.key) {
			i.cheatItem = binding.item
		}
	}
	if pressed(ebiten.KeyDelete) {
		i.cheatEnergy = 1
	}
	if pressed(ebiten.KeyInsert) {
		i.cheatEnergy = -1
	}
	return i
}
