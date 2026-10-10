package engine

import "testing"

func TestRetainedRouteReconsidersPredictedDamageEvenWhenKeysMatchOptional(t *testing.T) {
	w := path55PreparationScene(t)
	w.spawnEnemyShot(w.Player.X, w.Player.Y, EnemyShot{Speed: 0})
	var g expertRouteGuard
	plan := &g.plan
	plan.world, plan.pal, plan.count = w, 3, 6
	if err := plan.forecast.Load(w); err != nil {
		t.Fatal(err)
	}
	plan.before[0] = retainedGuardStateKey(w)
	damaged := false
	for pass := range plan.input {
		shield := plan.forecast.State().Equipment.Shield
		for range 3 {
			plan.forecast.AdvancePALTick()
		}
		r, err := plan.forecast.Advance(plan.input[pass])
		if err != nil || !r.Alive {
			t.Fatalf("isolated retained-route fixture failed: result%+v error%v", r, err)
		}
		damaged = damaged || r.Shield < shield
		plan.before[pass+1] = retainedGuardStateKey(plan.forecast.State())
	}
	if !damaged {
		t.Fatal("ordinary enemy projectile did not establish predicted damage")
	}
	before := forecastIsolationDigest(w)
	if _, ok := g.continuePlan(w, 3); ok || plan.count != 0 {
		t.Fatal("matching predicted keys retained a route containing known damage")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("reconsidering the retained route changed live gameplay")
	}
}
