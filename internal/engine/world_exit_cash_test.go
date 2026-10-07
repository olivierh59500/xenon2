package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

func TestWorldGuardianCashMatchesNativeTailPairsOptional(t *testing.T) {
	w := testWorld(t)
	for _, name := range []string{"cash-small", "cash-large"} {
		w.commonAnimations[name] = visualassets.NamedActorAnimation{Ending: "hold", Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: name}}}}
	}
	w.spawnExitCash(3)
	order := w.Pool.EntityIDs(ActorPoolProjectile, nil)
	rows := 0
	nativeCombatRows(t, "exit-cash-trace.csv", func(v []int64) {
		index := int(v[0])
		coin := w.Collectibles[index]
		if order[index] != coin.ID || w.Pool.Slot(coin.Binding.Slot).ResourceTag != int16(v[1]) || coin.X != float64(v[2]) || coin.Y != float64(v[3]) || coin.Motion.Mode != int(v[4]) || w.random.A != uint32(v[5]) || w.random.B != uint32(v[6]) || w.PendingExitDrops != int(v[7]) {
			t.Fatalf("cash pair differs%v: %+v", v, coin)
		}
		if index > 0 && coin.Order >= w.Collectibles[index-1].Order {
			t.Fatal("source tail rendering order differs")
		}
		rows++
	})
	if rows != 6 {
		t.Fatalf("incomplete cash comparisons:%d", rows)
	}
}
