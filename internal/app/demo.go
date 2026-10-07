package app

import (
	"xenon2/internal/engine"
	"xenon2/internal/presentation"
	"xenon2/internal/shopui"
)

const titleDemoIdleUpdates = 60 * 60

// demoDirector supplies the same sampled controls as a human player. It never
// grants equipment, changes collisions or advances past an undefeated guardian.
type demoDirector struct {
	pilot        engine.DemoPilot
	presentation engine.PresentationPilot
	screen       Screen
	phase        presentation.Phase
	wait         int
	logicFrame   uint64
	level        int
	controls     inputFrame
	shop         *shopui.State
	shopWait     int
}

// DemoActive reports whether the normal-input pilot still owns the controls.
func (g *Game) DemoActive() bool { return g.Config.Demo }

func manualDemoActivity(manual inputFrame) bool {
	return manual.deviceActivity || manual.anyKey || manual.mousePressed || manual.escape || manual.music || manual.pause || manual.cheatMenu ||
		manual.confirm || manual.menuConfirm || manual.fire || manual.firePressed || manual.divePressed || manual.left || manual.right ||
		manual.leftPressed || manual.rightPressed || manual.upPressed || manual.downPressed || manual.referenceLevel != 0 || manual.referenceShop ||
		manual.cheatItem != engine.ItemNone || manual.cheatEnergy != 0 || manual.gameMotion != (engine.MotionInput{})
}

func (g *Game) advanceTitleIdle(manual inputFrame) {
	if g.Config.Demo || g.Screen != TitleScreen || g.paused || g.fade != nil && !g.fade.Done || manualDemoActivity(manual) {
		g.titleIdleUpdates = 0
		return
	}
	g.titleIdleUpdates++
	if g.titleIdleUpdates == titleDemoIdleUpdates {
		g.beginTitleDemo()
	}
}

func (g *Game) beginTitleDemo() {
	g.Config.Demo, g.Config.HumanDemo = true, true
	g.Config.Cheats = engine.CheatOptions{}
	g.Config.Level = 1
	g.applyCheatOptions()
	g.demo = nil
	g.titleIdleUpdates = 0
	g.menu = 0
	g.pendingPlayers = 1
	g.pendingFire, g.pendingDive = false, false
	g.director.BeginStart()
	g.Screen = PresentationScreen
	g.selectMusic()
}

func (g *Game) demoControls(manual inputFrame) inputFrame {
	if !g.Config.Demo {
		return manual
	}
	if manualDemoActivity(manual) {
		g.Config.Demo = false
		g.demo = nil
		return manual
	}
	if g.demo == nil {
		g.demo = &demoDirector{screen: g.Screen, phase: g.director.Phase, logicFrame: ^uint64(0)}
	}
	d := g.demo
	if g.fade != nil && !g.fade.Done {
		return inputFrame{}
	}
	if d.screen != g.Screen || d.phase != g.director.Phase {
		d.wait = 0
		d.screen = g.Screen
		d.phase = g.director.Phase
	}
	d.wait++
	switch g.Screen {
	case TitleScreen:
		if d.wait >= 60 {
			if g.menu != 0 {
				return inputFrame{upPressed: true}
			}
			d.wait = 0
			return inputFrame{menuConfirm: true}
		}
	case PresentationScreen:
		switch g.director.Phase {
		case presentation.ReadyMessage:
			if g.director.Data.MessageSteps[g.director.Step] == 17 && d.wait >= 45 {
				d.wait = 0
				return inputFrame{confirm: true, firePressed: true}
			}
		case presentation.ContinueHold:
			if d.wait >= 45 {
				d.wait = 0
				return inputFrame{confirm: true}
			}
		case presentation.Initials:
			if d.wait >= 18 {
				d.wait = 0
				return inputFrame{confirm: true}
			}
		case presentation.ScoresHold:
			if d.wait >= 90 {
				d.wait = 0
				return inputFrame{confirm: true}
			}
		}
	case ShopScreen:
		return g.demoShopControls()
	case LevelScreen:
		driver, ok := g.Driver.(*worldDriver)
		if !ok {
			return inputFrame{}
		}
		w := driver.world
		if d.logicFrame != w.Frame || d.level != w.Level.Number {
			var input engine.Input
			if g.Config.HumanDemo {
				d.presentation.PALRefreshes = g.Config.LogicPALRefreshes
				input = d.presentation.NormalInput(w)
			} else {
				input = d.pilot.NormalInput(w)
			}
			d.controls = inputFrame{gameMotion: input.Motion, fire: input.Fire, divePressed: input.Dive}
			d.logicFrame, d.level = w.Frame, w.Level.Number
		} else {
			d.controls.divePressed = false
		}
		return d.controls
	}
	return inputFrame{}
}

func (g *Game) demoShopControls() inputFrame {
	d, s := g.demo, g.shop
	if s == nil || s.Ending {
		return inputFrame{}
	}
	if d.shop != s {
		d.shop = s
		d.shopWait = 0
	}
	if s.Busy() || s.HandRemaining > 0 || s.Revealed < len(s.Dialogue) || s.DisplayMoney != *s.Money {
		d.shopWait = 0
		return inputFrame{}
	}
	d.shopWait++
	if d.shopWait < 24 {
		return inputFrame{}
	}
	d.shopWait = 0
	click := func(column, row int) inputFrame {
		if row == 4 {
			return inputFrame{mousePressed: true, mouseX: 20, mouseY: 180}
		}
		for _, cell := range g.Bundle.ShopScene.Cells {
			if cell.Column == column && cell.Row == row {
				return inputFrame{mousePressed: true, mouseX: cell.X + 2, mouseY: cell.Y + 2}
			}
		}
		return inputFrame{}
	}
	if s.Phase == shopui.Selling {
		return click(0, 4)
	}
	if s.Phase != shopui.Buying {
		return inputFrame{}
	}
	w := g.Driver.(*worldDriver).world
	// Select one purchase across both pages so a priority on the second page
	// cannot be replaced by a cheaper optional upgrade on the first page.
	item := demoShopPurchase(w.Equipment, w.Money, s.Rules)
	if item == engine.ItemNone {
		return click(0, 4)
	}
	for index, entry := range s.Entries {
		if !entry.Available || entry.More || entry.Item != item {
			continue
		}
		if s.QuoteValid && s.Quoted == index {
			return inputFrame{confirm: true}
		}
		return click(index%5, index/5)
	}
	for index, entry := range s.Entries {
		if entry.More {
			return click(index%5, index/5)
		}
	}
	return click(0, 4)
}
