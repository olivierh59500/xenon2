package engine

// fourthMiddleNextInputViability owns one reusable native prediction workspace.
// It asks only whether one ordinary next command preserves the current shield.
// A safe command ends the fixed nine-command probe immediately.
type fourthMiddleNextInputViability struct{ forecast WorldForecast }
type fourthMiddleNextInputResult struct {
	safe    bool
	queries int
	motion  MotionInput
	shield  int
}

func (p *fourthMiddleNextInputViability) check(w *World, fire bool) (fourthMiddleNextInputResult, error) {
	result := fourthMiddleNextInputResult{shield: -1}
	for _, motion := range demoDirections {
		if err := p.forecast.Load(w); err != nil {
			return result, err
		}
		for range 3 {
			p.forecast.AdvancePALTick()
		}
		next, err := p.forecast.Advance(Input{Motion: motion, Fire: fire})
		result.queries++
		if err != nil {
			return result, err
		}
		if next.Shield > result.shield {
			result.shield = next.Shield
			result.motion = motion
		}
		if next.Alive && next.Lives == w.Equipment.Lives && next.Shield >= w.Equipment.Shield {
			result.safe, result.motion = true, motion
			return result, nil
		}
	}
	return result, nil
}
