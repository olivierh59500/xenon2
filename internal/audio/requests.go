package audio

// Requests preserves direct admissions around the last gameplay effect stop.
// Queued effects dispatch at their next audio interrupt.
type Requests struct {
	BeforeStop, AfterStop, Queued [4]string
	StopEffects                   bool
}

// Append folds a later boundary into the same fixed voice slots. The last
// direct admission remains before or after the last stop in the combined order.
func (r Requests) Append(next Requests) Requests {
	if next.StopEffects {
		for channel, id := range r.AfterStop {
			if id != "" {
				r.BeforeStop[channel] = id
			}
			r.AfterStop[channel] = ""
		}
		r.StopEffects = true
	}
	for channel, id := range next.BeforeStop {
		if id != "" {
			if next.StopEffects {
				r.BeforeStop[channel], r.AfterStop[channel] = id, ""
			} else {
				r.BeforeStop[channel], r.AfterStop[channel] = "", id
			}
		}
	}
	for channel, id := range next.AfterStop {
		if id != "" {
			r.BeforeStop[channel], r.AfterStop[channel] = "", id
		}
	}
	for channel, id := range next.Queued {
		if id != "" {
			r.Queued[channel] = id
		}
	}
	return r
}

// ApplyRequests submits one gameplay boundary in its original call order.
func (s *Stream) ApplyRequests(requests Requests) error {
	for channel, id := range requests.BeforeStop {
		if id != "" {
			if err := s.PlayEffect(id, channel); err != nil {
				return err
			}
		}
	}
	if requests.StopEffects {
		s.QueueStopEffects()
	}
	for channel, id := range requests.AfterStop {
		if id != "" {
			if err := s.PlayEffect(id, channel); err != nil {
				return err
			}
		}
	}
	for channel, id := range requests.Queued {
		if id != "" {
			if err := s.QueueEffect(id, channel); err != nil {
				return err
			}
		}
	}
	return nil
}
