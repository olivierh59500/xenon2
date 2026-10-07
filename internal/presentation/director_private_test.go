package presentation

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"xenon2/internal/visualassets"
)

func TestPrivateOriginalPresentationPasses(t *testing.T) {
	dir := os.Getenv("XENON2_SHOP_TEST_DIR")
	if dir == "" {
		t.Skip("local original presentation references not supplied")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "XenonII.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	title, err := visualassets.DecodeTitleArt(raw)
	if err != nil {
		t.Fatal(err)
	}
	data, err := visualassets.DecodePresentation(raw, title.Palette)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		confirm int
		begin   func(*Director)
	}{
		{name: "attract", confirm: -1},
		{name: "ready", confirm: 100, begin: func(d *Director) { d.BeginReady(1) }},
		{name: "continue", confirm: -1, begin: func(d *Director) { d.BeginContinue() }},
		{name: "game-over", confirm: -1, begin: func(d *Director) { d.BeginGameOver() }},
		{name: "menu-in", confirm: -1, begin: func(d *Director) { d.BeginMenu() }},
		{name: "menu-out", confirm: -1, begin: func(d *Director) { d.BeginStart() }},
		{name: "initials", confirm: -1, begin: func(d *Director) { d.InsertScore(500) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			f, err := os.Open(filepath.Join(filepath.Dir(dir), "analysis", test.name+"-trace.csv"))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			rows, err := csv.NewReader(f).ReadAll()
			if err != nil {
				t.Fatal(err)
			}
			d := NewDirector(data)
			if test.begin != nil {
				test.begin(d)
			}
			for frame, row := range rows[1:] {
				confirm := frame == test.confirm
				if test.name == "initials" {
					confirm = frame == 40 || frame == 50 || frame == 60
				}
				d.Advance(Input{Confirm: confirm})
				number := func(at int) int {
					v, err := strconv.Atoi(row[at])
					if err != nil {
						t.Fatal(err)
					}
					return v
				}
				if d.LogoScale != number(1) || strconv.FormatBool(d.ShowScores) != row[2] {
					t.Fatalf("pass %d logo/scores Go %d/%v original %s/%s phase %s", frame, d.LogoScale, d.ShowScores, row[1], row[2], d.Phase)
				}
				var actual []Caption
				for _, caption := range d.Captions {
					if caption.Scale > 0 {
						actual = append(actual, caption)
					}
				}
				for i := 0; i < 2; i++ {
					at := 3 + i*3
					if row[at] == "" {
						if len(actual) > i {
							t.Fatalf("pass %d unexpected caption %+v", frame, actual[i])
						}
						continue
					}
					if len(actual) <= i {
						t.Fatalf("pass %d missing caption %q phase %s", frame, row[at], d.Phase)
					}
					got := actual[i]
					if got.Text != row[at] || got.Scale != number(at+1) || got.CenterY != number(at+2) {
						t.Fatalf("pass %d caption %d Go %+v original %q/%s/%s phase %s", frame, i, got, row[at], row[at+1], row[at+2], d.Phase)
					}
				}
			}
			t.Logf("Compared %d original presentation passes.", len(rows)-1)
		})
	}
}
