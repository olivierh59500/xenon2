package engine

import "xenon2/internal/visualassets"

// InitializeFixedSpriteResidue changes only fields written by the original
// constructors. The hidden fractions and unused counters survive slot reuse.
func InitializeFixedSpriteResidue(kind visualassets.FixedSpriteKind, state FixedSpriteState, previous ActorResidue) ActorResidue {
	r := previous
	r.X, r.Y = int16(state.X), int16(state.Y)
	r.EmitterClock = 0
	switch kind.Behavior {
	case "scroll-bounce-attack":
		r.Counter, r.Direction, r.VerticalFraction = int16(state.VelocityY), int16(state.Amplitude), uint16(state.Distance)
		r.WaveBonusToken = 0
	case "horizontal-sweeper":
		r.Counter, r.Direction = 0, int16(state.VelocityX)
		r.WaveBonusToken = 0
	case "vertical-oscillator":
		r.Counter, r.Direction, r.VerticalFraction = 0, int16(state.VelocityY), 0
		r.WaveBonusToken = 0
	case "extending-beam":
		r.Counter, r.VerticalFraction = 0, 0
		return r
	}
	r.Health, r.PowerOrScore, r.StrongHealth = uint16(kind.Health), uint16(kind.Score), kind.StrongHealth
	return r
}

// FixedSpriteRenderLayout separates the actor's source position from the beam
// renderer's shaft and anchored tip positions.
type FixedSpriteRenderLayout struct {
	TipX, TipY, ShaftX, ShaftY, ShaftColumns int
}

func FixedSpriteBeamLayout(state FixedSpriteState) FixedSpriteRenderLayout {
	layout := FixedSpriteRenderLayout{TipX: state.X + state.Phase*16, TipY: state.Y + 8, ShaftX: (state.X >> 3) * 8, ShaftY: state.Y, ShaftColumns: state.Phase}
	if state.Variant == 1 {
		layout.TipX = state.X - state.Phase*16 + 15
		layout.ShaftX = ((state.X - state.Phase*16 + 16) >> 3) * 8
	}
	return layout
}

func (w *World) initializeFixedSpriteResidue(actor *WorldActor) {
	actor.Binding.Residue = InitializeFixedSpriteResidue(*actor.fixedKind, actor.fixedState, actor.Binding.Residue)
	if actor.fixedKind.Behavior == "extending-beam" {
		actor.Health, actor.Score, actor.WaveToken = int(actor.Binding.Residue.Health), int(actor.Binding.Residue.PowerOrScore), actor.Binding.Residue.WaveBonusToken
	}
	w.storeWorldResidue(actor.Binding)
}

func (w *World) storeFixedSpriteResidue(actor *WorldActor) {
	if actor.Binding.EntityID == 0 {
		return
	}
	s, r := &actor.fixedState, &actor.Binding.Residue
	r.X, r.Y = int16(s.X), int16(s.Y)
	r.Health, r.PowerOrScore, r.WaveBonusToken = uint16(actor.Health), uint16(actor.Score), actor.WaveToken
	r.SetFireState(s.FireAccumulator, r.FireRate())
	switch actor.fixedKind.Behavior {
	case "scroll-bounce-attack":
		r.Counter, r.Direction, r.VerticalFraction = int16(s.VelocityY), int16(s.Amplitude), uint16(s.Distance)
	case "horizontal-sweeper":
		r.Counter, r.Direction = int16(s.Phase), int16(s.VelocityX)
	case "vertical-oscillator":
		r.Direction = int16(s.VelocityY)
	case "extending-beam":
		r.Counter, r.Direction = int16(s.Phase), int16(s.PhaseDirection)
	}
	w.storeWorldResidue(actor.Binding)
	if !actor.Active {
		w.retireWorldActor(actor.Binding)
	}
}

func (w *World) composeFixedSprite(actor *WorldActor) {
	if actor.fixedKind.Behavior != "extending-beam" {
		return
	}
	s := &actor.fixedState
	layout := FixedSpriteBeamLayout(*s)
	previous := layout
	if actor.beamLayoutValid {
		previous = actor.beamLayout
	}
	actor.beamLayout, actor.beamLayoutValid = layout, true
	actor.DrawKind = "assembly"
	actor.Extras = actor.Extras[:0]
	actor.Extras = append(actor.Extras, WorldSpriteAttachment{Atlas: "fixed", Sprite: s.Animation.Sprite(FixedSpriteAnimation(*s, *actor.fixedKind)), X: float64(layout.TipX), Y: float64(layout.TipY), PreviousX: float64(previous.TipX), PreviousY: float64(previous.TipY), Interpolate: true})
	actor.Sprite = ""
	actor.TileOverlays = actor.TileOverlays[:0]
	v, ok := fixedSpriteVariant(*actor.fixedKind, s.Variant)
	if !ok || v.Cover == nil {
		return
	}
	for column := 0; column < layout.ShaftColumns; column++ {
		// Keep the native number of shaft tiles for the current pose; only
		// their placement follows the preceding render endpoint.
		actor.TileOverlays = append(actor.TileOverlays, WorldTileOverlay{Patch: *v.Cover, X: float64(layout.ShaftX + column*16), Y: float64(layout.ShaftY), PreviousX: float64(previous.ShaftX + column*16), PreviousY: float64(previous.ShaftY), Interpolate: true})
	}
}
