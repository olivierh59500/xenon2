//go:build !xenon2_logic_only

package app

import (
	"fmt"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

const logicOnlyTests = false

// ReadPixels needs a running graphics context. Execute the test suite from
// Update, matching Ebitengine's own internal testing run-loop pattern.
type renderTestLoop struct {
	tests    *testing.M
	exitCode int
}

func (g *renderTestLoop) Update() error            { g.exitCode = g.tests.Run(); return ebiten.Termination }
func (*renderTestLoop) Draw(*ebiten.Image)         {}
func (*renderTestLoop) Layout(int, int) (int, int) { return ScreenWidth, ScreenHeight }

func TestMain(m *testing.M) {
	loop := &renderTestLoop{tests: m, exitCode: 1}
	if err := ebiten.RunGame(loop); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(loop.exitCode)
}
