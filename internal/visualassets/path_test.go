package visualassets

import (
	"encoding/binary"
	"testing"
)

func TestPathBranchTargetsUseCommandIndices(t *testing.T) {
	words := []int16{10, 40, -53, 2, 64, 0, 0, 7, 8, 6, 0}
	data := make([]byte, len(words)*2)
	for i, word := range words {
		binary.BigEndian.PutUint16(data[i*2:], uint16(word))
	}
	commands, err := decodePath(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) != 4 || commands[0].Kind != "origin" || commands[0].Y != -53 || commands[2].Kind != "jump" || commands[2].Target != 1 {
		t.Fatalf("decoded commands=%+v", commands)
	}
	binary.BigEndian.PutUint16(data[18:], 7)
	if _, err := decodePath(data); err == nil {
		t.Fatal("jump into the middle of a command accepted")
	}
}
