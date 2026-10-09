package engine

import "testing"

func TestAuxiliaryAimUsesInstalledRearAndSideEmissions(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		weapon Item
		x, y   int
	}{
		{"rear", ItemRearShot, 160, 160},
		{"left", ItemSideShot, 70, 97},
		{"right", ItemSideShot, 250, 97},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			w := testWorld(t)
			w.Player.X, w.Player.Y = 160, 100
			w.Actors = []*WorldActor{presentationTestEnemy(1, scenario.x, scenario.y)}
			if presentationAuxiliaryShotOpportunity(w, MotionInput{}) {
				t.Fatal("missing auxiliary weapon justified firing")
			}
			w.Equipment.ApplyItem(scenario.weapon)
			before := forecastDigest(w)
			if !presentationAuxiliaryShotOpportunity(w, MotionInput{}) {
				t.Fatal("installed auxiliary emission was ignored")
			}
			if forecastDigest(w) != before {
				t.Fatal("aiming modified the live world")
			}
			w.Actors[0].Active = false
			if presentationAuxiliaryShotOpportunity(w, MotionInput{}) {
				t.Fatal("retired enemy justified firing")
			}
		})
	}
}

func TestAuxiliaryAimRejectsImmuneAndClippedShots(t *testing.T) {
	w := testWorld(t)
	w.Player.X, w.Player.Y = 160, 176
	w.Equipment.ApplyItem(ItemRearShot)
	w.Actors = []*WorldActor{presentationTestEnemy(1, 160, 185)}
	if presentationAuxiliaryShotOpportunity(w, MotionInput{}) {
		t.Fatal("rear shot outside its original clipping bounds justified firing")
	}
	w.Player.Y = 100
	w.Actors[0] = presentationTestEnemy(2, 160, 160)
	w.Actors[0].part.DamageMode = "block-shot"
	if presentationAuxiliaryShotOpportunity(w, MotionInput{}) {
		t.Fatal("immune formation justified auxiliary fire")
	}
}
