package assetimport

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrivateDiskImportOptional(t *testing.T) {
	path := os.Getenv("XENON2_PRIVATE_ADF")
	if path == "" {
		t.Skip("set XENON2_PRIVATE_ADF to the supported local disk")
	}
	root := t.TempDir()
	config := Config{ADF: path, Output: filepath.Join(root, "original"), Analysis: filepath.Join(root, "analysis"), DryRun: true}
	count, err := Import(config)
	if err != nil || count != 7 {
		t.Fatalf("dry import count%d err%v", count, err)
	}
	if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
		t.Fatalf("dry run wrote files: %v %v", entries, err)
	}
	config.DryRun = false
	if _, err := Import(config); err != nil {
		t.Fatal(err)
	}
	if err := Validate(config.Output, config.Analysis); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(config.Output, Executable.Name)); !os.IsNotExist(err) {
		t.Fatal("executable published as an ordinary asset")
	}
	path = filepath.Join(config.Output, Levels[0].Name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data[10] ^= 1
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := Validate(config.Output, config.Analysis); err == nil {
		t.Fatal("corrupt local asset accepted")
	}
}
