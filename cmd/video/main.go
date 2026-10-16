// Command video records the game canvas and its original four-voice soundtrack.
package main

import (
	"flag"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/democonstructionkit/video"
	"log"
	"time"
	"xenon2/internal/app"
)

func main() {
	c := video.Config{Output: "recordings/xenon2-presentation.mp4", Title: "Xenon 2 Go", Width: app.ScreenWidth * 4, Height: app.ScreenHeight * 4, FPS: 60, TPS: 60, SampleRate: 44100, Duration: 3 * time.Minute, PosterAt: 75 * time.Second}
	c.Flags(flag.CommandLine)
	completeFirstLevel := flag.Bool("complete-level1", false, "record the complete first level, including both merchants, using ordinary demo controls")
	flag.Parse()
	if *completeFirstLevel {
		c.Duration = 0
		c.Title = "Xenon 2 Go - Complete Level 1"
	}
	if c.Duration <= 0 && !*completeFirstLevel {
		log.Fatal("a positive recording duration is required while the full campaign pilot is in development")
	}
	if err := video.Run(c, func() (ebiten.Game, error) {
		if *completeFirstLevel {
			return app.NewFirstLevelRecordingGame()
		}
		return app.NewRecordingGame()
	}); err != nil {
		log.Fatal(err)
	}
}
