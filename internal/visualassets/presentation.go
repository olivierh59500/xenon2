package visualassets

import (
	"encoding/binary"
	"fmt"
	"image"
)

type MenuLine struct {
	ID                 string `json:"id"`
	Text, SelectedText string
	CenterY            int
}

// Presentation retains the original fixed-width captions and their display font.
type Presentation struct {
	CreditOutSteps                                       []int
	CreditSecondSteps                                    []int
	TextZoom                                             SpriteAtlas `json:"text_zoom"`
	TextZoomRegions                                      [][]string  `json:"text_zoom_regions"`
	TextZoomHeights                                      []int       `json:"text_zoom_heights"`
	AppearSteps, DisappearSteps, MessageSteps            []int
	HighScoreHeading, ContinueHeading, ContinueCounter   string
	LogoZoom                                             SpriteAtlas `json:"logo_zoom"`
	LogoZoomX, LogoZoomY                                 []int
	StarColors                                           []uint8    `json:"star_colors"`
	CreditsCaption                                       string     `json:"credits_caption"`
	Font                                                 Font       `json:"font"`
	MenuHeading                                          string     `json:"menu_heading"`
	Menu                                                 []MenuLine `json:"menu"`
	MusicOn, MusicOnSelected, MusicOff, MusicOffSelected string
	Credits                                              []string `json:"credits"`
	Ready, GameOver                                      string
	HighScoreLines                                       []string `json:"high_score_lines"`
	TextZoomSteps                                        []int    `json:"text_zoom_steps"`
}

func DecodePresentation(common []byte, palette [16][4]uint8) (*Presentation, error) {
	const characters = 41
	if len(common) < 0x9a36+characters*176 {
		return nil, fmt.Errorf("presentation font truncated")
	}
	p := &Presentation{Font: Font{Characters: string(common[0x86d2 : 0x86d2+characters]), Width: 16, Height: 22, Columns: 10, Image: image.NewNRGBA(image.Rect(0, 0, 160, 110))}}
	if p.Font.Characters != "ABCDEFGHIJKLMNOPQRSTUVWXYZ.:><0123456789+" {
		return nil, fmt.Errorf("unsupported presentation font order")
	}
	for i := 0; i < characters; i++ {
		if err := drawPlanar(p.Font.Image, image.Pt(i%10*16, i/10*22), common[0x9a36+i*176:0x9a36+(i+1)*176], 16, 22, 4, false, palette); err != nil {
			return nil, err
		}
	}
	caption := func(start int) string { return string(common[start : start+20]) }
	p.MenuHeading = caption(0x8dd4)
	p.StarColors = append([]uint8(nil), common[0x8614:0x861c]...)
	p.CreditsCaption = string(common[0x8f76:0x8f7f])
	p.Menu = []MenuLine{{ID: "one-player", Text: caption(0x8de8), SelectedText: caption(0x8dfc), CenterY: 60}, {ID: "two-player", Text: caption(0x8e10), SelectedText: caption(0x8e24), CenterY: 84}, {ID: "music", CenterY: 108}}
	p.MusicOn, p.MusicOnSelected, p.MusicOff, p.MusicOffSelected = caption(0x8e38), caption(0x8e4c), caption(0x8e60), caption(0x8e74)
	for i := 0; i < 12; i++ {
		p.Credits = append(p.Credits, caption(0x8746+i*20))
	}
	p.Ready, p.GameOver = caption(0x8836), caption(0x884a)
	p.HighScoreHeading = caption(0x8c1c)
	p.ContinueHeading = caption(0x8f2e)
	p.ContinueCounter = caption(0x8f42)
	readSteps := func(start int) []int {
		var steps []int
		for at := start; at+2 <= len(common) && len(steps) < 1000; at += 2 {
			v := int(int16(binary.BigEndian.Uint16(common[at:])))
			steps = append(steps, v)
			if v < 0 {
				break
			}
		}
		return steps
	}
	p.AppearSteps = readSteps(0x8bd6)
	p.DisappearSteps = readSteps(0x8bf8)
	p.MessageSteps = readSteps(0x8364)
	p.CreditSecondSteps = append([]int(nil), p.MessageSteps...)
	p.CreditOutSteps = readSteps(0x8364 + len(p.MessageSteps)*2)
	for i := 0; i < 10; i++ {
		p.HighScoreLines = append(p.HighScoreLines, string(common[0x8946+i*15:0x8946+(i+1)*15]))
	}
	for at := 0x833a; at < 0x8364; at += 2 {
		value := int(int16(binary.BigEndian.Uint16(common[at:])))
		p.TextZoomSteps = append(p.TextZoomSteps, value)
		if value < 0 {
			break
		}
	}
	title, err := DecodeTitleArt(common)
	if err != nil {
		return nil, err
	}
	var frames []*Sprite
	for step := 1; step <= 16; step++ {
		mask := binary.BigEndian.Uint16(common[0x8724+step*2:])
		xs, ys := []int{}, []int{}
		for x := 0; x < 208; x++ {
			if mask&(1<<uint(15-x%16)) != 0 {
				xs = append(xs, x)
			}
		}
		for y := 0; y < 54; y++ {
			if mask&(1<<uint(15-y%16)) != 0 {
				ys = append(ys, y)
			}
		}
		picture := image.NewNRGBA(image.Rect(0, 0, len(xs), len(ys)))
		for y, sourceY := range ys {
			for x, sourceX := range xs {
				picture.SetNRGBA(x, y, title.Image.NRGBAAt(sourceX, sourceY))
			}
		}
		frames = append(frames, &Sprite{Name: fmt.Sprintf("title-zoom-%02d", step), Width: len(xs), Height: len(ys), Image: picture})
		p.LogoZoomX = append(p.LogoZoomX, 160+(-112*step>>4))
		p.LogoZoomY = append(p.LogoZoomY, 100+(-80*step>>4))
	}
	p.LogoZoom = packPresentationFrames(frames)
	var glyphFrames []*Sprite
	for step := 1; step <= 16; step++ {
		mask := binary.BigEndian.Uint16(common[0x8724+step*2:])
		xs, ys := []int{}, []int{}
		for x := 0; x < 16; x++ {
			if mask&(1<<uint(15-x)) != 0 {
				xs = append(xs, x)
			}
		}
		for y := 0; y < 22; y++ {
			if mask&(1<<uint(15-y%16)) != 0 {
				ys = append(ys, y)
			}
		}
		p.TextZoomHeights = append(p.TextZoomHeights, len(ys))
		var names []string
		for glyph := 0; glyph < characters; glyph++ {
			picture := image.NewNRGBA(image.Rect(0, 0, len(xs), len(ys)))
			sourceX, sourceY := glyph%10*16, glyph/10*22
			for y, oldY := range ys {
				for x, oldX := range xs {
					picture.SetNRGBA(x, y, p.Font.Image.NRGBAAt(sourceX+oldX, sourceY+oldY))
				}
			}
			name := fmt.Sprintf("caption-%02d-%02d", step, glyph)
			names = append(names, name)
			glyphFrames = append(glyphFrames, &Sprite{Name: name, Width: len(xs), Height: len(ys), Image: picture})
		}
		p.TextZoomRegions = append(p.TextZoomRegions, names)
	}
	p.TextZoom = packSprites(glyphFrames)
	return p, nil
}

func packPresentationFrames(frames []*Sprite) SpriteAtlas {
	width, height := 1, 1
	for _, frame := range frames {
		width = max(width, frame.Width)
		height = max(height, frame.Height)
	}
	atlas := SpriteAtlas{Image: image.NewNRGBA(image.Rect(0, 0, width, len(frames)*height))}
	for index, frame := range frames {
		y := index * height
		for row := 0; row < frame.Height; row++ {
			copy(atlas.Image.Pix[(y+row)*atlas.Image.Stride:], frame.Image.Pix[row*frame.Image.Stride:row*frame.Image.Stride+frame.Width*4])
		}
		atlas.Sprites = append(atlas.Sprites, SpriteRegion{Name: frame.Name, X: 0, Y: y, Width: frame.Width, Height: frame.Height})
	}
	return atlas
}
