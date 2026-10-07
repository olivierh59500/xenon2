// Command export-shop-audio recovers the original merchant effects and speech.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"xenon2/internal/assetimport"
	"xenon2/internal/audioexport"
)

func main() {
	input := flag.String("analysis", ".local/imported", "excluded decoded source directory")
	output := flag.String("output", "assets/runtime/shop-audio", "excluded exported audio directory")
	flag.Parse()
	if flag.NArg() != 0 {
		fail(fmt.Errorf("unexpected positional arguments"))
	}
	data, err := os.ReadFile(filepath.Join(*input, assetimport.Levels[5].Name+".decoded"))
	if err != nil {
		fail(err)
	}
	bank, samples, err := audioexport.DecodeShopAudio(data)
	if err != nil {
		fail(err)
	}
	if err = os.MkdirAll(*output, 0755); err != nil {
		fail(err)
	}
	for _, sample := range bank.Samples {
		if err = os.WriteFile(filepath.Join(*output, sample.File), samples[sample.ID], 0644); err != nil {
			fail(err)
		}
	}
	b, err := json.MarshalIndent(bank, "", "  ")
	if err != nil {
		fail(err)
	}
	if err = os.WriteFile(filepath.Join(*output, "bank.json"), append(b, '\n'), 0644); err != nil {
		fail(err)
	}
	fmt.Printf("Exported %d shop audio resources and %d original effects.\n", len(samples), len(bank.Effects))
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
