package trigger

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSystemctlRunnerArgs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test")
	}
	dir := t.TempDir()
	logPath := filepath.Join(dir, "args")
	script := filepath.Join(dir, "systemctl")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$SYSTEMCTL_ARGS_LOG\"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SYSTEMCTL_ARGS_LOG", logPath)

	err := SystemctlRunner{}.RestartTarget(context.Background(), "networkd-prefix-changed.target")
	if err != nil {
		t.Fatalf("RestartTarget() error = %v", err)
	}
	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Split(strings.TrimSpace(string(b)), "\n")
	want := []string{"restart", "--no-block", "networkd-prefix-changed.target"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}
