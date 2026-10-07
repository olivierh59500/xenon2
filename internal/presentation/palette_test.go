package presentation

import "testing"

func TestPaletteFadeKeepsPALWaitsAndThreeBitSteps(t *testing.T) {
	fade := NewPaletteFadeIn()
	for tick := 1; tick <= 14; tick++ {
		written := fade.AdvancePAL()
		if written != (tick%2 == 0) || fade.Deduction != 7-tick/2 {
			t.Fatalf("PAL tick %d fade %+v", tick, fade)
		}
	}
	if !fade.Done || fade.AdvancePAL() {
		t.Fatal("fade-in did not stop after seven writes")
	}
	fade = NewPaletteFadeOut(2)
	for tick := 1; tick <= 24; tick++ {
		written := fade.AdvancePAL()
		if written != (tick%3 == 0) || fade.Deduction != tick/3 {
			t.Fatalf("PAL tick %d fade %+v", tick, fade)
		}
	}
	if !fade.Done {
		t.Fatal("fade-out did not retain its eighth black step")
	}
	var palette [16][4]uint8
	palette[3] = [4]uint8{238, 68, 34, 255}
	result := FadePalette(palette, 2)
	if result[3] != [4]uint8{170, 0, 0, 255} || palette[3][0] != 238 {
		t.Fatal("component saturation, alpha or source palette changed")
	}
}
