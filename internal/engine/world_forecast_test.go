package engine

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"xenon2/internal/visualassets"
)

// forecastDigest traverses semantic values without comparing process addresses
// or callback identities. Production cloning uses typed copies, not reflection.
func forecastFieldDifferences(t *testing.T, got, want *World) {
	t.Helper()
	left, right := reflect.ValueOf(got).Elem(), reflect.ValueOf(want).Elem()
	for index := 0; index < left.NumField(); index++ {
		name := left.Type().Field(index).Name
		if name == "weaponTargets" || name == "weaponTargetActors" {
			continue
		}
		if forecastValueDigest(left.Field(index)) != forecastValueDigest(right.Field(index)) {
			t.Logf("differing world field %s", name)
		}
	}
}

func forecastValueDigest(value reflect.Value) string { return forecastDigestValue(value) }

func forecastMapKey(value reflect.Value) string {
	switch value.Kind() {
	case reflect.String:
		return value.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("i%d", value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("u%d", value.Uint())
	default:
		panic("unsupported forecast digest map key: " + value.Type().String())
	}
}

func forecastDigest(value any) string { return forecastDigestValue(reflect.ValueOf(value)) }

func forecastDigestValue(value reflect.Value) string {
	var out strings.Builder
	seen := make(map[uintptr]bool)
	var write func(reflect.Value)
	write = func(v reflect.Value) {
		if !v.IsValid() {
			out.WriteString("nil;")
			return
		}
		switch v.Kind() {
		case reflect.Func:
			out.WriteString("callback;")
		case reflect.Pointer:
			if v.IsNil() {
				out.WriteString("nil;")
				return
			}
			name := v.Type().Elem().PkgPath()
			if name == "image" || strings.HasPrefix(name, "image/") {
				out.WriteString("art;")
				return
			}
			if strings.HasSuffix(name, "/visualassets") {
				write(v.Elem())
				return
			}
			ptr := v.Pointer()
			if seen[ptr] {
				out.WriteString("link;")
				return
			}
			seen[ptr] = true
			write(v.Elem())
		case reflect.Interface:
			if v.IsNil() {
				out.WriteString("nil;")
			} else {
				write(v.Elem())
			}
		case reflect.Struct:
			typ := v.Type()
			out.WriteString(typ.String())
			out.WriteByte('{')
			for i := 0; i < v.NumField(); i++ {
				name := typ.Field(i).Name
				if typ.Name() == "World" && (name == "weaponTargets" || name == "weaponTargetActors") {
					continue
				}
				if typ.Name() == "WeaponRuntime" && (name == "context" || name == "newID") {
					continue
				}
				out.WriteString(name)
				out.WriteByte(':')
				write(v.Field(i))
			}
			out.WriteByte('}')
		case reflect.Slice, reflect.Array:
			fmt.Fprintf(&out, "[%d]", v.Len())
			for i := 0; i < v.Len(); i++ {
				write(v.Index(i))
			}
		case reflect.Map:
			keys := v.MapKeys()
			sort.Slice(keys, func(i, j int) bool { return forecastMapKey(keys[i]) < forecastMapKey(keys[j]) })
			fmt.Fprintf(&out, "map%d{", len(keys))
			for _, k := range keys {
				write(k)
				write(v.MapIndex(k))
			}
			out.WriteByte('}')
		case reflect.String:
			fmt.Fprintf(&out, "%q;", v.String())
		case reflect.Bool:
			fmt.Fprintf(&out, "%t;", v.Bool())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fmt.Fprintf(&out, "%d;", v.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			fmt.Fprintf(&out, "%d;", v.Uint())
		case reflect.Float32, reflect.Float64:
			fmt.Fprintf(&out, "%g;", v.Float())
		default:
			fmt.Fprintf(&out, "%s;", v.Kind())
		}
	}
	write(value)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(out.String())))
}

func forecastOriginalScene(t testing.TB, level int, arena bool) *World {
	t.Helper()
	data := originalWorldData(t, level)
	if !arena {
		data = playableOriginalWorldData(t, level)
	}
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	w.Ready, w.MaterializationFrames = false, 0
	w.Player.X, w.Player.Y = 160, 176
	if arena {
		switch level {
		case 1:
			w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 3000, 3344, 3344
			w.cursor = RestartEncounterCursor(w.ScrollY)
		case 2:
			w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 2700, 2896, 2896
			w.cursor = RestartEncounterCursor(w.ScrollY)
		case 3:
			w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 2800, 2816, 2816
			w.cursor = RestartEncounterCursor(w.ScrollY)
			if err := w.activateThirdMiddle(); err != nil {
				t.Fatal(err)
			}
			for _, r := range w.Level.Encounters.Fixed {
				if r.EnemyKind == 1 {
					w.spawnThirdChain(r)
					break
				}
			}
		case 4:
			if err := w.activateFourthGuardian(false); err != nil {
				t.Fatal(err)
			}
		case 5:
			if err := w.activateFifthGuardian(visualassets.FixedEncounter{Y: 2336}, false); err != nil {
				t.Fatal(err)
			}
		}
	}
	return w
}

func TestWorldForecastMatchesOriginalStepsAcrossFiveLevelsOptional(t *testing.T) {
	for level := 1; level <= 5; level++ {
		for _, arena := range []bool{false, true} {
			t.Run(fmt.Sprintf("level%d/arena%v", level, arena), func(t *testing.T) {
				live := forecastOriginalScene(t, level, arena)
				var f WorldForecast
				before := forecastDigest(live)
				if err := f.Load(live); err != nil {
					t.Fatal(err)
				}
				if forecastDigest(live) != before {
					t.Fatal("loading changed the live world")
				}
				for pass := 0; pass < 24; pass++ {
					input := Input{Fire: pass%2 == 0, Motion: demoDirections[(pass/3)%len(demoDirections)]}
					for range 3 {
						live.AdvancePALTick()
					}
					liveBefore := forecastDigest(live)
					for range 3 {
						f.AdvancePALTick()
					}
					if _, err := f.Advance(input); err != nil {
						t.Fatal(err)
					}
					if forecastDigest(live) != liveBefore {
						t.Fatal("forecast callbacks changed the live state")
					}
					if err := live.Step(input); err != nil {
						t.Fatal(err)
					}
					if got, want := forecastDigest(f.State()), forecastDigest(live); got != want {
						forecastFieldDifferences(t, f.State(), live)
						t.Fatalf("forecast diverged at pass%d: got%s want%s", pass+1, got, want)
					}
					if !live.PlayerAlive || live.Ready || live.GameOver || live.ShopReady || live.LevelFinished {
						break
					}
				}
			})
		}
	}
}

func TestWorldForecastPreservesPALStrobeAndLiveStateOptional(t *testing.T) {
	live := forecastOriginalScene(t, 3, false)
	live.applyCarrierReward(18)
	var forecast WorldForecast
	if err := forecast.Load(live); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 31; tick++ {
		before := forecastDigest(live)
		forecast.AdvancePALTick()
		if forecastDigest(live) != before {
			t.Fatal("forecast PAL effect changed the live world")
		}
		live.AdvancePALTick()
		if got, want := forecastDigest(forecast.State()), forecastDigest(live); got != want {
			forecastFieldDifferences(t, forecast.State(), live)
			t.Fatalf("PAL tick%d diverged: got%s want%s", tick+1, got, want)
		}
		if tick < 30 {
			if _, err := forecast.Advance(Input{Fire: true}); err != nil {
				t.Fatal(err)
			}
			if err := live.Step(Input{Fire: true}); err != nil {
				t.Fatal(err)
			}
			if live.Frame != 0 || forecast.State().Frame != 0 {
				t.Fatal("palette strobe advanced ordinary gameplay")
			}
		}
	}
	if live.ScreenClearFrames != 0 || forecast.State().ScreenClearFrames != 0 {
		t.Fatal("source strobe never completed")
	}
}

func TestWorldForecastReloadAndRejectMissingState(t *testing.T) {
	var forecast WorldForecast
	if _, err := forecast.Advance(Input{}); err == nil {
		t.Fatal("unloaded forecast accepted a gameplay command")
	}
	if err := forecast.Load(nil); err == nil || forecast.State() != nil {
		t.Fatal("nil source did not invalidate the forecast")
	}
	first := testWorld(t)
	first.Ready = false
	if err := forecast.Load(first); err != nil {
		t.Fatal(err)
	}
	original := forecastDigest(first)
	for range 6 {
		if _, err := forecast.Advance(Input{Fire: true}); err != nil {
			t.Fatal(err)
		}
	}
	if forecastDigest(first) != original {
		t.Fatal("first prediction changed its source")
	}
	second := testWorld(t)
	second.Ready = false
	second.Player.X = 230
	second.Money = 1234
	if err := forecast.Load(second); err != nil {
		t.Fatal(err)
	}
	if forecast.State().Player.X != 230 || forecast.State().Money != 1234 {
		t.Fatal("reload retained old world state")
	}
	if err := forecast.Load(nil); err == nil || forecast.State() != nil {
		t.Fatal("failed load exposed stale prediction state")
	}
}

func TestWorldForecastStopsAtLifecycleBoundaries(t *testing.T) {
	for _, fixture := range []struct {
		name    string
		prepare func(*World)
		want    ForecastBoundary
	}{
		{"ready", func(w *World) { w.Ready = true }, ForecastReady},
		{"death", func(w *World) { w.PlayerAlive = false }, ForecastPlayerDeath},
		{"game over", func(w *World) { w.GameOver = true }, ForecastGameOver},
		{"merchant", func(w *World) { w.ShopReady = true }, ForecastShop},
		{"completed stage", func(w *World) { w.LevelFinished = true }, ForecastLevelFinished},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			live := testWorld(t)
			fixture.prepare(live)
			var forecast WorldForecast
			if err := forecast.Load(live); err != nil {
				t.Fatal(err)
			}
			before := forecastDigest(forecast.State())
			for range 3 {
				forecast.AdvancePALTick()
				result, err := forecast.Advance(Input{Fire: true, Motion: MotionInput{Right: true}})
				if err != nil || result.Boundary != fixture.want {
					t.Fatalf("boundary mismatch: %+v err%v", result, err)
				}
			}
			if forecastDigest(forecast.State()) != before {
				t.Fatal("forecast crossed an external lifecycle boundary")
			}
		})
	}
}

func BenchmarkWorldForecastOriginalArenas(b *testing.B) {
	for level := 1; level <= 5; level++ {
		b.Run(fmt.Sprintf("level%d", level), func(b *testing.B) {
			source := forecastOriginalScene(b, level, true)
			for _, horizon := range []int{0, 6, 24} {
				b.Run(fmt.Sprintf("passes%d", horizon), func(b *testing.B) {
					var forecast WorldForecast
					b.ReportAllocs()
					for b.Loop() {
						if err := forecast.Load(source); err != nil {
							b.Fatal(err)
						}
						for pass := 0; pass < horizon; pass++ {
							for range 3 {
								forecast.AdvancePALTick()
							}
							result, err := forecast.Advance(Input{Fire: pass%2 == 0})
							if err != nil {
								b.Fatal(err)
							}
							if result.Boundary != ForecastRunning {
								break
							}
						}
					}
				})
			}
		})
	}
}

func TestWorldForecastStopsAfterActualLethalContact(t *testing.T) {
	live, _ := contactDeathWorld(t, 16, true, false, false, false)
	var forecast WorldForecast
	before := forecastDigest(live)
	if err := forecast.Load(live); err != nil {
		t.Fatal(err)
	}
	result, err := forecast.Advance(Input{Motion: MotionInput{Right: true}, Fire: true})
	if err != nil || result.Alive || result.Boundary != ForecastPlayerDeath {
		t.Fatalf("actual contact boundary missing: %+v err%v", result, err)
	}
	if forecastDigest(live) != before {
		t.Fatal("forecast death changed the live player or contact target")
	}
	stopped := forecastDigest(forecast.State())
	for range 24 {
		forecast.AdvancePALTick()
		if _, err := forecast.Advance(Input{Fire: true}); err != nil {
			t.Fatal(err)
		}
	}
	if forecastDigest(forecast.State()) != stopped || forecast.State().Equipment.Lives != live.Equipment.Lives {
		t.Fatal("forecast advanced death animation into a frontend life transition")
	}
}
