package engine

import "xenon2/internal/visualassets"

// EncounterCursor tracks the furthest activated scroll positions. Original
// encounter streams are intentionally unsorted and must be scanned in order.
type EncounterCursor struct {
	MovingHighWater int
	FixedHighWater  int
}

func NewEncounterCursor() EncounterCursor {
	return EncounterCursor{MovingHighWater: 0x7fee, FixedHighWater: 0x7fee}
}

// RestartEncounterCursor recreates visible scenery around a checkpoint while
// allowing only the moving waves crossed after that checkpoint to reappear.
func RestartEncounterCursor(scrollY int) EncounterCursor {
	return EncounterCursor{MovingHighWater: scrollY + 1, FixedHighWater: scrollY + 192}
}

// Activate visits records whose triggers lie in [scrollY, previousHighWater).
// Moving backwards neither respawns records nor raises either high-water mark.
// Callbacks run in original stream order, after the current actor update pass.
func (c *EncounterCursor) Activate(scrollY int, encounters *visualassets.Encounters, moving func(visualassets.Wave), fixed func(visualassets.FixedEncounter)) {
	if scrollY >= c.MovingHighWater {
		return
	}
	if moving != nil {
		for _, wave := range encounters.Moving {
			if wave.TriggerY >= scrollY && wave.TriggerY < c.MovingHighWater {
				moving(wave)
			}
		}
	}
	c.MovingHighWater = scrollY
	if fixed != nil {
		for _, encounter := range encounters.Fixed {
			if encounter.TriggerY >= scrollY && encounter.TriggerY < c.FixedHighWater {
				fixed(encounter)
			}
		}
	}
	if scrollY < c.FixedHighWater {
		c.FixedHighWater = scrollY
	}
}
