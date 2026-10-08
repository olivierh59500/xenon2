//go:build android && xenon2_logic_only

package app

import (
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
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
	if err := writeAndroidLogicTestStatus(code); err != nil {
		fmt.Fprintln(os.Stderr, "write Android logic test status:", err)
		code = 2
	}
	if output != nil {
		output.Sync()
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
	watchdog, err := androidLogicWatchdogDuration(os.Getenv("XENON2_TEST_WATCHDOG_MS"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "configure Android logic test watchdog:", err)
		return 2
	}
	var finished atomic.Bool
	timer := time.AfterFunc(watchdog, func() {
		if !finished.CompareAndSwap(false, true) {
			return
		}
		fmt.Fprintf(os.Stderr, "\nAndroid logic test watchdog TIMEOUT after %s; this is a runner timeout, not a gameplay assertion failure.\n", watchdog)
		for size := 64 << 10; ; size *= 2 {
			stack := make([]byte, size)
			n := runtime.Stack(stack, true)
			if n < len(stack) {
				os.Stderr.Write(stack[:n])
				break
			}
		}
		os.Stdout.Sync()
		os.Stderr.Sync()
		if err := writeAndroidLogicTestStatus(124); err != nil {
			fmt.Fprintln(os.Stderr, "write Android logic watchdog status:", err)
			os.Stderr.Sync()
		}
		os.Exit(124)
	})
	defer timer.Stop()
	fmt.Fprintln(os.Stdout, "Android frontend logic tests: Draw and ReadPixels are disabled.")
	code := m.Run()
	if !finished.CompareAndSwap(false, true) {
		// The watchdog owns completion once its deadline wins the race. Avoid
		// overwriting its exit=124 status with an ordinary return status.
		select {}
	}
	return code
}

func androidLogicWatchdogDuration(value string) (time.Duration, error) {
	const maximumMilliseconds = 470000 // Ten seconds before the runner's eight-minute Go timeout.
	if value == "" {
		return maximumMilliseconds * time.Millisecond, nil
	}
	milliseconds, err := strconv.Atoi(value)
	if err != nil || milliseconds <= 0 || milliseconds > maximumMilliseconds {
		return 0, fmt.Errorf("XENON2_TEST_WATCHDOG_MS must be an integer from 1 to %d", maximumMilliseconds)
	}
	return time.Duration(milliseconds) * time.Millisecond, nil
}

func writeAndroidLogicTestStatus(code int) error {
	name := os.Getenv("XENON2_TEST_STATUS")
	if name == "" {
		return nil
	}
	status, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(status, "exit=%d\n", code)
	if err == nil {
		err = status.Sync()
	}
	if closeErr := status.Close(); err == nil {
		err = closeErr
	}
	return err
}
