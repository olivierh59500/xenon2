package amigadisk

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestByteKillerLiteralAndIntegrity(t *testing.T) {
	// A three-byte literal: tag00, count010, and reverse-order C/B/A bytes.
	bits := []uint32{0, 0, 0, 1, 0}
	for _, value := range []byte{'C', 'B', 'A'} {
		for shift := 7; shift >= 0; shift-- {
			bits = append(bits, uint32(value>>shift&1))
		}
	}
	word := uint32(1) << len(bits)
	for i, bit := range bits {
		word |= bit << i
	}
	packed := make([]byte, 16)
	binary.BigEndian.PutUint32(packed, 4)
	binary.BigEndian.PutUint32(packed[4:], 3)
	binary.BigEndian.PutUint32(packed[8:], word)
	binary.BigEndian.PutUint32(packed[12:], word)
	decoded, err := UnpackByteKiller(packed)
	if err != nil || string(decoded) != "ABC" {
		t.Fatalf("decoded=%q err=%v", decoded, err)
	}
	packed[8] ^= 1
	if _, err := UnpackByteKiller(packed); err == nil {
		t.Fatal("corrupt checksum accepted")
	}
	for _, size := range []int{0, 12, 15} {
		if _, err := UnpackByteKiller(packed[:size]); err == nil {
			t.Fatalf("truncated input %d accepted", size)
		}
	}
}

// The optional original inputs stay outside Git. Synthetic filesystem tests
// remain sufficient to run the public package tests without copyrighted data.
func TestPrivatePackedAssetsOptional(t *testing.T) {
	directory := os.Getenv("XENON2_PRIVATE_DISK_FILES")
	if directory == "" {
		t.Skip("set XENON2_PRIVATE_DISK_FILES to locally extracted files")
	}
	files := []struct {
		name, sha string
		size      int
	}{
		{"000B00E5", "31a1934992a983098f3cb660976c87cf380252c271001420a92327ff124e0ad7", 117248},
		{"00FA00FE", "ec53f9fcd6d550534e76a3a12931e2d34b5f6f863e939ed780d7a99528522049", 130048},
		{"02020113", "dedca47a3461515bae895bc5790d3289cd325a2c9b71153e93eac8c520a85446", 140800},
		{"031F0159", "b9b693499e6f4baa099241097a02003e6440cdf23b936bfca358d59193291c79", 176640},
		{"04820138", "294adc1c24016d73bd98924e65a5cab45c6d77930f2a028726a71e5ef90ea70b", 159744},
		{"05c400f8", "a1ee6ab8538b2e0588f99f11135c7047a922372e8e8651ea8696603d50807439", 126976},
	}
	for _, file := range files {
		t.Run(file.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(directory, file.name))
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := UnpackLevel(data)
			if err != nil {
				t.Fatal(err)
			}
			if len(decoded) != file.size || fmt.Sprintf("%x", sha256.Sum256(decoded)) != file.sha {
				t.Fatalf("decoded content differs: size%d sha%x", len(decoded), sha256.Sum256(decoded))
			}
			data = data[:len(data)-1]
			if _, err := UnpackLevel(data); err == nil {
				t.Fatal("truncated original accepted")
			}
		})
	}
	data, err := os.ReadFile(filepath.Join(directory, "XenonII"))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := UnpackByteKiller(data[0x10c:])
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 262440 || fmt.Sprintf("%x", sha256.Sum256(decoded)) != "2b194ddcae5517a57956659bafbfbc4a03e7899904235738a274bbfec9587d74" {
		t.Fatal("original executable unpack differs")
	}
}
