package app

// beginGameplayMusic is an explicit native replay admission. READY after a
// ship loss leaves this state active, so its existing score never restarts.
func (g *Game) beginGameplayMusic() {
	g.gameMusicRunning = true
	g.gameMusicAfterFade = false
	g.selectMusic()
}

func (g *Game) stopGameplayMusic() {
	g.gameMusicRunning = false
	g.gameMusicAfterFade = true
	g.stream.StopEffects()
	g.stream.StopMusic()
	g.soundtrack = ""
}

func (g *Game) selectMusic() {
	id := ""
	if !g.Config.Mute {
		if g.music && g.gameMusicRunning {
			id = "megablast-main"
		}
		if !g.gameMusicRunning && g.Screen == PresentationScreen && !g.readyRunning && !g.gameOverRunning && g.director.AttractMusic() {
			id = "megablast-menu"
		}
	}
	if id == g.soundtrack {
		return
	}
	old := g.soundtrack
	g.soundtrack = id
	if id == "" {
		if old == "megablast-menu" {
			g.stream.StopEffects()
		}
		g.stream.StopMusic()
		return
	}
	// Original score initialization stops every effect voice before restarting
	// the order lists. Fractional music phase remains owned by the stream.
	g.stream.StopEffects()
	if err := g.stream.PlayMusic(id); err != nil {
		g.err = err
	}
}
