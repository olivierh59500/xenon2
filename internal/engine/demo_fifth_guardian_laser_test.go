package engine

import "testing"

func TestFifthGuardianAimRecognizesUsefulMountedLaserWhilePrimaryMissesOptional(t *testing.T) {
	w := fifthMiddleMountedCapability(t, ItemLaser)
	before := forecastIsolationDigest(w)
	var forecast WorldForecast
	opportunity, supported := presentationGuardianShotOpportunity(w, MotionInput{}, &forecast, 3)
	if !supported || !opportunity {
		t.Fatal("actual left laser core hit did not justify the guardian trigger")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("mounted guardian aim modified live state")
	}
	if forecast.State().FifthMiddle.Parts[5].Health >= 200 {
		t.Fatal("recognized beam did not execute a real core damage callback")
	}
	for _, shot := range forecast.State().Weapons.projectiles {
		if shot.Render.Kind == "small-shot" && shot.Owner == 0 && shot.Render.Active && forecast.State().fifthMiddleActors[5].Collision.Contains(int(shot.Render.X), int(shot.Render.Y)) {
			t.Fatal("fixture no longer distinguishes the missed primary ray")
		}
	}
}
