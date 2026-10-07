package audioexport

import (
	"fmt"

	"xenon2/internal/audio"
)

// DecodeShopAudio exports the shop's separate effects-only Whittaker bank.
// The merchant's speech resources differ from the gameplay sample catalogue.
func DecodeShopAudio(data []byte) (*audio.Bank, map[string][]byte, error) {
	const base = 0x54e00
	if len(data) < 0x6defc-base {
		return nil, nil, fmt.Errorf("shop audio data is truncated")
	}
	c := &compiler{data: data, waveforms: make(map[string][]byte)}
	c.bank = audio.Bank{Version: audio.SchemaVersion, TickRate: audio.TickRate, Clock: audio.PALClock}
	noise := 0x650da - base
	if c.long(noise) != 64 || c.word(noise+4) != 8500 {
		return nil, nil, fmt.Errorf("unsupported shop waveform header")
	}
	waveStart := noise + 6 + 64 + 64
	if err := c.syntheticEffectsAt(syntheticLayout{records: 0x64e0e - base, envelopes: 0x6503a - base, relativeBase: 0x6476e - base, waveforms: waveStart, prefix: "shop-synthesized-effect", wavePrefix: "shop-waveform"}); err != nil {
		return nil, nil, err
	}
	c.addSample("shop-silence", make([]byte, 64))
	cursor := 0x6601c - base
	for i := 0; i < 4; i++ {
		n := int(c.long(cursor))
		hz := int(c.word(cursor + 4))
		if n < 2 || n%2 != 0 || cursor+6+n > len(data) || hz == 0 {
			return nil, nil, fmt.Errorf("invalid shop speech sample %d", i)
		}
		id := fmt.Sprintf("shop-sampled-effect-%02d", i)
		c.addSample(id, data[cursor+6:cursor+6+n])
		period := uint16(3579545 / hz)
		stop := int(uint8((n&65535)*50/hz+1)) - 1
		if stop < 0 {
			return nil, nil, fmt.Errorf("shop speech timer overflow")
		}
		events := []audio.Event{{Tick: 0, Channel: 0, Kind: "period", Period: period}, {Tick: 0, Channel: 0, Kind: "volume", Volume: 64}, {Tick: 0, Channel: 0, Kind: "start", Sample: id}, {Tick: 0, Channel: 0, Kind: "loop", Sample: "shop-silence"}, {Tick: stop, Channel: 0, Kind: "stop"}}
		c.bank.Effects = append(c.bank.Effects, audio.Sequence{ID: id, Ticks: stop + 1, LoopTick: -1, Events: events})
		cursor += 6 + n
	}
	if err := c.bank.Validate(c.waveforms); err != nil {
		return nil, nil, err
	}
	return &c.bank, c.waveforms, nil
}
