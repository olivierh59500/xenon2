package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

func TestWorldOrdinaryDamageFlashesThenUsesSourceCenteredExplosion(t *testing.T) {
	w := testWorld(t)
	w.Level.Actors.Atlas.Sprites = []visualassets.SpriteRegion{{Name: "enemy", Width: 24, Height: 18, AnchorX: 4, AnchorY: 6}}
	w.commonAnimations["explosion-large"] = visualassets.NamedActorAnimation{Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "large-explosion", Duration: 1}}}}
	actor := &WorldActor{Active: true, ActorList: "moving", Atlas: "moving", Sprite: "enemy", X: 70, Y: 90, Health: 3, Score: 100, part: &visualassets.ActorPart{ResourceTag: 200, StrongHealth: true, DamageMode: "individual"}}
	if err := w.bindWorldActor(actor); err != nil {
		t.Fatal(err)
	}
	w.Actors = []*WorldActor{actor}
	w.damageActor(actor, 1)
	if !actor.Active || !actor.Flash || len(w.Actors) != 1 || w.Score != 0 {
		t.Fatal("nonlethal damage must flash without spawning death effects")
	}
	w.damageActor(actor, 2)
	if actor.Active || w.Score != 100 || len(w.Actors) != 2 || w.Actors[0].Sprite != "large-explosion" || w.Actors[0].X != 78 || w.Actors[0].Y != 92 {
		t.Fatal("lethal damage must create the original centered large explosion")
	}
}

func TestWorldGroupDeathSkipsLinkedInvisiblePieces(t *testing.T) {
	w := testWorld(t)
	w.commonAnimations["explosion-small"] = visualassets.NamedActorAnimation{Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "small-explosion"}}}}
	head := &WorldActor{Active: true, ActorList: "moving", Health: 1, Score: 100, part: &visualassets.ActorPart{ResourceTag: 200, DamageMode: "group"}}
	visible := &WorldActor{Active: true, ActorList: "moving", leader: head, part: &visualassets.ActorPart{ResourceTag: 204, DamageMode: "group"}}
	linked := &WorldActor{Active: true, ActorList: "moving", leader: head, part: &visualassets.ActorPart{ResourceTag: 208, DamageMode: "group", Linked: true}}
	for _, actor := range []*WorldActor{head, visible, linked} {
		if err := w.bindWorldActor(actor); err != nil {
			t.Fatal(err)
		}
	}
	w.Actors = []*WorldActor{head, visible, linked}
	w.damageActor(visible, 1)
	effects := 0
	for _, actor := range w.Actors {
		if actor.Sprite == "small-explosion" {
			effects++
		}
	}
	if effects != 2 || head.Active || visible.Active || linked.Active || w.Score != 100 {
		t.Fatal("group death must destroy all members but omit the linked piece's effect")
	}
}

func TestWorldOrdinaryDeathNativeCallbackOptional(t *testing.T) {
	nativeCombatRows(t, "ordinary-death-trace.csv", func(v []int64) {
		w := testWorld(t)
		w.Level.Actors.Atlas.Sprites = []visualassets.SpriteRegion{{Name: "enemy", Width: 24, Height: 18, AnchorX: 4, AnchorY: 6}}
		for _, name := range []string{"explosion-small", "explosion-large"} {
			w.commonAnimations[name] = visualassets.NamedActorAnimation{Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: name}}}}
		}
		actor := &WorldActor{Active: true, ActorList: "moving", Atlas: "moving", Sprite: "enemy", X: 70, Y: 90, Health: 3, Score: 100, part: &visualassets.ActorPart{ResourceTag: 200, StrongHealth: v[0] != 0, DamageMode: "individual"}}
		if err := w.bindWorldActor(actor); err != nil {
			t.Fatal(err)
		}
		w.Actors = []*WorldActor{actor}
		w.damageActor(actor, uint16(v[1]))
		if actor.Health != int(v[2]) || actor.Active != (v[3] == 0) || len(w.Actors)-1 != int(v[4]) || w.Score != int(v[8]) {
			t.Fatalf("death callback differs%v: %+v", v, actor)
		}
		if v[4] != 0 {
			if w.Actors[0].X != float64(v[5]) || w.Actors[0].Y != float64(v[6]) {
				t.Fatal("source explosion center differs")
			}
			want := "sampled-effect-05"
			if v[7] == 131 {
				want = "sampled-effect-03"
			}
			if w.SoundRequests[2] != want {
				t.Fatal("source explosion sound differs")
			}
		}
	})
}
