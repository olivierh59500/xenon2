package audio

import (
	"encoding/binary"
	"fmt"
	"io"
	"sort"
	"sync"
)

type voice struct {
	current, next string
	position      int
	// Remaining Paula cycles are scaled by the output rate for exact integer
	// deadlines. Period writes affect the next byte, preserving this countdown.
	remaining      int64
	period, volume uint16
	enabled        bool
}

type playback struct {
	sequence   *Sequence
	age, index int
	phase      byte
	active     bool
}

// Stream is a concurrency-safe, stereo, signed 16-bit little-endian reader.
// Its clock belongs to audio consumption, independently of rendering updates.
type Stream struct {
	mu                        sync.Mutex
	bank                      *Bank
	waveforms                 map[string][]byte
	sampleRate                int
	remaining, clockRemainder int
	music                     playback
	musicPhase                byte
	effects                   [4]playback
	queued                    [4]*Sequence
	queuedStops               uint8
	shadow, voices            [4]voice
	filter                    *a500Filter
}

func NewStream(bank *Bank, waveforms map[string][]byte, sampleRate int) (*Stream, error) {
	if sampleRate < 8000 || sampleRate > 192000 {
		return nil, fmt.Errorf("unsupported output sample rate")
	}
	if err := bank.Validate(waveforms); err != nil {
		return nil, err
	}
	return &Stream{bank: bank, waveforms: waveforms, sampleRate: sampleRate}, nil
}

// NewA500Stream adds the permanent Amiga 500 output response to the digital
// voice mixer. NewStream retains raw PCM for register/dispatch comparisons.
func NewA500Stream(bank *Bank, waveforms map[string][]byte, sampleRate int) (*Stream, error) {
	stream, err := NewStream(bank, waveforms, sampleRate)
	if err != nil {
		return nil, err
	}
	stream.filter = newA500Filter(sampleRate)
	return stream, nil
}

// PlayMusic restarts a named soundtrack without exposing source driver state.
func (s *Stream) PlayMusic(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.playMusic(id)
}

func (s *Stream) playMusic(id string) error {
	for i := range s.bank.Music {
		if s.bank.Music[i].ID == id {
			s.music = playback{sequence: &s.bank.Music[i], active: true, phase: s.musicPhase}
			s.shadow = [4]voice{}
			for ch := range s.voices {
				if !s.effects[ch].active {
					s.voices[ch] = voice{}
				}
			}
			return nil
		}
	}
	return fmt.Errorf("unknown soundtrack %q", id)
}

// RestartMusic releases current effects and initializes the score while
// retaining requests waiting for their next interrupt, as the original caller.
func (s *Stream) RestartMusic(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopEffects(false)
	return s.playMusic(id)
}

func (s *Stream) StopMusic() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.music = playback{}
	s.shadow = [4]voice{}
	for ch := range s.voices {
		if !s.effects[ch].active {
			s.voices[ch] = voice{}
		}
	}
}

// PlayEffect routes an original effect to one of the four hardware voices.
// A new effect on the same channel replaces the previous one.
func (s *Stream) PlayEffect(id string, channel int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if channel < 0 || channel >= 4 {
		return fmt.Errorf("invalid audio channel")
	}
	for i := range s.bank.Effects {
		if s.bank.Effects[i].ID == id {
			s.startEffect(&s.bank.Effects[i], channel)
			return nil
		}
	}
	return fmt.Errorf("unknown audio effect %q", id)
}

func (s *Stream) startEffect(sequence *Sequence, channel int) {
	// A replacement owns a new record and does not inherit its predecessor's stop.
	s.queuedStops &^= 1 << channel
	s.effects[channel] = playback{sequence: sequence, active: true}
	s.voices[channel] = voice{}
	for _, event := range sequence.StartEvents {
		s.apply(&s.voices[channel], event)
	}
}

// QueueEffect dispatches at the next fifty-Hz tick. Multiple requests for one
// voice replace each other before dispatch, matching the original sound queues.
func (s *Stream) QueueEffect(id string, channel int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if channel < 0 || channel >= 4 {
		return fmt.Errorf("invalid audio channel")
	}
	for i := range s.bank.Effects {
		if s.bank.Effects[i].ID == id {
			s.queued[channel] = &s.bank.Effects[i]
			return nil
		}
	}
	return fmt.Errorf("unknown audio effect %q", id)
}

// EffectActive reports the original effect ownership flag for one voice.
// Queued effects do not own their voice until the next audio tick.
func (s *Stream) EffectActive(channel int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return channel >= 0 && channel < 4 && s.effects[channel].active
}

// QueueStopEffects marks the four active records for termination on the next
// audio tick. Existing sound requests still dispatch afterward, as in the driver.
// A direct replacement admitted before that tick clears its own pending stop.
func (s *Stream) QueueStopEffects() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queuedStops = 0x0f
}

func (s *Stream) StopEffects() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopEffects(true)
}

func (s *Stream) stopEffects(clearQueued bool) {
	s.queuedStops = 0
	for ch := range s.effects {
		if clearQueued {
			s.queued[ch] = nil
		}
		owned := s.effects[ch].active
		s.effects[ch] = playback{}
		if owned {
			s.restoreMusic(ch)
		}
	}
}

func (s *Stream) restoreMusic(ch int) {
	s.voices[ch] = s.shadow[ch]
	// The original driver restores its cached reload buffer after an effect.
	s.voices[ch].current = s.shadow[ch].next
	// A cached silence reload remains valid even while the score voice rests.
	s.voices[ch].enabled = s.shadow[ch].enabled || s.shadow[ch].next != ""
	s.voices[ch].position = 0
	s.voices[ch].remaining = int64(s.voices[ch].period) * int64(s.sampleRate)
}

func (s *Stream) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if len(p) < 4 {
		return 0, io.ErrShortBuffer
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(p) / 4 * 4
	for off := 0; off < n; off += 4 {
		if s.remaining == 0 {
			s.tick()
			s.clockRemainder += s.sampleRate
			s.remaining = s.clockRemainder / TickRate
			s.clockRemainder %= TickRate
		}
		var left, right int32
		for ch := range s.voices {
			v := s.sample(&s.voices[ch])
			if ch == 0 || ch == 3 {
				left += v
			} else {
				right += v
			}
		}
		if s.filter != nil {
			left, right = s.filter.sample(0, left), s.filter.sample(1, right)
		}
		binary.LittleEndian.PutUint16(p[off:], uint16(int16(left)))
		binary.LittleEndian.PutUint16(p[off+2:], uint16(int16(right)))
		s.remaining--
	}
	return n, nil
}

func (s *Stream) sample(v *voice) int32 {
	if !v.enabled || v.period == 0 {
		return 0
	}
	data := s.waveforms[v.current]
	if len(data) == 0 {
		return 0
	}
	if v.remaining == 0 {
		v.remaining = int64(v.period) * int64(s.sampleRate)
	}
	value := int32(int8(data[v.position])) * int32(v.volume) * 2
	v.remaining -= int64(s.bank.Clock)
	for v.remaining <= 0 {
		v.position++
		if v.position == len(data) {
			v.position = 0
			v.current = v.next
			data = s.waveforms[v.current]
			if len(data) == 0 {
				v.enabled, v.remaining = false, 0
				break
			}
		}
		v.remaining += int64(v.period) * int64(s.sampleRate)
	}
	return value
}

func (s *Stream) tick() {
	for ch, sequence := range s.queued {
		if sequence != nil {
			s.startEffect(sequence, ch)
			s.queued[ch] = nil
		}
	}
	s.advance(&s.music, func(e Event) {
		s.apply(&s.shadow[e.Channel], e)
		if !s.effects[e.Channel].active {
			s.apply(&s.voices[e.Channel], e)
		}
	})
	if s.music.sequence != nil {
		s.musicPhase = s.music.phase
	}
	if s.queuedStops != 0 {
		for ch := range s.effects {
			if s.queuedStops&(1<<ch) == 0 {
				continue
			}
			owned := s.effects[ch].active
			s.effects[ch] = playback{}
			if owned {
				s.restoreMusic(ch)
			}
		}
		s.queuedStops = 0
	}
	for ch := range s.effects {
		if !s.effects[ch].active {
			continue
		}
		s.advance(&s.effects[ch], func(e Event) {
			s.apply(&s.voices[ch], e)
			if e.Kind == "stop" {
				s.effects[ch].active = false
				s.restoreMusic(ch)
			}
		})
		if !s.effects[ch].active {
			s.restoreMusic(ch)
		}
	}
}

func (s *Stream) advance(p *playback, consume func(Event)) {
	if !p.active {
		return
	}
	next := uint16(p.phase) + uint16(p.sequence.PhaseIncrement)
	p.phase = byte(next)
	if next > 255 {
		return
	}
	if p.age >= p.sequence.Ticks {
		if p.sequence.LoopTick < 0 {
			p.active = false
			return
		}
		p.age = p.sequence.LoopTick
		p.index = sort.Search(len(p.sequence.Events), func(i int) bool { return p.sequence.Events[i].Tick >= p.age })
	}
	for p.index < len(p.sequence.Events) && p.sequence.Events[p.index].Tick == p.age {
		consume(p.sequence.Events[p.index])
		p.index++
	}
	p.age++
}

func (s *Stream) apply(v *voice, e Event) {
	switch e.Kind {
	case "start":
		v.current = e.Sample
		v.position = 0
		v.remaining = int64(v.period) * int64(s.sampleRate)
		v.enabled = true
	case "loop":
		v.next = e.Sample
	case "stop":
		v.enabled = false
	case "period":
		v.period = e.Period
	case "volume":
		v.volume = e.Volume
	}
}
