# Original soundtrack and sound effects

The Amiga floppy release embeds a David Whittaker score and replay driver.
This data is not a ProTracker module. The game also contains sampled effects
and speech, plus synthesized effects made from short waveform tables.
The Black Monks `Menue` program is a separate trainer with its own tune;
that tune is not part of the converted game's audio.

## Offline conversion

`cmd/export-audio` reads the checked, locally unpacked game data and writes:

- Twenty signed, eight-bit instrument samples and a silent reload buffer.
- Seven waveform fragments used by the synthesized effects.
- Eighteen sampled effects and speech resources.
- Resolved events for the main and menu music profiles.
- Resolved events for twenty-three synthesized effects and eighteen sampled effects.

The score compiler resolves order lists, instruments, note durations,
arpeggios, volume envelopes, pitch slides, vibrato, rests and ties. Its
output contains named samples and voice updates. It contains no executable
bytes, source addresses, machine registers or musical command bytecode.
Original resources and generated output remain outside Git.

```sh
GOWORK=off go run ./cmd/export-audio \
  -analysis .local/imported \
  -output assets/runtime/audio
```

The two music profiles preserve the original caller's distinct volume
settings. The score's fractional phase correction remains separate from
the resolved event loop; repeating the song does not restart that phase.
Sampled effect thirteen deliberately continues until replaced or stopped.
Several synthesized effects also sustain through an event loop.

## Go playback

`internal/audio.Stream` supplies stereo, signed 16-bit little-endian PCM.
It uses the PAL audio clock and advances voice events at fifty ticks per
second, independently of the graphics refresh rate. An initial sample and
its next DMA reload buffer are separate: selecting a loop buffer does not
cut off the sample currently playing.

```go
bank, samples, err := audio.LoadFS(os.DirFS(resourceDirectory), "audio")
if err != nil {
    return err
}
stream, err := audio.NewStream(bank, samples, 44100)
if err != nil {
    return err
}
if err := stream.PlayMusic("megablast-main"); err != nil {
    return err
}
```

The stream can be passed to Ebitengine's audio player. `PlayEffect` accepts
a named effect and one of four channels; an effect replaces the previous
effect on that channel. Music voice updates continue in the background,
and the cached musical reload buffer is restored when an effect ends.
`StopMusic` and `StopEffects` provide independent controls.

Channels zero and three feed the left output; channels one and two feed
the right output. Signed sample values and hardware volume masking are
preserved. The digital mixer uses sample-and-hold output; it does not yet
model the Amiga's analogue reconstruction filter or front-panel audio filter.
The PAL clock and volume masking also agree with the
[WinUAE Paula implementation](https://github.com/tonioni/WinUAE/blob/master/audio.cpp).

## Comparisons

Private reference tests compare the exported events with the original
driver's observed voice output. They cover eighteen thousand main-music
ticks on all four voices, including the repeat boundary, six thousand menu
music ticks, and twelve
thousand three hundred effect ticks across all forty-one effects.
These comparisons include selected waveform bytes, buffer lengths,
periods, effective volumes and DMA activation.

The original executable and reference traces remain local. They are not
needed by the player or normal tests. Ordinary tests verify signed stereo
output, independence from reader buffer size, resource validation and
concurrent playback controls.

```sh
GOWORK=off go test -race ./internal/audio ./internal/audioexport
XENON2_AUDIO_TEST_DIR="$PWD/.local" GOWORK=off \
  go test ./internal/audioexport -run '^TestPrivateOriginal' -count=1
```

The isolated driver comparisons establish audio-data conversion. Correct
timing and channel selection for every game event also depend on the
gameplay and interface integrations; those are separate comparisons.

The held flamer reads the effect ownership flag of voice one before
requesting another short loop. Releasing it marks all four effect records
for termination. `EffectActive` exposes ownership without counting queued
requests as active; `QueueStopEffects` applies the termination at the next
50 Hz audio tick. Requests already awaiting that tick still dispatch after
the stop, and terminated voices recover their music reload buffers. The
frontend delivers these status and stop boundaries before and after each
original logic pass.

The stream benchmark renders 1,024 stereo frames in approximately 23
microseconds on an Apple M4 Max, with no steady playback allocations.
This is a desktop measurement, not a Pixel performance result.

## Music continuity across player transitions

The source main score keeps running during ship loss, score initials, continue
and subsequent READY. Only the first READY starts the main score; merchant
returns and a newly loaded stage have their separate post-fade replay starts.
The frontend retains this admission state instead of deriving music solely
from the currently visible screen.

A PCM regression follows an actual collision loss through initials, accepted
continue and READY. 428,505 output frames from the unaffected left music voices
match an independent uninterrupted score stream. Effect-stop tests also verify
that terminating inactive effect records does not restart music DMA positions.
Termination queued for the 50 Hz tick preserves pending dispatch order and only
restores channels that an effect actually owns. These checks open no audio
device and do not establish the unmodelled Amiga analogue filter response.
