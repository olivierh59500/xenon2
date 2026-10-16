package audioexport

import (
	"fmt"

	"xenon2/internal/audio"
)

// syntheticEffects resolves the original waveform selection, alternating
// pitch increments, pitch resets and volume envelopes into ordinary events.
func (c *compiler) syntheticEffects() error {
	return c.syntheticEffectsAt(syntheticLayout{records: 0x1b792, envelopes: 0x1b9be, relativeBase: musicBase, waveforms: 0x2abe0, prefix: "synthesized-effect", wavePrefix: "waveform"})
}

type syntheticLayout struct {
	records, envelopes, relativeBase, waveforms int
	prefix, wavePrefix                          string
}

func (c *compiler) syntheticEffectsAt(layout syntheticLayout) error {
	waves := make(map[string]string)
	for index := 0; index < 23; index++ {
		at := layout.records + index*24
		delta, initial := int16(c.word(at)), int16(c.word(at+2))
		pair := c.long(at + 4)
		period := int16(c.word(at + 8))
		wavePair := c.long(at + 10)
		resetRate, alternateRate := c.byte(at+14), c.byte(at+15)
		selector, waveSelector := c.byte(at+16), c.byte(at+17)
		remaining, volumeRate := c.byte(at+18), c.byte(at+19)
		envelope := layout.relativeBase + int(c.word(layout.envelopes+int(c.byte(at+20))*2))
		words := int(c.word(at + 22))
		if words < 1 || words > 4096 {
			return fmt.Errorf("invalid synthesized effect length")
		}
		resetCounter, alternateCounter, volumeCounter := resetRate, alternateRate, byte(1)
		var events []audio.Event
		lastPeriod, lastVolume := uint16(0), uint16(0)
		lastWave := ""
		active := true
		tick := 0
		loop := -1
		seen := make(map[string]int)
		for ; tick < 30000 && active; tick++ {
			state := fmt.Sprintf("%d/%d/%d/%d/%d/%d/%d/%d/%d/%d/%s/%t", period, selector, waveSelector, remaining, envelope, resetCounter, alternateCounter, volumeCounter, lastPeriod, lastVolume, lastWave, tick == 0)
			if previous, ok := seen[state]; ok {
				loop = previous
				break
			}
			seen[state] = tick
			if remaining != 0 {
				remaining--
				if remaining == 0 {
					events = append(events, audio.Event{Tick: tick, Channel: 0, Kind: "stop"})
					active = false
					break
				}
			}
			if alternateRate != 0 {
				alternateCounter--
				if alternateCounter == 0 {
					alternateCounter = alternateRate
					selector = selector>>1 | selector<<7
					change := uint16(pair)
					if selector&128 == 0 {
						change = uint16(pair >> 16)
					}
					period += int16(change)
				}
			}
			period += delta
			if resetRate != 0 {
				resetCounter--
				if resetCounter == 0 {
					resetCounter = resetRate
					period = initial
				}
			}
			volumeCounter--
			if volumeCounter == 0 {
				volumeCounter = volumeRate
				v := c.byte(envelope)
				if v == 0 {
					events = append(events, audio.Event{Tick: tick, Channel: 0, Kind: "stop"})
					active = false
					break
				}
				if v < 128 {
					envelope++
					volume := uint16(v)
					if volume&64 != 0 {
						volume = 64
					} else {
						volume &= 63
					}
					if volume != lastVolume {
						events = append(events, audio.Event{Tick: tick, Channel: 0, Kind: "volume", Volume: volume})
						lastVolume = volume
					}
				}
			}
			frequency := period
			if frequency < 124 {
				frequency = 124
			}
			if uint16(frequency) != lastPeriod {
				lastPeriod = uint16(frequency)
				events = append(events, audio.Event{Tick: tick, Channel: 0, Kind: "period", Period: lastPeriod})
			}
			waveSelector = waveSelector>>1 | waveSelector<<7
			waveOffset := int16(uint16(wavePair))
			if waveSelector&128 == 0 {
				waveOffset = int16(uint16(wavePair >> 16))
			}
			start := layout.waveforms + int(waveOffset)
			if start < 0 || start+words*2 > len(c.data) {
				return fmt.Errorf("synthesized waveform leaves sample bank")
			}
			key := fmt.Sprintf("%d:%d", start, words)
			id, ok := waves[key]
			if !ok {
				id = fmt.Sprintf("%s-%02d", layout.wavePrefix, len(waves))
				waves[key] = id
				c.addSample(id, c.data[start:start+words*2])
			}
			if id != lastWave {
				events = append(events, audio.Event{Tick: tick, Channel: 0, Kind: "loop", Sample: id})
				lastWave = id
			}
			if tick == 0 {
				events = append(events, audio.Event{Tick: tick, Channel: 0, Kind: "start", Sample: id})
			}
		}
		if active && loop < 0 {
			return fmt.Errorf("synthesized effect %d does not finish within ten minutes", index)
		}
		length := tick + 1
		if loop >= 0 {
			length = tick
		}
		c.bank.Effects = append(c.bank.Effects, audio.Sequence{ID: fmt.Sprintf("%s-%02d", layout.prefix, index), Ticks: length, LoopTick: loop, Events: events})
	}
	return c.err
}
