// Command export-audio resolves the local soundtrack into a code-free bank.
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
	output := flag.String("output", "assets/runtime/audio", "excluded exported audio directory")
	flag.Parse()
	if flag.NArg() != 0 {
		fail(fmt.Errorf("unexpected positional argument"))
	}
	data, err := os.ReadFile(filepath.Join(*input, assetimport.Executable.Name+".decoded"))
	if err != nil {
		fail(err)
	}
	bank, waveforms, err := audioexport.DecodeMusic(data)
	if err != nil {
		fail(err)
	}
	if err = os.MkdirAll(*output, 0755); err != nil {
		fail(err)
	}
	for _, sample := range bank.Samples {
		if err = os.WriteFile(filepath.Join(*output, sample.File), waveforms[sample.ID], 0644); err != nil {
			fail(err)
		}
	}
	encoded, err := json.MarshalIndent(bank, "", "  ")
	if err != nil {
		fail(err)
	}
	if err = os.WriteFile(filepath.Join(*output, "bank.json"), append(encoded, '\n'), 0644); err != nil {
		fail(err)
	}
	fmt.Printf("Exported %d samples, %d music sequences and %d effects.\n", len(bank.Samples), len(bank.Music), len(bank.Effects))
	for _, score := range bank.Music {
		fmt.Printf("%s: %d ticks, %d events, loop at %d.\n", score.ID, score.Ticks, len(score.Events), score.LoopTick)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
