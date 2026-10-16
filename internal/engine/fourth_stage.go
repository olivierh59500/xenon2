package engine

// FourthStageDecision selects the source periodic falling-actor family. Its
// allocator and two random calls belong to the World construction phase.
type FourthStageDecision struct {
	MaximumScrollY int
	Spawn          bool
	FamilyOffset   int
}

func FourthStagePrelude(frame uint64, scrollY, maximum int) FourthStageDecision {
	decision := FourthStageDecision{MaximumScrollY: maximum}
	if maximum >= 3568 && scrollY <= 3568 {
		decision.MaximumScrollY = 3568
	}
	if frame&15 != 0 || scrollY > 3664 || scrollY < 2784 {
		return decision
	}
	decision.Spawn = true
	if frame&32 != 0 {
		decision.FamilyOffset = 2
	}
	return decision
}
