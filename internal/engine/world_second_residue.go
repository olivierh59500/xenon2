package engine

// initializeSecondEmitterResidue retains unused slot data while initializing
// the world-anchored coordinates, phase, repeat count and emitter word.
func (w *World) initializeSecondEmitterResidue(actor *WorldActor) {
	r := &actor.Binding.Residue
	r.Counter, r.VerticalFraction, r.EmitterClock = 0, 0, 0
	if s := actor.fixedHatch; s != nil {
		r.X, r.Y = int16(s.X), int16(s.WorldY)
	}
	if s := actor.fixedPod; s != nil {
		r.X, r.Y, r.VerticalFraction = int16(s.X), int16(s.WorldY), uint16(s.Repeats)
	}
	w.storeWorldResidue(actor.Binding)
}

func (w *World) initializeSecondCreatureResidue(actor *WorldActor) {
	r := &actor.Binding.Residue
	r.X, r.Y, r.EmitterClock = int16(actor.X), int16(actor.Y), 0
	r.Health, r.PowerOrScore, r.WaveBonusToken = uint16(actor.Health), uint16(actor.Score), 0
	r.StrongHealth = actor.part.StrongHealth
	if s := actor.hatchCreature; s != nil {
		r.Counter, r.Direction = int16(s.Timer), int16(s.Direction)
	}
	if s := actor.podCreature; s != nil {
		r.Counter = int16(s.Phase)
	}
	w.storeWorldResidue(actor.Binding)
}

func (w *World) storeSecondSpecializedResidue(actor *WorldActor) {
	if actor.Binding.EntityID == 0 {
		return
	}
	r := &actor.Binding.Residue
	switch {
	case actor.fixedHatch != nil:
		s := actor.fixedHatch
		r.X, r.Y, r.Counter = int16(s.X), int16(s.WorldY), int16(s.Phase)
	case actor.fixedPod != nil:
		s := actor.fixedPod
		r.X, r.Y, r.Counter, r.VerticalFraction = int16(s.X), int16(s.WorldY), int16(s.Phase), uint16(s.Repeats)
	case actor.hatchCreature != nil:
		s := actor.hatchCreature
		r.X, r.Y, r.Counter, r.Direction = int16(s.X), int16(s.Y), int16(s.Timer), int16(s.Direction)
		r.Health, r.PowerOrScore = uint16(actor.Health), uint16(actor.Score)
	case actor.podCreature != nil:
		s := actor.podCreature
		r.X, r.Y, r.Counter = int16(s.X), int16(s.Y), int16(s.Phase)
		r.Health, r.PowerOrScore = uint16(actor.Health), uint16(actor.Score)
	}
	w.storeWorldResidue(actor.Binding)
	if !actor.Active {
		w.retireWorldActor(actor.Binding)
	}
}
