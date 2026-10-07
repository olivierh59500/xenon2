package engine

import (
	"reflect"
	"testing"

	"xenon2/internal/visualassets"
)

func TestEncounterUnsortedOrderAndReverse(t *testing.T) {
	e := visualassets.Encounters{Moving: []visualassets.Wave{
		{TriggerY: 98, EnemyKind: 1}, {TriggerY: 95, EnemyKind: 2},
		{TriggerY: 99, EnemyKind: 3}, {TriggerY: 100, EnemyKind: 4},
		{TriggerY: 90, EnemyKind: 5},
	}}
	c := EncounterCursor{MovingHighWater: 100, FixedHighWater: 100}
	var got []int
	activate := func(y int) {
		c.Activate(y, &e, func(w visualassets.Wave) { got = append(got, w.EnemyKind) }, nil)
	}
	activate(95)
	activate(99)
	activate(94)
	activate(90)
	if !reflect.DeepEqual(got, []int{1, 2, 3, 5}) {
		t.Fatalf("original-order activations got %v", got)
	}
}

func TestCheckpointRecreatesVisibleFixedObjects(t *testing.T) {
	e := visualassets.Encounters{
		Moving: []visualassets.Wave{{TriggerY: 1000}, {TriggerY: 1001}},
		Fixed:  []visualassets.FixedEncounter{{TriggerY: 999}, {TriggerY: 1000}, {TriggerY: 1191}, {TriggerY: 1192}},
	}
	c := RestartEncounterCursor(1000)
	var moving, fixed []int
	c.Activate(1000, &e,
		func(w visualassets.Wave) { moving = append(moving, w.TriggerY) },
		func(f visualassets.FixedEncounter) { fixed = append(fixed, f.TriggerY) })
	if !reflect.DeepEqual(moving, []int{1000}) || !reflect.DeepEqual(fixed, []int{1000, 1191}) {
		t.Fatalf("checkpoint activation moving=%v, fixed=%v", moving, fixed)
	}
}
