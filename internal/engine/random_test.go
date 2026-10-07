package engine

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestRandomStateOriginalSequence(t *testing.T) {
	random := NewRandomState()
	for step, expected := range []uint32{2554442909, 4216165471, 2475594233, 2396736689, 577355092, 2974094347, 3551437502, 2230545811, 1487020194, 3717602410} {
		if actual := random.Next(); actual != expected {
			t.Fatalf("random step %d: got %d, expected %d", step, actual, expected)
		}
	}
	before := random
	if _, err := random.Below(0); err == nil || random != before {
		t.Fatal("an invalid bound must fail without consuming the random stream")
	}
	for _, limit := range []uint16{1, 2, 7, 100, 300, 65535} {
		value, err := random.Below(limit)
		if err != nil || value >= limit {
			t.Fatalf("bound %d: value=%d error=%v", limit, value, err)
		}
	}
}

func TestRandomStateNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local original traces")
	}
	file, err := os.Open(filepath.Join(root, "random-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows[1:] {
		values := make([]uint64, len(row))
		for i, field := range row {
			values[i], err = strconv.ParseUint(field, 10, 32)
			if err != nil {
				t.Fatal(err)
			}
		}
		random := RandomState{A: uint32(values[2]), B: uint32(values[3])}
		actual := random.Next()
		if actual != uint32(values[4]) || random.A != uint32(values[5]) || random.B != uint32(values[6]) {
			t.Fatalf("case %d step %d: result %d state %+v, expected %v", values[0], values[1], actual, random, row)
		}
	}
}
