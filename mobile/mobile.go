// Package mobile attaches the Go game to Android's Ebitengine view.
package mobile

import (
	"github.com/hajimehoshi/ebiten/v2"
	enginemobile "github.com/hajimehoshi/ebiten/v2/mobile"
	"xenon2/internal/app"
)

func init() {
	ebiten.SetTPS(60)
	enginemobile.SetGame(app.NewMobileGame(app.Config{Level: 1, StartScreen: app.PresentationScreen}))
}

// Dummy ensures gomobile generates bindings for the mobile package.
func Dummy() {}
