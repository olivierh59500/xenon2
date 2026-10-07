package audio

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
)

// LoadFS accepts only exported event metadata and separate PCM waveforms.
func LoadFS(resources fs.FS, directory string) (*Bank, map[string][]byte, error) {
	data, err := fs.ReadFile(resources, path.Join(directory, "bank.json"))
	if err != nil {
		return nil, nil, err
	}
	var bank Bank
	if err = json.Unmarshal(data, &bank); err != nil {
		return nil, nil, err
	}
	waveforms := make(map[string][]byte, len(bank.Samples))
	for _, sample := range bank.Samples {
		if !fs.ValidPath(sample.File) || path.Base(sample.File) != sample.File || path.Ext(sample.File) != ".pcm" {
			return nil, nil, fmt.Errorf("invalid sample filename")
		}
		pcm, err := fs.ReadFile(resources, path.Join(directory, sample.File))
		if err != nil {
			return nil, nil, err
		}
		waveforms[sample.ID] = pcm
	}
	if err = bank.Validate(waveforms); err != nil {
		return nil, nil, err
	}
	return &bank, waveforms, nil
}
