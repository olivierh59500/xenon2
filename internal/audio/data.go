// Package audio replays code-free voice events and signed PCM resources.
package audio

import "fmt"

const (
	SchemaVersion = 1
	PALClock      = 3546895
	TickRate      = 50
)

// Sample describes a signed, eight-bit mono waveform stored separately.
type Sample struct {
	ID    string `json:"id"`
	File  string `json:"file"`
	Bytes int    `json:"bytes"`
}

// Event updates one voice at a musical tick. Loop changes affect the next
// buffer reload; they do not interrupt a sample already playing.
type Event struct {
	Tick    int    `json:"tick"`
	Channel int    `json:"channel"`
	Kind    string `json:"kind"`
	Sample  string `json:"sample,omitempty"`
	Period  uint16 `json:"period,omitempty"`
	Volume  uint16 `json:"volume,omitempty"`
}

// Sequence is a finite or looping list of already resolved voice events.
type Sequence struct {
	ID             string  `json:"id"`
	Ticks          int     `json:"ticks"`
	LoopTick       int     `json:"loop_tick"`
	PhaseIncrement byte    `json:"phase_increment,omitempty"`
	Events         []Event `json:"events"`
}

// Bank contains no source addresses, executable bytes or tracker bytecode.
type Bank struct {
	Version  int        `json:"version"`
	TickRate int        `json:"tick_rate"`
	Clock    int        `json:"clock"`
	Samples  []Sample   `json:"samples"`
	Music    []Sequence `json:"music"`
	Effects  []Sequence `json:"effects"`
}

// Validate checks event references before an audio stream is constructed.
func (b *Bank) Validate(waveforms map[string][]byte) error {
	if b.Version != SchemaVersion || b.TickRate != TickRate || b.Clock != PALClock {
		return fmt.Errorf("unsupported audio bank clock or version")
	}
	ids := make(map[string]bool)
	for _, s := range b.Samples {
		if s.ID == "" || ids[s.ID] || s.Bytes < 2 || s.Bytes != len(waveforms[s.ID]) {
			return fmt.Errorf("invalid audio sample %q", s.ID)
		}
		ids[s.ID] = true
	}
	for _, group := range [][]Sequence{b.Music, b.Effects} {
		for _, s := range group {
			if s.ID == "" || s.Ticks < 1 || s.LoopTick < -1 || s.LoopTick >= s.Ticks {
				return fmt.Errorf("invalid audio sequence %q", s.ID)
			}
			last := -1
			for _, e := range s.Events {
				if e.Tick < last || e.Tick < 0 || e.Tick >= s.Ticks || e.Channel < 0 || e.Channel >= 4 {
					return fmt.Errorf("invalid audio event in %q", s.ID)
				}
				last = e.Tick
				switch e.Kind {
				case "start", "loop":
					if !ids[e.Sample] {
						return fmt.Errorf("unknown audio sample %q", e.Sample)
					}
				case "period":
					if e.Period == 0 {
						return fmt.Errorf("zero audio period")
					}
				case "volume":
					if e.Volume > 64 {
						return fmt.Errorf("invalid audio volume")
					}
				case "stop":
				default:
					return fmt.Errorf("unsupported voice event %q", e.Kind)
				}
			}
		}
	}
	return nil
}
