package engine

// WaveBonusEntry counts independent wave members, excluding their body parts.
type WaveBonusEntry struct {
	ID        uint16
	Remaining uint16
}

// WaveBonusCache preserves the original eight shared encounter buckets. Waves
// created within one four-pass period can share a normal or heavy token.
type WaveBonusCache struct {
	NextID, NormalID, HeavyID uint16
	Entries                   [8]WaveBonusEntry
}

func NewWaveBonusCache() WaveBonusCache {
	return WaveBonusCache{NextID: 1}
}

// BeginPass refreshes the pair of tokens every fourth main simulation pass.
func (s *WaveBonusCache) BeginPass(frame uint64) {
	if frame&3 != 0 {
		return
	}
	s.NormalID, s.HeavyID = s.NextID, s.NextID+1
	s.NextID += 2
}

// Register assigns a wave's original shared token. A full table leaves the
// token untracked; that wave can still exist but cannot produce a bonus drop.
func (s *WaveBonusCache) Register(heavy bool, count int) uint16 {
	token := s.NormalID
	if heavy {
		token = s.HeavyID
	}
	if token == 0 || count <= 0 {
		return token
	}
	for i := range s.Entries {
		if s.Entries[i].ID == token {
			s.Entries[i].Remaining += uint16(count)
			return token
		}
	}
	for i := range s.Entries {
		if s.Entries[i].ID == 0 {
			s.Entries[i] = WaveBonusEntry{ID: token, Remaining: uint16(count)}
			s.NextID++
			break
		}
	}
	return token
}

// Defeat returns true only when the final independent member of a tracked
// bucket is destroyed. The caller chooses a normal or heavy cash reward.
func (s *WaveBonusCache) Defeat(token uint16) bool {
	if token == 0 {
		return false
	}
	for i := range s.Entries {
		if s.Entries[i].ID != token {
			continue
		}
		s.Entries[i].Remaining--
		if s.Entries[i].Remaining == 0 {
			s.Entries[i].ID = 0
			return true
		}
		return false
	}
	return false
}

// Escape invalidates the entire shared token, including other waves assigned
// within its four-pass period. Escaped waves do not earn the final cash drop.
func (s *WaveBonusCache) Escape(token uint16) {
	if token == 0 {
		return
	}
	for i := range s.Entries {
		if s.Entries[i].ID == token {
			s.Entries[i] = WaveBonusEntry{}
			return
		}
	}
}
