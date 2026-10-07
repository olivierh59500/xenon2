// Command disk-inspect inventories an AmigaDOS disk without executing its code.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"xenon2/internal/amigadisk"
)

type entry struct {
	amigadisk.Entry
	SHA256 string `json:"sha256,omitempty"`
	Prefix string `json:"prefix,omitempty"`
}

func main() {
	adf := flag.String("adf", "", "user-supplied AmigaDOS disk image")
	extract := flag.String("extract", "", "optional local directory for all disk files")
	unpack := flag.Bool("unpack", false, "decode the supported executable packer into the extraction directory")
	flag.Parse()
	if *adf == "" || flag.NArg() != 0 || *unpack && *extract == "" {
		fail(fmt.Errorf("use -adf disk.adf; -unpack requires -extract"))
	}
	data, err := os.ReadFile(*adf)
	if err != nil {
		fail(err)
	}
	disk, err := amigadisk.ParseDisk(data)
	if err != nil {
		fail(err)
	}
	entries := make([]entry, 0)
	for _, file := range disk.Entries() {
		item := entry{Entry: file}
		if !file.Directory {
			content, err := disk.ReadFile(file.Path)
			if err != nil {
				fail(err)
			}
			item.SHA256 = fmt.Sprintf("%x", sha256.Sum256(content))
			item.Prefix = hex.EncodeToString(content[:min(24, len(content))])
			if *extract != "" {
				if err := write(filepath.Join(*extract, filepath.FromSlash(file.Path)), content); err != nil {
					fail(err)
				}
				if *unpack && file.Path == "XenonII" {
					if len(content) < 0x118 {
						fail(fmt.Errorf("XenonII executable is truncated"))
					}
					decoded, err := amigadisk.UnpackByteKiller(content[0x10c:])
					if err != nil {
						fail(err)
					}
					if err := write(filepath.Join(*extract, "XenonII-unpacked.bin"), decoded); err != nil {
						fail(err)
					}
				}
			}
		}
		entries = append(entries, item)
	}
	result := struct {
		Name    string  `json:"name"`
		Format  string  `json:"format"`
		Bytes   int     `json:"bytes"`
		SHA256  string  `json:"sha256"`
		Entries []entry `json:"entries"`
	}{disk.Name, disk.Format, len(data), fmt.Sprintf("%x", sha256.Sum256(data)), entries}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fail(err)
	}
}

func write(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
