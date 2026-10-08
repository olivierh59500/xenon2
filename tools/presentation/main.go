// Command presentation prepares English captions without external font tools.
package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

//go:embed captions.json
var captions []byte

type cue struct {
	Start, End float64
	Lines      []string
}

func timestamp(seconds float64) string {
	milliseconds := int64(math.Round(seconds * 1000))
	return fmt.Sprintf("%02d:%02d:%02d,%03d", milliseconds/3600000, milliseconds/60000%60, milliseconds/1000%60, milliseconds%1000)
}

func run(output string) error {
	var cues []cue
	if err := json.Unmarshal(captions, &cues); err != nil {
		return err
	}
	parsed, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return err
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: 29, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return err
	}
	defer face.Close()
	if err := os.MkdirAll(output, 0755); err != nil {
		return err
	}
	var concat, subtitles strings.Builder
	concat.WriteString("ffconcat version 1.0\n")
	end := 0.0
	for index, cue := range cues {
		if cue.Start != end || cue.End <= cue.Start || cue.End > 300 || len(cue.Lines) < 1 || len(cue.Lines) > 2 {
			return fmt.Errorf("invalid continuous caption interval %d", index+1)
		}
		panel := image.NewRGBA(image.Rect(0, 0, 1280, 96))
		draw.Draw(panel, panel.Bounds(), &image.Uniform{C: color.RGBA{R: 11, G: 19, B: 29, A: 255}}, image.Point{}, draw.Src)
		for line, text := range cue.Lines {
			if font.MeasureString(face, text).Ceil() > 1200 {
				return fmt.Errorf("caption %d leaves the panel: %s", index+1, text)
			}
			tint := color.RGBA{R: 225, G: 233, B: 238, A: 255}
			if line == 0 {
				tint = color.RGBA{R: 255, G: 157, B: 73, A: 255}
			}
			drawer := font.Drawer{Dst: panel, Src: &image.Uniform{C: tint}, Face: face, Dot: fixed.P(40, 36+line*38)}
			drawer.DrawString(text)
		}
		name := fmt.Sprintf("caption-%03d.png", index)
		file, err := os.Create(filepath.Join(output, name))
		if err != nil {
			return err
		}
		encodeErr := png.Encode(file, panel)
		closeErr := file.Close()
		if encodeErr != nil {
			return encodeErr
		}
		if closeErr != nil {
			return closeErr
		}
		fmt.Fprintf(&concat, "file '%s'\nduration %.6f\n", name, cue.End-cue.Start)
		fmt.Fprintf(&subtitles, "%d\n%s --> %s\n%s\n\n", index+1, timestamp(cue.Start), timestamp(cue.End), strings.Join(cue.Lines, "\n"))
		end = cue.End
	}
	if end != 300 {
		return fmt.Errorf("caption track ends at %.3fs, expected five minutes", end)
	}
	// A final repeated image preserves the last concat duration through EOF.
	fmt.Fprintf(&concat, "file 'caption-%03d.png'\n", len(cues)-1)
	if err := os.WriteFile(filepath.Join(output, "captions.ffconcat"), []byte(concat.String()), 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(output, "presentation.en.srt"), []byte(subtitles.String()), 0644)
}

func main() {
	output := flag.String("output", "recordings/presentation-captions", "directory for generated PNG panels, concat list and English SRT")
	flag.Parse()
	if err := run(*output); err != nil {
		log.Fatal(err)
	}
}
