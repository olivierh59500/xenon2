package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func thirdChainMaximumSeed(t *testing.T) RandomState {
	t.Helper()
	seed := NewRandomState()
	for range 128 {
		candidate := seed
		if candidate.Next()&6 == 6 && candidate.Next()&6 == 6 {
			return seed
		}
		seed.Next()
	}
	t.Fatal("source generator did not provide two legal maximum chain extensions")
	return RandomState{}
}

func thirdPreparedChainBounds(w *World, state ThirdChainState, art visualassets.ThirdChainArt) []CollisionRect {
	var bounds []CollisionRect
	for index, part := range state.Parts {
		if !part.Visible || part.Removed {
			continue
		}
		name := art.Bodies[state.Variant]
		if index == 7 {
			name = part.Animation.Sprite(art.Tails[state.Variant])
		}
		if box, ok := w.movingSpriteBoxes[name]; ok {
			bounds = append(bounds, ActorCollisionRect(box, part.X, part.Y))
		}
	}
	return bounds
}

func TestThirdChainPreparationUsesOriginalExtendedTailWidthsOptional(t *testing.T) {
	data := playableOriginalWorldData(t, 3)
	for _, fixture := range []struct {
		name       string
		x, variant int
		preparedX  int
	}{
		{"left-opening", 40, 0, 192},
		{"left-later", 56, 0, 208},
		{"right-opening", 264, 1, 128},
		{"right-later", 280, 1, 128},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			w, err := NewWorld(data)
			if err != nil {
				t.Fatal(err)
			}
			var record *visualassets.FixedEncounter
			for index := range data.Encounters.Fixed {
				candidate := &data.Encounters.Fixed[index]
				if candidate.EnemyKind == 1 && candidate.X == fixture.x && candidate.Variant == fixture.variant {
					record = candidate
					break
				}
			}
			if record == nil {
				t.Fatal("missing original chain encounter")
			}
			// Arrange the already-born chain 64 pixels above the player, before
			// its 32-pixel activation band. Its source constructor, eight parts,
			// sprite prefixes and subsequent extension remain in use.
			w.Player.X, w.Player.Y = 160, 120
			w.ScrollY = record.Y - 56
			w.spawnThirdChain(*record)
			var leader *WorldActor
			for _, actor := range w.Actors {
				if actor.thirdChainPart == 1 {
					leader = actor
					break
				}
			}
			if leader == nil {
				t.Fatal("source constructor did not create its chain leader")
			}
			before, player, random, pool, equipment := *leader.thirdChain, w.Player, w.RandomState(), *w.Pool, w.Equipment
			x, y := fixture.preparedX, 120
			selectedX, selectedY, found := thirdChainPreparation(w)
			initialCovered := w.Coverage.Touches(x, y, w.ScrollY, *w.Level.PlayerStencil)
			if found == initialCovered || found && (selectedX != x || selectedY != y) {
				t.Fatalf("preparation differs: %d/%d found %v covered %v, source-width lane %d/120", selectedX, selectedY, found, initialCovered, x)
			}
			if *leader.thirdChain != before || w.Player != player || w.RandomState() != random || *w.Pool != pool || w.Equipment != equipment {
				t.Fatal("preparation changed the live chain, ship, random, pool or gear")
			}
			art := data.FixedSprites.Third.Chain
			future, next := before, thirdChainMaximumSeed(t)
			var extended []CollisionRect
			for range 128 {
				future.Advance(ThirdChainInput{ScrollDelta: 1, PlayerY: 120}, art, &next)
				if future.Phase != 16 {
					continue
				}
				bounds := thirdPreparedChainBounds(w, future, art)
				center := thirdMiddlePlayerBounds(w, PlayerMotionState{X: 160, Y: 120})
				for _, box := range bounds {
					if center.Intersects(box) {
						extended = bounds
						break
					}
				}
				if len(extended) != 0 {
					break
				}
			}
			if len(extended) != 8 {
				t.Fatal("legal full source extension did not demonstrate the center-lane hazard")
			}
			for _, inertia := range []int{-6, -3, 0, 3, 6} {
				ship := thirdMiddlePlayerBounds(w, PlayerMotionState{X: x, Y: y, Inertia: inertia})
				if ship.Empty() {
					t.Fatal("source steering collision prefix is missing")
				}
				for _, box := range extended {
					if ship.Intersects(box) {
						t.Fatalf("prepared lane overlaps original chain width: inertia %d ship %+v chain %+v", inertia, ship, box)
					}
				}
			}
			clear, covered := 0, 0
			for camera := record.Y - 184; camera <= record.Y-56; camera++ {
				w.ScrollY = camera
				for index := range leader.thirdChain.Parts {
					leader.thirdChain.Parts[index].Y = record.Y - camera
				}
				before, random, pool := *leader.thirdChain, w.RandomState(), *w.Pool
				preparedX, preparedY, available := thirdChainPreparation(w)
				if w.Coverage.Touches(x, 120, camera, *w.Level.PlayerStencil) {
					covered++
					if available {
						t.Fatalf("covered preparation lane remained available at camera %d: %d/%d", camera, preparedX, preparedY)
					}
				} else {
					clear++
					if !available || preparedX != x || preparedY != 120 {
						t.Fatalf("clear original preparation lane was declined at camera %d: %d/%d available %v", camera, preparedX, preparedY, available)
					}
				}
				if *leader.thirdChain != before || w.RandomState() != random || *w.Pool != pool {
					t.Fatal("terrain availability check changed live chain, RNG or actor storage")
				}
			}
			if clear+covered != 129 || fixture.name == "left-later" && covered == 0 {
				t.Fatalf("fixture missed source preparation band or terrain fallback: clear %d covered %d", clear, covered)
			}
			t.Logf("Original preparation band: %d clear positions, %d covered positions", clear, covered)
			leader.thirdChain.Parts[0].Y = w.Player.Y - 65
			if _, _, found := thirdChainPreparation(w); found {
				t.Fatal("chain beyond the source preparation distance changed the target lane")
			}
			leader.Active = false
			if _, _, found := thirdChainPreparation(w); found {
				t.Fatal("retired source chain retained a preparation target")
			}
		})
	}
}
