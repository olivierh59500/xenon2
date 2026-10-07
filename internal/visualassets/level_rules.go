package visualassets

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Checkpoint struct {
	TriggerY int `json:"trigger_y"`
	PlayerX  int `json:"player_x"`
	WorldY   int `json:"world_y"`
}

type LevelRules struct {
	Level                    int                `json:"level"`
	InitialMinimumScrollY    int                `json:"initial_minimum_scroll_y"`
	InitialMaximumScrollY    int                `json:"initial_maximum_scroll_y"`
	OrdinaryHealthMultiplier int                `json:"ordinary_health_multiplier"`
	StrongHealthMultiplier   int                `json:"strong_health_multiplier"`
	Checkpoints              []Checkpoint       `json:"checkpoints"`
	ScrollTransitions        []ScrollTransition `json:"scroll_transitions"`
	DefaultEnemyShot         string             `json:"default_enemy_shot"`
	EnemyShotChoices         []string           `json:"enemy_shot_choices"`
	EnemyShots               SpriteAtlas        `json:"enemy_shots"`
	TileShotSprites          [2]string          `json:"tile_shot_sprites,omitempty"`
	TileShotAnimations       [2]ActorAnimation  `json:"tile_shot_animations,omitempty"`
}

// ScrollTransition names a verified stage event and its resulting bounds. The
// independent Go stage controller determines when the event actually occurs.
type ScrollTransition struct {
	Event          string `json:"event"`
	Condition      string `json:"condition"`
	ThresholdY     int    `json:"threshold_y,omitempty"`
	RangeMaximumY  int    `json:"range_maximum_y,omitempty"`
	SetMinimum     *int   `json:"set_minimum,omitempty"`
	SetMaximum     *int   `json:"set_maximum,omitempty"`
	SetScroll      *int   `json:"set_scroll,omitempty"`
	MaximumAtLeast *int   `json:"maximum_at_least,omitempty"`
	ScoreBonus     int    `json:"score_bonus,omitempty"`
	Once           bool   `json:"once,omitempty"`
}

// DecodeLevelRules recognizes verified constant assignments in the offline
// initializer. The resource contains the resulting numbers, not instructions,
// register names, native branches or addresses.
func DecodeLevelRules(number int, level, common []byte, palette [16][4]uint8, encounters *Encounters) (*LevelRules, error) {
	ends := []int{0x567bc, 0x57252, 0x570c2, 0x57388, 0x57666}
	if number < 1 || number > len(ends) || len(level) < 0x24 || binary.BigEndian.Uint16(level[0x20:]) != 0x6000 {
		return nil, fmt.Errorf("unsupported level initializer")
	}
	start := 0x22 + int(int16(binary.BigEndian.Uint16(level[0x22:])))
	end := ends[number-1] - levelBase
	if start < 0 || end <= start || end > len(level) {
		return nil, fmt.Errorf("initializer leaves its source")
	}
	initializer := level[start:end]
	minimum, err := initializerWord(initializer, 0x0ce8)
	if err != nil {
		return nil, err
	}
	maximum, err := initializerWord(initializer, 0x0cea)
	if err != nil {
		return nil, err
	}
	ordinary, err := initializerHealth(initializer, 0x0424)
	if err != nil {
		return nil, err
	}
	strong, err := initializerHealth(initializer, 0x0426)
	if err != nil {
		return nil, err
	}
	rules := &LevelRules{Level: number, InitialMinimumScrollY: minimum, InitialMaximumScrollY: maximum, OrdinaryHealthMultiplier: ordinary, StrongHealthMultiplier: strong}
	rules.ScrollTransitions = knownScrollTransitions(number)
	for _, event := range encounters.Fixed {
		if event.EnemyKind == 0 {
			rules.Checkpoints = append(rules.Checkpoints, Checkpoint{TriggerY: event.TriggerY, PlayerX: event.X, WorldY: event.Y})
		}
	}
	images := make([]*Sprite, 0)
	names := map[int]string{}
	add := func(address int) (string, error) {
		if name, ok := names[address]; ok {
			return name, nil
		}
		name := fmt.Sprintf("enemy-shot-%d", len(images))
		data, position := level, address-levelBase
		if address < levelBase {
			data, position = common, address
		}
		picture, err := DecodeActorSprite(data, position, name, palette)
		if err != nil {
			return "", err
		}
		images = append(images, picture)
		names[address] = name
		return name, nil
	}
	defaultShot := int(binary.BigEndian.Uint32(level[0x28:]))
	rules.DefaultEnemyShot, err = add(defaultShot)
	if err != nil {
		return nil, err
	}
	pathTable, err := offset(level, 0x30, 4)
	if err != nil {
		return nil, err
	}
	table, choices := pathTable-32, 5
	if number == 5 {
		table, choices = pathTable-24, 4
	}
	if table < 0 {
		return nil, fmt.Errorf("enemy shot choice table is truncated")
	}
	for index := range choices {
		pointer := int(binary.BigEndian.Uint32(level[table+index*4:]))
		data, position := level, pointer-levelBase
		if pointer < levelBase {
			data, position = common, pointer
		}
		if position < 0 || position+4 > len(data) {
			return nil, fmt.Errorf("enemy shot image slot is outside source")
		}
		address := int(binary.BigEndian.Uint32(data[position:]))
		name, err := add(address)
		if err != nil {
			return nil, err
		}
		rules.EnemyShotChoices = append(rules.EnemyShotChoices, name)
	}
	if number == 3 {
		name, err := add(0x5e69e)
		if err != nil {
			return nil, err
		}
		rules.TileShotSprites = [2]string{name, name}
	}
	if number == 5 {
		for variant, root := range []int{0x57538, 0x57550} {
			clip, err := decodeActorAnimation(level, root-levelBase, add)
			if err != nil {
				return nil, err
			}
			rules.TileShotAnimations[variant] = clip
			rules.TileShotSprites[variant] = clip.Frames[0].Sprite
		}
	}
	rules.EnemyShots = packSprites(images)
	return rules, nil
}

func knownScrollTransitions(number int) []ScrollTransition {
	value := func(v int) *int { return &v }
	switch number {
	case 1:
		return []ScrollTransition{
			{Event: "middle-region-active", Condition: "scroll-y-in-range", ThresholdY: 2624, RangeMaximumY: 3344, MaximumAtLeast: value(3344)},
			{Event: "middle-gate-crossed", Condition: "player-world-y-below", ThresholdY: 2688, SetScroll: value(2496), SetMaximum: value(2496), Once: true},
			{Event: "final-guardian-active", Condition: "guardian-active", MaximumAtLeast: value(448)},
		}
	case 2:
		return []ScrollTransition{
			{Event: "middle-guardian-active", Condition: "guardian-active-and-scrolling-backward", MaximumAtLeast: value(2880)},
			{Event: "middle-guardian-defeated", Condition: "guardian-defeated", SetMinimum: value(0), Once: true},
		}
	case 3:
		return []ScrollTransition{
			{Event: "middle-boundary-reached", Condition: "scroll-y-equals-minimum", ThresholdY: 2800, SetMaximum: value(2800)},
			{Event: "middle-boundary-transition-finished", Condition: "transition-flag-consumed", SetMinimum: value(0)},
			{Event: "end-region-active", Condition: "scroll-y-at-most", ThresholdY: 208, MaximumAtLeast: value(32)},
		}
	case 4:
		return []ScrollTransition{
			{Event: "early-boundary-crossed", Condition: "maximum-at-least-threshold-and-scroll-at-most", ThresholdY: 3568, SetMaximum: value(3568)},
			{Event: "middle-guardian-defeated", Condition: "guardian-defeated", SetMinimum: value(0), SetScroll: value(2208), SetMaximum: value(2208), ScoreBonus: 2000, Once: true},
		}
	case 5:
		return []ScrollTransition{
			{Event: "final-guardian-started", Condition: "guardian-started", SetScroll: value(416), SetMaximum: value(416), Once: true},
		}
	default:
		return nil
	}
}

func initializerWord(initializer []byte, destination uint16) (int, error) {
	for offset := 0; offset+6 <= len(initializer); offset += 2 {
		if binary.BigEndian.Uint16(initializer[offset:]) == 0x31fc && binary.BigEndian.Uint16(initializer[offset+4:]) == destination {
			return int(int16(binary.BigEndian.Uint16(initializer[offset+2:]))), nil
		}
		if binary.BigEndian.Uint16(initializer[offset:]) == 0x4278 && binary.BigEndian.Uint16(initializer[offset+2:]) == destination {
			return 0, nil
		}
	}
	return 0, fmt.Errorf("initializer constant field is missing")
}

func initializerHealth(initializer []byte, destination uint16) (int, error) {
	prefix := []byte{0x30, 0x38, 0x0e, 0x02, 0xc0, 0xfc}
	for offset := 0; offset+12 <= len(initializer); offset += 2 {
		if bytes.Equal(initializer[offset:offset+6], prefix) && binary.BigEndian.Uint16(initializer[offset+8:]) == 0x31c0 && binary.BigEndian.Uint16(initializer[offset+10:]) == destination {
			return int(binary.BigEndian.Uint16(initializer[offset+6:])), nil
		}
	}
	return 0, fmt.Errorf("initializer health multiplier is missing")
}
