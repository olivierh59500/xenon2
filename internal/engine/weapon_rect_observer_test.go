package engine

import (
	"fmt"
	"testing"
)

// This isolated capability fixture starts with hypothetical equipment. It is
// not the actual carried profile or a connected fifth-stage admission. Pose
// replay uses the native controller without retaining its earlier shot events.
func fifthMiddleMountedCapability(t *testing.T, item Item) *World {
	t.Helper()
	data := originalWorldData(t, 5)
	equipment := NewEquipment()
	if !equipment.ApplyItem(item) {
		t.Fatal("cannot initialize hypothetical left-mounted capability")
	}
	data.InitialEquipment = &equipment
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	var recordFound bool
	for _, record := range w.Level.Encounters.Fixed {
		if record.EnemyKind != 5 {
			continue
		}
		w.ScrollY, w.PreviousScrollY, w.RenderScrollY = record.TriggerY, record.TriggerY, record.TriggerY
		w.MaximumScrollY, w.VisitedScrollY = record.TriggerY, record.TriggerY
		w.cursor = RestartEncounterCursor(w.ScrollY)
		if err := w.activateFifthGuardian(record, false); err != nil {
			t.Fatal(err)
		}
		recordFound = true
		break
	}
	if !recordFound {
		t.Fatal("original middle selector was not present")
	}
	w.Player.X, w.Player.Y = 182, 176
	if item == ItemCannon {
		w.Player.X = 178
	}
	w.Ready, w.MaterializationFrames = false, 0
	for pass := 0; w.FifthMiddle.Parts[0].Y < 62 && pass < 512; pass++ {
		w.Frame++
		w.ScrollY--
		w.FifthMiddle.Advance(w.fifthMiddleArt, w.ScrollY, 1, w.MaximumScrollY, w.Player.X, w.Player.Y, &w.random)
	}
	for pass := 0; pass < 2; pass++ {
		w.Frame++
		w.ScrollY--
		w.advanceFifthGuardian(false)
	}
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	w.cursor = RestartEncounterCursor(w.ScrollY)
	if w.FifthMiddle.Parts[5].Health != 200 || w.Equipment.Shield != 39 || w.Cheats.Enabled() {
		t.Fatal("pose initialization changed ordinary health or game rules")
	}
	return w
}

func TestMountedRectObserverDamagesOriginalFifthCoreWhilePrimaryMissesOptional(t *testing.T) {
	for _, item := range []Item{ItemCannon, ItemLaser} {
		t.Run(fmt.Sprint(item), func(t *testing.T) {
			live := fifthMiddleMountedCapability(t, item)
			before := forecastIsolationDigest(live)
			var observed, plain WorldForecast
			if err := observed.Load(live); err != nil {
				t.Fatal(err)
			}
			if err := plain.Load(live); err != nil {
				t.Fatal(err)
			}
			newID := live.nextActorID
			points, useful, rectangles := 0, 0, 0
			for pass := 0; pass < 18; pass++ {
				input := Input{Fire: true}
				for tick := 0; tick < 3; tick++ {
					observed.AdvancePALTick()
					plain.AdvancePALTick()
				}
				if _, err := observed.AdvanceWeaponObserved(input, func(event WeaponPointImpact) {
					if event.OwnerSlot == 0 && event.Kind == "small-shot" {
						points++
						if observed.State().fifthMiddleActors[5].Collision.Contains(event.X, event.Y) {
							t.Fatal("primary ray unexpectedly overlaps the narrow core")
						}
					}
				}, func(event WeaponRectImpact) {
					rectangles++
					if event.OwnerSlot != 1 || event.ProjectileID <= newID {
						t.Fatalf("new left mount lost its emission identity: %+v", event)
					}
					state := observed.State()
					for _, projectile := range state.Weapons.projectiles {
						if projectile.Render.ID != event.ProjectileID {
							continue
						}
						if item == ItemCannon && projectile.Owner != 0 {
							t.Fatal("observer metadata repurposed cannon's existing owner field")
						}
						if item == ItemLaser && (projectile.Owner != 1 || projectile.Binding.Residue.OwnerSlot != state.Weapons.mounts[1].Binding.Slot) {
							t.Fatal("observer metadata changed native laser owner lookup")
						}
					}
					if event.Area.Intersects(state.fifthMiddleActors[5].Collision) && guardianRectImpactUseful(state, event) {
						useful++
					}
					var nested WorldForecast
					if err := nested.Load(state); err != nil || nested.State().Weapons.context.ObservePointImpact != nil || nested.State().Weapons.context.ObserveRectImpact != nil {
						t.Fatal("cross-load retained an external observer")
					}
				}); err != nil {
					t.Fatal(err)
				}
				if _, err := plain.Advance(input); err != nil {
					t.Fatal(err)
				}
				if observed.State().Weapons.context.ObserveRectImpact != nil || observed.State().Weapons.context.ObservePointImpact != nil || forecastDigest(observed.State()) != forecastDigest(plain.State()) {
					t.Fatal("observing changed native state or retained a callback")
				}
				if observed.State().FifthMiddle.Parts[5].Health < 200 {
					break
				}
			}
			if points == 0 || rectangles == 0 || useful == 0 || observed.State().FifthMiddle.Parts[5].Health >= 200 {
				t.Fatalf("mounted native core damage absent: points%d rectangles%d useful%d health%d", points, rectangles, useful, observed.State().FifthMiddle.Parts[5].Health)
			}
			if forecastIsolationDigest(live) != before {
				t.Fatal("capability prediction changed the live source world")
			}
			t.Logf("item%d: native core200→%d; primary missed, mounted useful queries%d", item, observed.State().FifthMiddle.Parts[5].Health, useful)
		})
	}
}

func TestMountedRectOldProjectileIsReportedButNotCreditedAsNewOpportunityOptional(t *testing.T) {
	live := fifthMiddleMountedCapability(t, ItemLaser)
	// One ordinary pass leaves the first beam extending below the core.
	// Its next callback reaches the core after the trigger is released.
	for pass := 0; pass < 1; pass++ {
		for tick := 0; tick < 3; tick++ {
			live.AdvancePALTick()
		}
		if err := live.Step(Input{Fire: true}); err != nil {
			t.Fatal(err)
		}
	}
	if live.FifthMiddle.Parts[5].Health != 200 {
		t.Fatal("old-beam fixture damaged the core before the observed pass")
	}
	baselineID, before := live.nextActorID, forecastIsolationDigest(live)
	var forecast WorldForecast
	if err := forecast.Load(live); err != nil {
		t.Fatal(err)
	}
	reportedOld, oldUseful, creditedNew := 0, 0, 0
	for tick := 0; tick < 3; tick++ {
		forecast.AdvancePALTick()
	}
	if _, err := forecast.AdvanceWeaponObserved(Input{}, nil, func(event WeaponRectImpact) {
		useful := guardianRectImpactUseful(forecast.State(), event)
		if event.ProjectileID <= baselineID {
			reportedOld++
			if useful {
				oldUseful++
			}
		}
		// Filtering is a caller policy: the passive observer reports every
		// query, while a new-fire opportunity excludes older projectile IDs.
		if event.ProjectileID > baselineID && event.OwnerSlot >= 0 && useful {
			creditedNew++
		}
	}); err != nil {
		t.Fatal(err)
	}
	if reportedOld == 0 || oldUseful == 0 || creditedNew != 0 || forecast.State().FifthMiddle.Parts[5].Health != 197 {
		t.Fatalf("old/new attribution: reported%d useful%d credited%d core%d", reportedOld, oldUseful, creditedNew, forecast.State().FifthMiddle.Parts[5].Health)
	}
	if forecastIsolationDigest(live) != before {
		t.Fatal("old-projectile observation changed the live world")
	}
}

func TestRectImpactRespectsNativeSatelliteBordersAndFinalArmorOptional(t *testing.T) {
	w := originalGuardianTargetWorld(t, 4, false)
	actor := w.fourthMiddleActors[16]
	area := actor.Collision
	area.Bottom = area.Top + 2
	impact := WeaponRectImpact{Area: area, Damage: 2}
	before := w.FourthMiddle.Parts[16].Health
	if guardianRectImpactUseful(w, impact) {
		t.Fatal("rectangle touching armored border claimed interior damage")
	}
	if !w.weaponHitRect(area, 2, false) || w.FourthMiddle.Parts[16].Health != before {
		t.Fatal("native satellite border was not a harmless absorbing hit")
	}
	area.Top, area.Bottom = actor.Collision.Top+1, actor.Collision.Top+2
	impact.Area = area
	if !guardianRectImpactUseful(w, impact) {
		t.Fatal("clear native satellite strip rejected")
	}
	if !w.weaponHitRect(area, 2, false) || w.FourthMiddle.Parts[16].Health != before-2 {
		t.Fatal("native clear strip did not damage satellite")
	}

	w = fifthFinalPointScene(t, 0)
	band, mount := w.fifthFinalActors[2], w.fifthFinalActors[3]
	area = CollisionRect{Left: mount.Collision.Left, Right: mount.Collision.Right, Top: mount.Collision.Top, Bottom: band.Collision.Bottom}
	impact = WeaponRectImpact{Kind: "laser", Area: area, Damage: 3, AllTargets: true, Laser: true}
	beforeState := *w.FifthFinal
	if guardianRectImpactUseful(w, impact) {
		t.Fatal("later mount behind native armor was credited")
	}
	if !w.weaponHitLaser(area, 3) || *w.FifthFinal != beforeState {
		t.Fatal("native armor did not stop beam before later mount")
	}
}

func TestRectObserverClearsForPointOnlyBoundaryAndErrorOptional(t *testing.T) {
	w := fifthMiddleMountedCapability(t, ItemLaser)
	var forecast WorldForecast
	if err := forecast.Load(w); err != nil {
		t.Fatal(err)
	}
	forecast.State().Weapons.context.ObserveRectImpact = func(WeaponRectImpact) { t.Fatal("point-only call retained rectangle observer") }
	if _, err := forecast.AdvanceObserved(Input{Fire: true}, nil); err != nil {
		t.Fatal(err)
	}
	if forecast.State().Weapons.context.ObserveRectImpact != nil {
		t.Fatal("point-only return retained rectangle observer")
	}
	forecast.State().Ready = true
	if _, err := forecast.AdvanceWeaponObserved(Input{}, nil, func(WeaponRectImpact) { t.Fatal("boundary emitted rectangle query") }); err != nil {
		t.Fatal(err)
	}
	if forecast.State().Weapons.context.ObserveRectImpact != nil {
		t.Fatal("boundary retained observer")
	}
	forecast.State().Ready = false
	forecast.State().Equipment.FirePeriod = 0
	if _, err := forecast.AdvanceWeaponObserved(Input{}, nil, func(WeaponRectImpact) {}); err == nil {
		t.Fatal("fixture failed to reach source cadence error")
	}
	if forecast.State().Weapons.context.ObserveRectImpact != nil || forecast.State().Weapons.context.ObservePointImpact != nil {
		t.Fatal("error retained observer")
	}
}
