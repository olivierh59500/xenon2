package app

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"xenon2/internal/controls"
	"xenon2/internal/engine"
)

// inputFrame separates device sampling from one display-rate state update.
// Recorded checks can supply the same controls without OS keyboard injection.
type inputFrame struct {
	escape, music, pause, anyKey                      bool
	deviceActivity                                    bool
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
	i.deviceActivity = len(inpututil.AppendPressedKeys(nil)) != 0 || len(ebiten.AppendTouchIDs(nil)) != 0
	for button := ebiten.MouseButtonLeft; button <= ebiten.MouseButtonMax; button++ {
		i.deviceActivity = i.deviceActivity || ebiten.IsMouseButtonPressed(button)
	}
	wheelX, wheelY := ebiten.Wheel()
	i.deviceActivity = i.deviceActivity || wheelX != 0 || wheelY != 0
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
		{ebiten.KeyB, engine.ItemBomb}, {ebiten.KeyH, engine.ItemHomingMissile},
		{ebiten.KeyO, engine.ItemProtection}, {ebiten.KeyV, engine.ItemBitmapShades},
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

// mergeTouchInput routes mobile edges through the same controls as the keyboard.
// Menu and shop navigation keep their original one-press selection semantics.
func mergeTouchInput(i inputFrame, touch controls.Frame) inputFrame {
	i.deviceActivity = i.deviceActivity || touch.AnyPressed || touch.Tap || touch.X != 0 || touch.Y != 0
	for _, held := range touch.Held {
		i.deviceActivity = i.deviceActivity || held
	}
	i.anyKey = i.anyKey || touch.AnyPressed
	i.escape = i.escape || touch.Pressed[controls.Menu]
	i.pause = i.pause || touch.Pressed[controls.Pause]
	i.cheatMenu = i.cheatMenu || touch.Pressed[controls.Cheats]
	i.confirm = i.confirm || touch.Pressed[controls.Enter] || touch.Pressed[controls.Fire]
	i.menuConfirm = i.menuConfirm || touch.Pressed[controls.Enter] || touch.Pressed[controls.Fire]
	i.firePressed = i.firePressed || touch.Pressed[controls.Fire]
	i.divePressed = i.divePressed || touch.Pressed[controls.Dive]
	i.fire = i.fire || touch.Held[controls.Fire]
	i.left = i.left || touch.X < 0
	i.right = i.right || touch.X > 0
	i.leftPressed = i.leftPressed || touch.LeftPressed
	i.rightPressed = i.rightPressed || touch.RightPressed
	i.upPressed = i.upPressed || touch.UpPressed
	i.downPressed = i.downPressed || touch.DownPressed
	i.gameMotion.Left = i.gameMotion.Left || touch.X < 0
	i.gameMotion.Right = i.gameMotion.Right || touch.X > 0
	i.gameMotion.Up = i.gameMotion.Up || touch.Y < 0
	i.gameMotion.Down = i.gameMotion.Down || touch.Y > 0
	if touch.Tap {
		i.mousePressed, i.mouseX, i.mouseY = true, int(touch.TapX), int(touch.TapY)
	}
	return i
}
