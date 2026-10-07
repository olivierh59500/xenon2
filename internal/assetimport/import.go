// Package assetimport restores offline analysis inputs from a user-owned ADF.
// The production game consumes exported images and semantic data, never these
// packed programs or a 68000 interpreter.
package assetimport

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"xenon2/internal/amigadisk"
)

// Resource records the verified version used to derive the asset exporters.
type Resource struct {
	Name          string `json:"name"`
	Bytes         int    `json:"bytes"`
	SHA256        string `json:"sha256"`
	DecodedBytes  int    `json:"decoded_bytes"`
	DecodedSHA256 string `json:"decoded_sha256"`
}

// Levels retain disk order, including the final packed presentation payload.
// Each decoded payload contains code as well as data and belongs to analysis.
var Levels = []Resource{
	{"000B00E5", 72472, "827ada5a91d93684feaef4539d4f1031bf919fe962be7948898c40e22d311496", 117248, "31a1934992a983098f3cb660976c87cf380252c271001420a92327ff124e0ad7"},
	{"00FA00FE", 80456, "376f9fbccb738005e87a2cdc909cccab69d9d29c668214c8ec29cb0f7d80d3d3", 130048, "ec53f9fcd6d550534e76a3a12931e2d34b5f6f863e939ed780d7a99528522049"},
	{"02020113", 87312, "2234f300d9554e0da7f1fdd053a754220642b7ffefd4553881f381027c27b062", 140800, "dedca47a3461515bae895bc5790d3289cd325a2c9b71153e93eac8c520a85446"},
	{"031F0159", 140230, "cb47ed783f7961598f0e17a1fc92769c1d4d975dadb706a4ecdd64f3c0ff64c2", 176640, "b9b693499e6f4baa099241097a02003e6440cdf23b936bfca358d59193291c79"},
	{"04820138", 117050, "b4018aa3744f40f2d880e93be46709cc3ec963fb4d7b74bab6a03e760eb2a483", 159744, "294adc1c24016d73bd98924e65a5cab45c6d77930f2a028726a71e5ef90ea70b"},
	{"05c400f8", 60128, "7d10333fdc093cbe3ff057b49c3cff2b46758131f0f37ea540a880168258046e", 126976, "a1ee6ab8538b2e0588f99f11135c7047a922372e8e8651ea8696603d50807439"},
}

// Executable is decoded only into the separately excluded analysis directory.
var Executable = Resource{"XenonII", 200128, "1e00a5edb56450d63ff1da5b4950ccdcce9d1e3cbef472ab81131f0eb349ce89", 262440, "2b194ddcae5517a57956659bafbfbc4a03e7899904235738a274bbfec9587d74"}

type Config struct {
	ADF      string
	Output   string
	Analysis string
	DryRun   bool
}

type imported struct {
	Resource
	packed, decoded []byte
}

// Import verifies every required input before publishing any local file.
// This deliberately rejects different releases rather than guessing offsets.
func Import(config Config) (int, error) {
	if config.ADF == "" || config.Output == "" || config.Analysis == "" {
		return 0, fmt.Errorf("ADF, asset output and analysis output are required")
	}
	data, err := os.ReadFile(config.ADF)
	if err != nil {
		return 0, err
	}
	disk, err := amigadisk.ParseDisk(data)
	if err != nil {
		return 0, err
	}
	resources := append(append([]Resource(nil), Levels...), Executable)
	files := make([]imported, 0, len(resources))
	for _, resource := range resources {
		packed, err := disk.ReadFile(resource.Name)
		if err != nil {
			return 0, err
		}
		if err := verify(resource.Name, packed, resource.Bytes, resource.SHA256); err != nil {
			return 0, err
		}
		var decoded []byte
		if resource.Name == Executable.Name {
			decoded, err = amigadisk.UnpackByteKiller(packed[0x10c:])
		} else {
			decoded, err = amigadisk.UnpackLevel(packed)
		}
		if err != nil {
			return 0, fmt.Errorf("decode %s: %w", resource.Name, err)
		}
		if err := verify(resource.Name+" decoded", decoded, resource.DecodedBytes, resource.DecodedSHA256); err != nil {
			return 0, err
		}
		files = append(files, imported{resource, packed, decoded})
	}
	if config.DryRun {
		return len(files), nil
	}
	for _, file := range files {
		if file.Name != Executable.Name {
			if err := writeAtomic(filepath.Join(config.Output, file.Name), file.packed); err != nil {
				return 0, err
			}
		}
		if err := writeAtomic(filepath.Join(config.Analysis, file.Name+".decoded"), file.decoded); err != nil {
			return 0, err
		}
	}
	return len(files), nil
}

// Validate checks the locally restored packed data without reading an ADF.
func Validate(output, analysis string) error {
	for _, resource := range Levels {
		data, err := os.ReadFile(filepath.Join(output, resource.Name))
		if err != nil {
			return err
		}
		if err := verify(resource.Name, data, resource.Bytes, resource.SHA256); err != nil {
			return err
		}
	}
	for _, resource := range append(append([]Resource(nil), Levels...), Executable) {
		data, err := os.ReadFile(filepath.Join(analysis, resource.Name+".decoded"))
		if err != nil {
			return err
		}
		if err := verify(resource.Name+" decoded", data, resource.DecodedBytes, resource.DecodedSHA256); err != nil {
			return err
		}
	}
	return nil
}

func verify(name string, data []byte, size int, hash string) error {
	actual := fmt.Sprintf("%x", sha256.Sum256(data))
	if len(data) != size || actual != hash {
		return fmt.Errorf("unsupported %s: got %d bytes with SHA256 %s; expected %d bytes with SHA256 %s", name, len(data), actual, size, hash)
	}
	return nil
}

func writeAtomic(path string, data []byte) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}
	file, err := os.CreateTemp(directory, ".import-*")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Chmod(0644); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
