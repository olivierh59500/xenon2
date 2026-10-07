package engine

import (
	"fmt"

	"xenon2/internal/visualassets"
)

// FormationMotion reproduces the initialization of a particular wave member
// and body part. A part's score field also supplies its own path separation.
func FormationMotion(wave visualassets.Wave, member, partIndex int, part visualassets.ActorPart) (PathMotionConfig, error) {
	if member < 0 || member >= wave.Count || partIndex < 0 || wave.Spacing < 0 {
		return PathMotionConfig{}, fmt.Errorf("invalid formation member or spacing")
	}
	offset := member * (wave.Spacing % 100)
	config := PathMotionConfig{Budget: wave.MotionBudget}
	if wave.Spacing < 100 {
		config.Delay = offset
	} else {
		config.StartXOffset = offset
	}
	if partIndex != 0 {
		config.Delay += part.Score
	}
	if config.Delay > 32768 {
		return PathMotionConfig{}, fmt.Errorf("formation delay exceeds its motion range")
	}
	return config, nil
}

// FollowLeaderAnchor copies the integer anchor of a linked leader. The part's
// image anchor supplies its visible offset; this is not a separate trajectory.
func FollowLeaderAnchor(part *PathMotionState, leader PathMotionState) {
	part.X = leader.X&^0xffff | part.X&0xffff
	part.Y = leader.Y&^0xffff | part.Y&0xffff
}
