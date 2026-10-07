// Command xenon2 presents exported game resources through Ebitengine.
package main

import (
	"flag"
	"fmt"
	"os"

	"xenon2/internal/app"
)

func main() {
	data := flag.String("data", "", "optional directory containing exported PNG/JSON/PCM resources")
	frames := flag.Int("frames", 0, "stop after this many display updates; zero runs normally")
	screenshot := flag.String("screenshot", "", "save the final frame as PNG")
	level := flag.Int("level", 1, "reference level, from one to five")
	view := flag.String("view", "attract", "initial view: menu, attract, level or shop")
	mute := flag.Bool("mute", false, "disable audio output")
	flag.Parse()
	if flag.NArg() != 0 {
		fail(fmt.Errorf("unexpected positional argument"))
	}
	var bundle *app.Bundle
	var err error
	if *data == "" {
		bundle, err = app.LoadEmbedded()
	} else {
		bundle, err = app.LoadFS(os.DirFS(*data))
	}
	if err != nil {
		fail(err)
	}
	config := app.Config{Level: *level, Frames: *frames, Screenshot: *screenshot, Mute: *mute}
	switch *view {
	case "menu":
		config.StartScreen = app.TitleScreen
	case "level":
		config.StartScreen = app.LevelScreen
	case "shop":
		config.StartScreen = app.ShopScreen
	case "attract":
		config.StartScreen = app.PresentationScreen
	default:
		fail(fmt.Errorf("unknown view %q", *view))
	}
	if err = app.Run(bundle, config); err != nil {
		fail(err)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
