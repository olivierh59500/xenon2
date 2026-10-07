// Package audioexport converts the local musical data into resolved voice
// events. It is an offline tool and is not a dependency of the game renderer.
package audioexport

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"xenon2/internal/audio"
)

const musicBase = 0x19f9e

type instrument struct {
	id    string
	scale uint16
	loop  string
}
type channel struct {
	order, position, cursor                              int
	remain, duration                                     uint16
	instrument                                           int
	note, transpose                                      byte
	arpeggio, arpStart, envelope, envCursor              int
	envRate, envRemain                                   byte
	slide                                                bool
	slideRate                                            int8
	slideDelay                                           byte
	slideOffset                                          int16
	vibrato, vibratoRate, vibratoExtent, vibratoPosition byte
	loopPending, enabled                                 bool
	period, volume                                       uint16
	current, next                                        string
}

type compiler struct {
	data                        []byte
	waveforms                   map[string][]byte
	bank                        audio.Bank
	instruments                 []instrument
	channels                    [4]channel
	tempo                       int16
	master                      uint16
	phase, increment, transpose byte
	fadeRate, fadeCounter       byte
	running                     bool
	events                      []audio.Event
	tick                        int
	err                         error
}

// DecodeMusic resolves the original four-channel score, including its volume
// envelopes, pitch modulation, rest/tie semantics and eventual order loops.
func DecodeMusic(data []byte) (*audio.Bank, map[string][]byte, error) {
	if len(data) < 0x3ae94 {
		return nil, nil, fmt.Errorf("audio source is truncated")
	}
	c := &compiler{data: data, waveforms: make(map[string][]byte)}
	c.bank = audio.Bank{Version: audio.SchemaVersion, TickRate: audio.TickRate, Clock: audio.PALClock}
	if err := c.samples(); err != nil {
		return nil, nil, err
	}
	for _, profile := range []struct {
		id     string
		master uint16
	}{{"megablast-main", 0x3040}, {"megablast-menu", 0x3f40}} {
		sequence, err := c.musicSequence(profile.id, profile.master)
		if err != nil {
			return nil, nil, err
		}
		c.bank.Music = append(c.bank.Music, sequence)
	}
	if err := c.sampledEffects(); err != nil {
		return nil, nil, err
	}
	if err := c.syntheticEffects(); err != nil {
		return nil, nil, err
	}
	if err := c.bank.Validate(c.waveforms); err != nil {
		return nil, nil, err
	}
	return &c.bank, c.waveforms, nil
}

func (c *compiler) musicSequence(id string, master uint16) (audio.Sequence, error) {
	c.tempo = int16(int8(c.data[0x1a828]))
	c.master = master
	c.phase, c.increment, c.transpose, c.fadeRate, c.fadeCounter = 0, 0, 0, 0, 0
	c.events = nil
	c.tick = 0
	c.running = true
	for i := range c.channels {
		order := musicBase + int(c.word(0x1a82a+i*2))
		cursor := musicBase + int(c.word(order))
		c.channels[i] = channel{order: order, position: 2, cursor: cursor, remain: 1, loopPending: true, arpStart: 0x1a706, arpeggio: 0x1a706, instrument: -1}
	}
	seen := make(map[string]int)
	loop := -1
	for c.tick < 30000 && c.running && c.err == nil {
		key := c.stateKey()
		if old, ok := seen[key]; ok {
			loop = old
			break
		}
		seen[key] = c.tick
		c.step()
		c.tick++
	}
	if c.err != nil {
		return audio.Sequence{}, c.err
	}
	if c.tick == 30000 {
		return audio.Sequence{}, fmt.Errorf("music does not terminate or repeat within ten minutes")
	}
	return audio.Sequence{ID: id, Ticks: c.tick, LoopTick: loop, PhaseIncrement: c.data[0x1a829], Events: c.events}, nil
}

func (c *compiler) samples() error {
	cursor := 0x1bb80
	for i := 0; i < 20; i++ {
		n := int(c.long(cursor))
		frequency := c.word(cursor + 4)
		if n < 2 || n%2 != 0 || cursor+6+n > len(c.data) || frequency == 0 {
			return fmt.Errorf("invalid music sample %d", i)
		}
		id := fmt.Sprintf("music-%02d", i)
		scale := uint16(3579545 / int(frequency))
		// The original initialization tunes these four instrument periods.
		switch i {
		case 5, 6:
			scale -= 8
		case 7, 11:
			scale -= 2
		}
		c.addSample(id, c.data[cursor+6:cursor+6+n])
		loop := "silence"
		if int32(c.long(0x1a738+i*12+4)) >= 0 {
			loop = id
		}
		c.instruments = append(c.instruments, instrument{id: id, scale: scale, loop: loop})
		cursor += 6 + n
	}
	c.addSample("silence", make([]byte, 64))
	return nil
}

func (c *compiler) sampledEffects() error {
	cursor := 0x2ba9c
	for i := 0; i < 18; i++ {
		n := int(c.long(cursor))
		frequency := c.word(cursor + 4)
		if n < 2 || n%2 != 0 || cursor+6+n > len(c.data) || frequency == 0 {
			return fmt.Errorf("invalid sampled effect %d", i)
		}
		id := fmt.Sprintf("sampled-effect-%02d", i)
		c.addSample(id, c.data[cursor+6:cursor+6+n])
		period := uint16(3579545 / int(frequency))
		ticks := int(uint8((n&65535)*50/int(frequency) + 1))
		if ticks < 1 {
			return fmt.Errorf("sampled effect duration overflow")
		}
		// The effect timer is decremented by the first VBL after the trigger.
		ticks--
		events := []audio.Event{{Tick: 0, Channel: 0, Kind: "period", Period: period}, {Tick: 0, Channel: 0, Kind: "volume", Volume: 64}, {Tick: 0, Channel: 0, Kind: "start", Sample: id}, {Tick: 0, Channel: 0, Kind: "loop", Sample: "silence"}, {Tick: ticks, Channel: 0, Kind: "stop"}}
		sequence := audio.Sequence{ID: id, Ticks: ticks + 1, LoopTick: -1, Events: events}
		// This effect is intentionally sustained until another event replaces it.
		if c.data[0x1b352+i*16+13] != 0 {
			sequence.Ticks = 2
			sequence.LoopTick = 1
			sequence.Events = events[:4]
			sequence.Events[3].Sample = id
		}
		c.bank.Effects = append(c.bank.Effects, sequence)
		cursor += 6 + n
	}
	return nil
}

func (c *compiler) addSample(id string, pcm []byte) {
	c.waveforms[id] = append([]byte(nil), pcm...)
	c.bank.Samples = append(c.bank.Samples, audio.Sample{ID: id, File: id + ".pcm", Bytes: len(pcm)})
}

func (c *compiler) stateKey() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d/%d/%d/%d/%d/%d|", c.tempo, c.master, c.phase, c.transpose, c.fadeRate, c.fadeCounter)
	for _, s := range c.channels {
		fmt.Fprintf(&b, "%v|", s)
	}
	return b.String()
}

func (c *compiler) emit(ch int, kind string, sample string, period, volume uint16) {
	c.events = append(c.events, audio.Event{Tick: c.tick, Channel: ch, Kind: kind, Sample: sample, Period: period, Volume: volume})
}

func (c *compiler) step() {
	phase := uint16(c.phase) + uint16(c.increment)
	c.phase = byte(phase)
	if phase > 255 {
		return
	}
	if c.fadeRate != 0 {
		c.fadeCounter--
		if c.fadeCounter == 0 {
			if c.master == 0 || c.master == 1 {
				c.stopAll()
				return
			}
			c.master--
			c.fadeCounter = c.fadeRate
		}
	}
	for i := range c.channels {
		s := &c.channels[i]
		if !s.loopPending {
			s.loopPending = true
			c.loop(i, c.instruments[s.instrument].loop)
		}
		s.remain--
		if s.remain == 0 {
			c.note(i)
			continue
		}
		if s.remain == 1 {
			if c.byte(s.cursor) != 0x83 && s.enabled {
				s.enabled = false
				c.emit(i, "stop", "", 0, 0)
			}
			continue
		}
		c.modulate(i)
	}
}

func (c *compiler) note(i int) {
	s := &c.channels[i]
	s.slide = false
	for guard := 0; guard < 256 && c.err == nil; guard++ {
		v := c.byte(s.cursor)
		s.cursor++
		if v < 128 {
			if s.instrument < 0 || s.envelope == 0 {
				c.err = fmt.Errorf("note precedes instrument or envelope")
				return
			}
			s.note = v
			s.envCursor = s.envelope
			volume := c.byte(s.envCursor)
			s.envCursor++
			s.envRemain = s.envRate
			c.volume(i, uint16(volume)*c.master>>6)
			c.period(i, c.pitch(i, 0))
			inst := c.instruments[s.instrument]
			c.loop(i, inst.id)
			c.start(i, inst.id)
			s.loopPending = false
			s.remain = s.duration
			return
		}
		switch {
		case v >= 0xe0:
			s.duration = uint16(int16(v-0xdf) * c.tempo)
		case v >= 0xb0:
			s.instrument = int(v - 0xb0)
			if s.instrument >= len(c.instruments) {
				c.err = fmt.Errorf("unknown music instrument %d", s.instrument)
			}
		case v >= 0xa0:
			s.envelope = musicBase + int(c.word(0x1ba5c+int(v-0xa0)*2))
			s.envRate = c.byte(s.envelope - 1)
		case v >= 0x90:
			s.arpStart = musicBase + int(c.word(0x1a6e6+int(v-0x90)*2))
			s.arpeggio = s.arpStart
		case v == 0x80:
			at := s.order + s.position
			if c.word(at) == 0 {
				at = s.order
				s.position = 0
			}
			s.cursor = musicBase + int(c.word(at))
			s.position += 2
		case v == 0x81:
			s.slide = true
			s.slideOffset = 0
			s.slideRate = int8(c.byte(s.cursor))
			s.slideDelay = c.byte(s.cursor + 1)
			s.cursor += 2
		case v == 0x82:
			s.remain = s.duration
			c.loop(i, "silence")
			return
		case v == 0x83:
			s.remain = s.duration
			if !s.enabled {
				c.start(i, s.next)
			}
			return
		case v == 0x84:
			c.stopAll()
			return
		case v == 0x85:
			c.transpose = c.byte(s.cursor)
			s.cursor++
		case v == 0x86:
			s.vibrato = 255
			s.vibratoRate = c.byte(s.cursor)
			s.vibratoExtent = c.byte(s.cursor + 1)
			s.vibratoPosition = 0
			s.cursor += 2
		case v == 0x87:
			s.vibrato = 0
		case v == 0x88:
			s.transpose = c.byte(s.cursor)
			s.cursor++
		case v == 0x89:
			s.order = musicBase + int(c.word(s.cursor))
			s.position = 0
			s.cursor += 2
		case v == 0x8a:
			c.tempo = int16(int8(c.byte(s.cursor)))
			s.cursor++
		case v == 0x8b:
			c.fadeRate = c.byte(s.cursor)
			c.fadeCounter = c.fadeRate
			s.cursor++
		default:
			c.err = fmt.Errorf("invalid music command %s", strconv.FormatInt(int64(v), 16))
		}
	}
	if c.err == nil {
		c.err = fmt.Errorf("music command chain does not reach a note")
	}
}

func (c *compiler) pitch(i int, arpeggio byte) uint16 {
	s := &c.channels[i]
	n := s.note + c.transpose + s.transpose + arpeggio
	if int(n) >= 72 {
		c.err = fmt.Errorf("music pitch %d leaves period table", n)
		return 1
	}
	return uint16(uint32(c.word(0x1a656+int(n)*2)) * uint32(c.instruments[s.instrument].scale) >> 10)
}

func (c *compiler) modulate(i int) {
	s := &c.channels[i]
	arpeggio := c.byte(s.arpeggio)
	s.arpeggio++
	if arpeggio&128 != 0 {
		s.arpeggio = s.arpStart
	}
	period := c.pitch(i, arpeggio&127)
	if s.slide {
		if s.slideDelay > 0 {
			s.slideDelay--
		} else {
			s.slideOffset += int16(s.slideRate)
			period -= uint16(s.slideOffset)
		}
	}
	if s.vibrato != 0 {
		if int8(s.vibrato) < 0 {
			s.vibratoPosition += s.vibratoRate
			if s.vibratoPosition == s.vibratoExtent {
				s.vibrato ^= 128
			}
		} else {
			s.vibratoPosition -= s.vibratoRate
			if s.vibratoPosition == 0 {
				s.vibrato ^= 128
			}
		}
		if s.vibratoPosition == 0 {
			s.vibrato ^= 1
		}
		delta := uint16(int16(int8(s.vibratoPosition)))
		if s.vibrato&1 != 0 {
			period += delta
		} else {
			period -= delta
		}
	}
	c.period(i, period)
	old := s.envRemain
	s.envRemain--
	if old == 0 {
		s.envRemain = s.envRate
		v := c.byte(s.envCursor)
		if v < 128 {
			s.envCursor++
		}
		c.volume(i, uint16(v&127)*c.master>>6)
	}
}

func (c *compiler) start(i int, id string) {
	s := &c.channels[i]
	s.enabled = true
	s.current = id
	c.emit(i, "start", id, 0, 0)
}
func (c *compiler) loop(i int, id string) { c.channels[i].next = id; c.emit(i, "loop", id, 0, 0) }
func (c *compiler) period(i int, v uint16) {
	if c.channels[i].period != v {
		c.channels[i].period = v
		c.emit(i, "period", "", v, 0)
	}
}
func (c *compiler) volume(i int, v uint16) {
	// Paula ignores the upper volume bits; bit six selects full volume.
	if v&64 != 0 {
		v = 64
	} else {
		v &= 63
	}
	if c.channels[i].volume != v {
		c.channels[i].volume = v
		c.emit(i, "volume", "", 0, v)
	}
}
func (c *compiler) stopAll() {
	c.running = false
	for i := range c.channels {
		c.channels[i].enabled = false
		c.emit(i, "stop", "", 0, 0)
	}
}
func (c *compiler) byte(at int) byte {
	if at < 0 || at >= len(c.data) {
		c.err = fmt.Errorf("musical data is truncated")
		return 0
	}
	return c.data[at]
}
func (c *compiler) word(at int) uint16 {
	if at < 0 || at+2 > len(c.data) {
		c.err = fmt.Errorf("musical word is truncated")
		return 0
	}
	return binary.BigEndian.Uint16(c.data[at:])
}
func (c *compiler) long(at int) uint32 {
	if at < 0 || at+4 > len(c.data) {
		c.err = fmt.Errorf("musical sample header is truncated")
		return 0
	}
	return binary.BigEndian.Uint32(c.data[at:])
}
