package shopui

import (
	"fmt"
	"sort"
	"strings"

	"xenon2/internal/engine"
	"xenon2/internal/visualassets"
)

type Phase string

const (
	PortraitEntrance Phase = "portrait-entrance"
	TelevisionsOn    Phase = "televisions-on"
	TelevisionsOff   Phase = "televisions-off"
	HeadphoneHand    Phase = "headphone-hand"
	Selling          Phase = "selling"
	Buying           Phase = "buying"
	PurchaseRequest  Phase = "purchase-request"
	Paying           Phase = "paying"
	PortraitExit     Phase = "portrait-exit"
	MerchantEnding   Phase = "merchant-ending"
	MerchantWelcome  Phase = "merchant-welcome"
	EndingFade       Phase = "ending-fade"
	EndingDot        Phase = "ending-dot"
	EndingDotFade    Phase = "ending-dot-fade"
	EndingWait       Phase = "ending-wait"
)

type Entry struct {
	Item            engine.Item
	Sale            engine.SalePosition
	Available, More bool
}

type Glyph struct {
	Character   rune
	X, Y        int
	Word        bool
	Number      bool
	NumberStart bool
}

// Cue preserves whether the original caller dispatches a voice directly or
// leaves a request for the next audio interrupt.
type Cue struct {
	ID        string
	Channel   int
	Immediate bool
}

// State preserves equipment and cash owned by the active game world.
type State struct {
	Ending                        bool
	EndingMessage                 int
	EndingHold                    int
	WelcomeMessage, WelcomeHold   int
	DisplayPhase                  Phase
	StopEffectsRequested          bool
	AdviceIndex                   *int
	localAdviceIndex              int
	Phase                         Phase
	PhasePass                     int
	IconPasses                    [20]int
	IntroHandFrame                int
	IntroHandRemaining            int
	afterTelevisions              Phase
	Entrance                      int
	Exiting                       bool
	Headphones, LowerOverlay      int
	HandFrame, HandRemaining      int
	BlinkFrames                   int
	LeftEyeClosed, RightEyeClosed bool
	AdviceFrames                  int
	ForceEyesOpen                 bool
	NoiseFrame                    int
	DisplayMoney                  int
	Television                    [20]int
	Noise                         [20][28]uint16
	equipment                     *engine.Equipment
	Money                         *int
	Rules                         engine.ShopRules
	catalogue                     *visualassets.ShopCatalogue
	scene                         *visualassets.ShopScene
	random                        func() uint16
	Selling                       bool
	Column, Row, Offset           int
	Entries                       [20]Entry
	Quoted                        int
	QuoteValid                    bool
	Dialogue                      []Glyph
	Revealed                      int
	Mouth                         int
	Frame                         int
	Done                          bool
	cues                          []Cue
	purchaseItem                  engine.Item
	paymentRemaining              int
	dialogueRunning               bool
	dialogueWords                 []int
	dialogueWord                  int
	dialogueColumn, dialogueRow   int
	numberWait                    bool
	numberTalking                 bool
	saleQuote                     bool
	saleHandAt, saleHandPass      int
	saleHandPending               bool
}

func New(equipment *engine.Equipment, money *int, rules engine.ShopRules, catalogue *visualassets.ShopCatalogue, scene *visualassets.ShopScene, random func() uint16) *State {
	s := &State{equipment: equipment, Money: money, DisplayMoney: *money, Rules: rules, catalogue: catalogue, scene: scene, random: random, Selling: true, Column: 1, Row: 4, Phase: PortraitEntrance, Quoted: -1, Headphones: 48, LowerOverlay: 68, Entrance: 17, ForceEyesOpen: true}
	s.rebuild()
	s.AdviceIndex = &s.localAdviceIndex
	return s
}

func (s *State) rebuild() {
	s.Entries = [20]Entry{}
	for i := range s.Television {
		s.Television[i] = -9
		s.IconPasses[i] = 0
	}
	if s.Selling {
		slots := []engine.WeaponSlot{s.equipment.Primary, s.equipment.Mounts[0], s.equipment.Mounts[1], s.equipment.Mounts[2], s.equipment.Mounts[3], s.equipment.Rear, s.equipment.Side}
		positions := make([]int, 0, 7)
		for i := range slots {
			if _, err := s.Rules.QuoteSale(*s.equipment, engine.SalePosition(i)); err == nil {
				positions = append(positions, i)
			}
		}
		sort.Slice(positions, func(i, j int) bool { return slots[positions[i]].Serial > slots[positions[j]].Serial })
		for i, position := range positions {
			s.Entries[i] = Entry{Item: slots[position].Item, Sale: engine.SalePosition(position), Available: true}
		}
		return
	}
	for i := 0; i < 19; i++ {
		item := engine.Item(s.Offset + 2 + i)
		if item > engine.ItemBitmapShades {
			break
		}
		price, _ := s.Rules.Price(item)
		s.Entries[i] = Entry{Item: item, Available: price <= *s.Money && price <= s.Rules.StockLimit}
	}
	if s.Offset != 0 {
		s.Entries[19] = Entry{More: true, Available: true}
	} else {
		price, _ := s.Rules.Price(engine.ItemBomb)
		s.Entries[19] = Entry{More: true, Available: price <= *s.Money && price <= s.Rules.StockLimit}
	}
}

// Move applies the original wrap-around five-column/five-row navigation.
// The final row has just the EXIT and confirmation buttons.
func (s *State) Move(dx, dy int) {
	if s.Phase != Selling && s.Phase != Buying || s.numberWait {
		return
	}
	if s.DisplayMoney != *s.Money || s.HandRemaining > 0 || s.Revealed < len(s.Dialogue) {
		return
	}
	if dx != 0 {
		if s.Row == 4 {
			if s.Column == 0 {
				s.Column = 1
			} else {
				s.Column = 0
			}
		} else {
			s.Column = (s.Column + dx + 5) % 5
		}
	} else if dy != 0 {
		s.Row = (s.Row + dy + 5) % 5
		if s.Row == 4 && s.Column > 1 {
			s.Column = 1
		}
	}
	s.cues = append(s.cues, Cue{ID: "shop-synthesized-effect-12", Channel: 2})
}

func (s *State) Confirm() error {
	if s.Phase != Selling && s.Phase != Buying || s.numberWait || s.saleQuote {
		return nil
	}
	if s.Revealed < len(s.Dialogue) {
		s.Revealed = len(s.Dialogue)
		s.Mouth = 0
		return nil
	}
	if s.HandRemaining > 0 {
		return nil
	}
	if s.DisplayMoney != *s.Money {
		return nil
	}
	s.cues = append(s.cues, Cue{ID: "shop-synthesized-effect-13", Channel: 2})
	if s.Row == 4 {
		if s.Column == 0 {
			if s.Selling {
				s.beginTelevisionsOff(Buying)
			} else {
				s.Rules.Leave(s.equipment)
				s.beginTelevisionsOff(PortraitExit)
			}
			return nil
		}
		return s.transact()
	}
	index := s.Row*5 + s.Column
	entry := s.Entries[index]
	if entry.More && entry.Available {
		s.Offset += 19
		if s.Offset >= 25 {
			s.Offset = 0
		}
		s.QuoteValid = false
		s.beginTelevisionsOff(Buying)
		return nil
	}
	if !entry.Available {
		if entry.Item != engine.ItemNone || entry.More {
			s.say(s.scene.Messages["out-of-stock"])
		}
		s.QuoteValid = false
		return nil
	}
	if s.QuoteValid && s.Quoted == index {
		return s.transact()
	}
	s.Quoted = index
	s.QuoteValid = true
	amount := 0
	var err error
	if s.Selling {
		amount, err = s.Rules.QuoteSale(*s.equipment, entry.Sale)
	} else {
		amount, err = s.Rules.Price(entry.Item)
	}
	if err != nil {
		s.QuoteValid = false
		return err
	}
	Item := s.catalogue.Items[int(entry.Item)-1]
	s.beginDialogue()
	if s.Selling {
		s.saleQuote = true
		s.BlinkFrames = -1
	}
	s.appendDialogue(strings.Join(Item.Lines, "\r"))
	if s.Selling {
		slots := []engine.WeaponSlot{s.equipment.Primary, s.equipment.Mounts[0], s.equipment.Mounts[1], s.equipment.Mounts[2], s.equipment.Mounts[3], s.equipment.Rear, s.equipment.Side}
		switch entry.Item {
		case engine.ItemForwardShot, engine.ItemDrone, engine.ItemLaser, engine.ItemRearShot, engine.ItemSideShot, engine.ItemDoubleShot, engine.ItemMineSmall, engine.ItemMineLarge:
			s.appendDialogue(s.scene.Messages["power-level"])
			s.appendNumber(slots[int(entry.Sale)].Tier + 1)
		}
	}
	if s.Selling {
		if len(s.scene.SaleHand) > 0 {
			s.saleHandAt = len(s.Dialogue)
		}
		s.appendDialogue(s.scene.Messages["sale-price"])
	} else {
		s.appendDialogue(s.scene.Messages["buy-price"])
	}
	s.appendNumber(amount)
	return nil
}

func (s *State) transact() error {
	if !s.QuoteValid || s.Quoted < 0 || s.Quoted >= 20 {
		return nil
	}
	entry := s.Entries[s.Quoted]
	if s.Selling {
		if _, err := s.Rules.Sell(s.equipment, s.Money, entry.Sale); err != nil {
			s.QuoteValid = false
			return err
		}
		s.DisplayMoney = *s.Money
		s.say(s.scene.Messages["sale-complete"])
	} else {
		if err := s.Rules.CanBuy(*s.equipment, *s.Money, entry.Item); err != nil {
			s.say(s.scene.Messages["cannot-fit"])
			s.QuoteValid = false
			return nil
		}
		s.purchaseItem = entry.Item
		s.paymentRemaining, _ = s.Rules.Price(entry.Item)
		s.Phase = PurchaseRequest
		s.say(s.scene.Messages["pay-request"])
		s.QuoteValid = false
		return nil
	}
	s.QuoteValid = false
	if s.Selling {
		s.Entries[s.Quoted] = Entry{}
	}
	return nil
}

func (s *State) finishPurchase() {
	item := s.purchaseItem
	s.equipment.ApplyItem(item)
	s.purchaseItem = engine.ItemNone
	s.Phase = Buying
	if item == engine.ItemAdvice && len(s.scene.AdviceTips) >= s.Rules.Level && s.AdviceIndex != nil {
		tips := s.scene.AdviceTips[s.Rules.Level-1]
		index := min(5, *s.AdviceIndex/4)
		s.say(tips[index])
		if *s.AdviceIndex != 8 && *s.AdviceIndex != 20 {
			*s.AdviceIndex += 4
		}
		s.AdviceFrames = 12
	} else {
		s.say(s.scene.Messages["another-item"])
	}
	for i, entry := range s.Entries {
		if entry.More {
			if s.Offset == 0 {
				price, _ := s.Rules.Price(engine.ItemBomb)
				s.Entries[i].Available = price <= *s.Money && price <= s.Rules.StockLimit
			}
			continue
		}
		if entry.Item != engine.ItemNone {
			price, _ := s.Rules.Price(entry.Item)
			s.Entries[i].Available = price <= *s.Money && price <= s.Rules.StockLimit
		}
	}
}

func (s *State) say(text string) {
	s.beginDialogue()
	s.appendDialogue(text)
}

func (s *State) beginDialogue() {
	s.dialogueRunning = true
	s.Dialogue = s.Dialogue[:0]
	s.dialogueWords = s.dialogueWords[:0]
	s.dialogueWord = 0
	s.Revealed = 0
	s.dialogueColumn, s.dialogueRow = 0, 0
	s.numberWait = false
	s.saleQuote = false
	s.saleHandAt, s.saleHandPass = -1, 0
	s.saleHandPending = false
}

func (s *State) appendNumber(value int) {
	// Returning from the first recursive digit clears the original talking
	// flag; subsequent digits still wait, but use the idle mouth callback.
	for index, char := range fmt.Sprint(value) {
		s.Dialogue = append(s.Dialogue, Glyph{Character: char, X: s.scene.DialogueX + s.dialogueColumn*s.scene.Font.Width, Y: s.scene.DialogueY + s.dialogueRow*s.scene.DialogueLineStep, Number: true, NumberStart: index == 0})
		s.dialogueColumn++
	}
}

func (s *State) appendDialogue(text string) {
	chars := []rune(text)
	column, row, position := s.dialogueColumn, s.dialogueRow, 0
	defer func() { s.dialogueColumn, s.dialogueRow = column, row }()
	for {
		// The original restarts the voice routine after every carriage return,
		// including empty lines before or after the visible words.
		s.dialogueWords = append(s.dialogueWords, len(s.Dialogue))
		length := 0
		for end := position; end < len(chars) && chars[end] != ' ' && chars[end] != '\r'; end++ {
			length++
		}
		if column+length > s.scene.DialogueColumns {
			column = 0
			row++
		}
		first := true
		for {
			if position == len(chars) {
				return
			}
			char := chars[position]
			position++
			if char == '\r' {
				column = 0
				row++
				break
			}
			if char == ' ' {
				column++
				for position < len(chars) && chars[position] == ' ' {
					position++
					column++
				}
				if position == len(chars) {
					return
				}
				break
			}
			s.Dialogue = append(s.Dialogue, Glyph{Character: char, X: s.scene.DialogueX + column*s.scene.Font.Width, Y: s.scene.DialogueY + row*s.scene.DialogueLineStep, Word: first})
			first = false
			column++
		}
	}
}

func (s *State) Advance() {
	s.Frame++
	s.DisplayPhase = s.Phase
	if s.Phase == PurchaseRequest && s.Revealed == len(s.Dialogue) {
		if s.dialogueRunning {
			// Empty trailing lines still enter the original speech routine
			// before the first payment wait.
			s.advanceDialogueToWait()
		}
		s.Phase = Paying
		s.dialogueRunning = false
	}
	if s.Phase == Paying {
		if s.paymentRemaining >= 100 {
			*s.Money -= 100
			s.paymentRemaining -= 100
			s.DisplayMoney = *s.Money
			s.cues = append(s.cues, Cue{ID: "shop-synthesized-effect-13", Channel: 2})
			// Payment waits update televisions before the idle eyes/mouth.
			s.advanceTelevisions()
			s.advancePortrait(false)
			return
		}
		*s.Money -= s.paymentRemaining
		s.paymentRemaining = 0
		s.DisplayMoney = *s.Money
		s.finishPurchase()
	}
	if s.advanceTransition() {
		return
	}
	if s.DisplayMoney > *s.Money {
		s.DisplayMoney = max(*s.Money, s.DisplayMoney-100)
	}
	if s.dialogueRunning || s.Revealed < len(s.Dialogue) {
		if !s.advanceDialogueToWait() {
			return
		}
		if !s.numberWait && !s.saleHandPending {
			s.advanceTelevisions()
			s.advancePortrait(true)
		}
		return
	}
	s.advanceTelevisions()
	s.advancePortrait(false)
}

// The source waits after an odd absolute character column. Spaces advance
// the cursor without drawing, so a display interval can contain more than
// two glyphs. Word samples are admitted before that word's first glyph.
func (s *State) advanceDialogueToWait() bool {
	s.dialogueRunning = true
	if s.numberWait {
		// Numeric output waits before these callbacks, whereas ordinary
		// speech performs them before its display wait.
		s.advanceTelevisions()
		s.advancePortrait(s.numberTalking)
		s.numberWait = false
	}
	if s.saleHandPending {
		s.advanceSaleHand()
		if s.HandRemaining > 0 {
			return true
		}
		s.saleHandPending = false
	}
	for {
		if s.saleHandAt >= 0 && s.saleHandAt == s.Revealed {
			s.saleHandAt = -1
			s.HandFrame = -1
			for _, frame := range s.scene.SaleHand {
				s.HandRemaining += frame.Duration
			}
			s.saleHandPending = s.HandRemaining > 0
			if s.saleHandPending {
				return true
			}
		}
		for s.dialogueWord < len(s.dialogueWords) && s.dialogueWords[s.dialogueWord] <= s.Revealed {
			if s.dialogueWords[s.dialogueWord] == s.Revealed {
				s.speakWord()
			}
			s.dialogueWord++
		}
		if s.Revealed == len(s.Dialogue) {
			s.dialogueRunning = false
			if s.saleQuote {
				s.saleQuote = false
				s.BlinkFrames = 0
			}
			if s.Phase == MerchantWelcome {
				if s.WelcomeMessage == 2 {
					s.Phase = Selling
				} else {
					s.WelcomeHold = 1
				}
			}
			return false
		}
		glyph := s.Dialogue[s.Revealed]
		if len(s.dialogueWords) == 0 && glyph.Word {
			s.speakWord()
		}
		s.Revealed++
		if glyph.Number {
			s.numberWait = true
			s.numberTalking = glyph.NumberStart
			return true
		}
		if (glyph.X/max(1, s.scene.Font.Width)+1)&1 != 0 {
			return true
		}
	}
}

func (s *State) advanceSaleHand() {
	pass := s.saleHandPass
	remaining := pass
	for index, frame := range s.scene.SaleHand {
		if remaining < frame.Duration {
			s.HandFrame = index
			break
		}
		remaining -= frame.Duration
	}
	// The approach/return loops draw the portrait before the monitors;
	// the nine three-pass hand holds call the shared monitor-first wait.
	if pass < 8 || pass >= 35 {
		s.advancePortrait(false)
		s.advanceTelevisions()
	} else {
		s.advanceTelevisions()
		s.advancePortrait(false)
	}
	s.saleHandPass++
	s.HandRemaining--
}

func (s *State) speakWord() {
	value := s.random() & 3
	for value == 0 {
		value = s.random() & 3
	}
	s.cues = append(s.cues, Cue{ID: fmt.Sprintf("shop-sampled-effect-%02d", value-1), Channel: 0, Immediate: true})
}

func (s *State) advancePortrait(talking bool) {
	s.Mouth = 0
	if talking {
		value := s.random()
		for s.Mouth < 3 && value&1 == 0 {
			s.Mouth++
			value >>= 1
		}
	}
	if s.ForceEyesOpen {
		s.LeftEyeClosed, s.RightEyeClosed = false, false
		if s.AdviceFrames > 0 {
			s.AdviceFrames--
		}
		return
	}
	if s.AdviceFrames > 0 {
		s.LeftEyeClosed = false
		s.AdviceFrames--
	} else if s.random()&0x2654 == 0 {
		s.LeftEyeClosed, s.RightEyeClosed = false, false
	}
	if s.BlinkFrames < 0 {
		s.LeftEyeClosed = true
		return
	}
	if s.BlinkFrames > 0 {
		s.BlinkFrames--
		s.LeftEyeClosed, s.RightEyeClosed = true, true
	} else if s.random()&0x2654 == 0 {
		s.BlinkFrames = 8
		s.LeftEyeClosed, s.RightEyeClosed = true, true
	}
}

func TelevisionPixels(words [28]uint16, palette [16][4]uint8) [32 * 28 * 4]byte {
	var pixels [32 * 28 * 4]byte
	for row, bits := range words {
		for column := 0; column < 32; column++ {
			index := 0
			mix := bits ^ (bits >> 8)
			if column < 16 {
				if mix&(1<<uint(15-column)) != 0 {
					index = 7
				}
			} else {
				mask := uint16(1 << uint(31-column))
				if mix&mask != 0 {
					index |= 1
				}
				if bits&mask != 0 {
					index |= 6
				}
			}
			c := palette[index]
			off := (row*32 + column) * 4
			copy(pixels[off:off+4], c[:])
		}
	}
	return pixels
}

func (s *State) TakeCues() []Cue { cues := s.cues; s.cues = nil; return cues }

func (s *State) beginTelevisionsOff(next Phase) {
	s.Phase, s.PhasePass, s.afterTelevisions = TelevisionsOff, 0, next
	for i := range s.Television {
		s.Television[i] = 0
	}
}

func (s *State) advanceTransition() bool {
	switch s.Phase {
	case PortraitEntrance:
		s.Headphones = max(0, s.Headphones-4)
		s.LowerOverlay = max(0, s.LowerOverlay-4)
		s.Entrance--
		if s.Headphones == 0 && s.Entrance == 5 {
			s.cues = append(s.cues, Cue{ID: "shop-sampled-effect-03", Channel: 0, Immediate: true})
		}
		if s.Entrance == 0 {
			s.Phase, s.PhasePass = TelevisionsOn, 0
			s.afterTelevisions = HeadphoneHand
		}
	case TelevisionsOn:
		s.PhasePass++
		for i := range s.Television {
			if s.PhasePass < 9 {
				s.Television[i] = s.PhasePass - 9
			} else {
				s.Television[i] = 1
			}
		}
		if s.PhasePass == 12 {
			s.Phase = s.afterTelevisions
			s.PhasePass = 0
			if s.Phase == HeadphoneHand && len(s.scene.IntroHand) > 0 {
				s.IntroHandRemaining = s.scene.IntroHand[0].Duration
			} else if s.Phase == HeadphoneHand {
				s.startSelling()
			} else {
				s.Phase = Buying
				s.Column, s.Row = 0, 0
				s.say(s.scene.Messages["buy-question"])
			}
		}
	case HeadphoneHand:
		pass := s.PhasePass
		if pass == 18 {
			s.StopEffectsRequested = true
			s.ForceEyesOpen = false
		}
		s.advanceTelevisions()
		s.advancePortrait(false)
		s.IntroHandFrame = 0
		for s.IntroHandFrame < len(s.scene.IntroHand) && pass >= s.scene.IntroHand[s.IntroHandFrame].Duration {
			pass -= s.scene.IntroHand[s.IntroHandFrame].Duration
			s.IntroHandFrame++
		}
		s.PhasePass++
		total := 0
		for _, frame := range s.scene.IntroHand {
			total += frame.Duration
		}
		if s.PhasePass == total {
			if s.Ending {
				s.Phase = MerchantEnding
				s.EndingMessage = 0
				s.say(s.scene.Messages["ending-viewers"])
				break
			}
			s.startSelling()
		}
	case MerchantWelcome:
		if s.dialogueRunning || s.Revealed < len(s.Dialogue) {
			return false
		}
		s.advanceTelevisions()
		s.advancePortrait(false)
		if s.WelcomeHold < 17 {
			s.WelcomeHold++
			return true
		}
		s.WelcomeMessage++
		s.WelcomeHold = 0
		s.say(s.scene.Messages[[]string{"last-level-welcome", "last-level-offer", "last-level-warning"}[s.WelcomeMessage]])
		return false
	case TelevisionsOff:
		s.PhasePass++
		for i := range s.Television {
			s.Television[i] = -min(9, s.PhasePass)
		}
		if s.PhasePass == 18 {
			if s.afterTelevisions == PortraitExit {
				s.Phase = PortraitExit
				s.Exiting = true
			} else {
				s.Selling = false
				s.QuoteValid = false
				s.Column, s.Row = 1, 4
				s.rebuild()
				s.Phase, s.PhasePass = TelevisionsOn, 0
			}
		}
	case PortraitExit:
		s.Headphones = min(48, s.Headphones+4)
		s.LowerOverlay = min(68, s.LowerOverlay+4)
		if s.LowerOverlay == 68 {
			if s.Ending {
				s.Phase = EndingFade
			} else {
				s.Done = true
			}
		}
	case MerchantEnding:
		if s.Revealed < len(s.Dialogue) {
			return false
		}
		s.EndingHold++
		hold := 30
		if s.EndingMessage == 2 {
			hold = 17
		}
		if s.EndingHold == hold {
			s.EndingMessage++
			s.EndingHold = 0
			if s.EndingMessage == 3 {
				s.beginTelevisionsOff(PortraitExit)
			} else {
				text := s.scene.Messages["ending-viewers"] + s.scene.Messages["ending-switch-off"]
				if s.EndingMessage == 1 {
					revealed := s.Revealed
					s.say(text)
					s.Revealed = revealed
				} else {
					s.say(s.scene.Messages["ending-question"])
				}
			}
		}
	case EndingFade, EndingDotFade:
		return true
	case EndingDot:
		s.PhasePass++
		if s.PhasePass == 150 {
			s.Phase = EndingDotFade
		}
	case EndingWait:
		s.PhasePass++
		if s.PhasePass == 80 {
			s.Done = true
		}
	default:
		return false
	}
	if s.DisplayPhase != HeadphoneHand {
		for i, counter := range s.Television {
			if counter <= 2 && counter != -8 {
				s.advanceNoise(i)
			}
		}
		s.advancePortrait(false)
	}
	return true
}

func (s *State) startSelling() {
	s.Column, s.Row = 0, 0
	if s.Rules.Level == 5 && s.scene.Messages["last-level-welcome"] != "" {
		s.Phase = MerchantWelcome
		s.WelcomeMessage, s.WelcomeHold = 0, 0
		s.say(s.scene.Messages["last-level-welcome"])
		return
	}
	s.Phase = Selling
	s.say(s.scene.Messages["sell-question"])
}

func (s *State) advanceNoise(index int) {
	value := s.random()
	for row := range s.Noise[index] {
		value = value*1021 + 41
		s.Noise[index][row] = value
	}
}

// Busy reports the staged passages during which the original ignores controls.
func (s *State) Busy() bool {
	return s.Phase != Selling && s.Phase != Buying || s.numberWait || s.saleHandPending || s.saleQuote
}

func (s *State) advanceTelevisions() {
	s.NoiseFrame = s.Frame
	for i, entry := range s.Entries {
		populated := entry.Item != engine.ItemNone || entry.More
		if !populated {
			s.Television[i] = 0
			continue
		}
		s.Television[i]--
		if s.Television[i] <= 0 {
			s.Television[i] = 20 + int(s.random()&255)
		}
	}
	// Counter reseeds precede every television draw in the original. Noise
	// consumes the same shared stream only after all counters were updated.
	for i := range s.Entries {
		if s.Television[i] > 2 {
			s.IconPasses[i]++
		}
		if s.Television[i] == 2 {
			primary := false
			for _, cue := range s.cues {
				primary = primary || cue.Channel == 2 && !cue.Immediate
			}
			if !primary {
				s.cues = append(s.cues, Cue{ID: "shop-synthesized-effect-11", Channel: 1})
			}
		}
		if s.Television[i] <= 2 && s.Television[i] != -8 {
			s.advanceNoise(i)
		}
	}
}

func (s *State) BeginEndingDot() {
	s.Phase, s.PhasePass = EndingDot, 0
	s.Dialogue = nil
	s.cues = append(s.cues, Cue{ID: "shop-synthesized-effect-18", Channel: 2})
}
func (s *State) BeginEndingWait() {
	s.Phase, s.PhasePass = EndingWait, 0
	s.StopEffectsRequested = true
}
