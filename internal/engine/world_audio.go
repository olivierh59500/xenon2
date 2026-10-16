package engine

import "xenon2/internal/audio"

// AudioRequests preserves the relative order of direct admissions and the last
// effect-stop request. Queued effects dispatch at their next audio interrupt.
type AudioRequests = audio.Requests

func (w *World) requestImmediateSound(channel int, effect string) {
	if channel < 0 || channel >= len(w.ImmediateSoundRequests) {
		return
	}
	w.ImmediateSoundRequests[channel] = effect
	w.immediateSoundsBeforeStop &^= 1 << channel
}

func (w *World) requestEffectStop() {
	w.StopEffectsRequested = true
	w.immediateSoundsBeforeStop = 0
	for channel, effect := range w.ImmediateSoundRequests {
		if effect != "" {
			w.immediateSoundsBeforeStop |= 1 << channel
		}
	}
}

// TakeAudioRequests consumes a complete gameplay sound boundary once. The last
// direct admission on a voice replaces any earlier admission on that voice.
func (w *World) TakeAudioRequests() AudioRequests {
	requests := AudioRequests{StopEffects: w.StopEffectsRequested, Queued: w.SoundRequests}
	for channel, effect := range w.ImmediateSoundRequests {
		if requests.StopEffects && w.immediateSoundsBeforeStop&(1<<channel) != 0 {
			requests.BeforeStop[channel] = effect
		} else {
			requests.AfterStop[channel] = effect
		}
	}
	w.SoundRequests, w.ImmediateSoundRequests = [4]string{}, [4]string{}
	w.StopEffectsRequested, w.immediateSoundsBeforeStop = false, 0
	return requests
}
