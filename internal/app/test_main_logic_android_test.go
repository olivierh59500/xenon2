//go:build android && xenon2_logic_only

package app

import (
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"syscall"
	"testing"
)

const logicOnlyTests = true

// This executable runs frontend logic without an Android view or graphics
// context. Dedicated graphics fixtures skip; optional pixel captures do nothing.
func TestMain(m *testing.M) {
	var output *os.File
	code := 2
	if name := os.Getenv("XENON2_TEST_OUTPUT"); name != "" {
		var err error
		output, err = os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		if err == nil {
			err = syscall.Dup3(int(output.Fd()), 1, 0)
		}
		if err == nil {
			err = syscall.Dup3(int(output.Fd()), 2, 0)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "capture Android logic test output:", err)
		} else {
			os.Stdout, os.Stderr = output, output
			code = runAndroidLogicTests(m)
		}
	} else {
		code = runAndroidLogicTests(m)
	}
	if name := os.Getenv("XENON2_TEST_STATUS"); name != "" {
		if err := os.WriteFile(name, []byte(fmt.Sprintf("exit=%d\n", code)), 0600); err != nil {
			fmt.Fprintln(os.Stderr, "write Android logic test status:", err)
			code = 2
		}
	}
	if output != nil {
		output.Close()
	}
	os.Exit(code)
}

func runAndroidLogicTests(m *testing.M) int {
	if !flag.Parsed() {
		flag.Parse()
	}
	if encoded := os.Getenv("XENON2_TEST_PATTERN_BASE64"); encoded != "" {
		pattern, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			fmt.Fprintln(os.Stderr, "decode Android logic test pattern:", err)
			return 2
		}
		if err := flag.Set("test.run", string(pattern)); err != nil {
			fmt.Fprintln(os.Stderr, "select Android logic tests:", err)
			return 2
		}
	}
	fmt.Fprintln(os.Stdout, "Android frontend logic tests: Draw and ReadPixels are disabled.")
	return m.Run()
}
