package app

import (
	"context"
	"log"
	"net/netip"
	"path/filepath"
	"testing"
	"time"

	"github.com/rokoucha/networkd-prefix-watcher/internal/config"
	"github.com/rokoucha/networkd-prefix-watcher/internal/prefix"
)

type fakeRunner struct {
	targets []string
}

func (r *fakeRunner) RestartTarget(_ context.Context, target string) error {
	r.targets = append(r.targets, target)
	return nil
}

func TestReconcileInitialDoesNotTriggerByDefault(t *testing.T) {
	cfg := testConfig(t)
	runner := &fakeRunner{}
	cur := prefix.Normalize([]netip.Prefix{netip.MustParsePrefix("2001:db8::/56")})
	if err := reconcile(context.Background(), cfg, log.Default(), runner, cur); err != nil {
		t.Fatalf("reconcile() error = %v", err)
	}
	if len(runner.targets) != 0 {
		t.Fatalf("triggered on initial state")
	}
}

func TestReconcileChangeTriggers(t *testing.T) {
	cfg := testConfig(t)
	runner := &fakeRunner{}
	first := prefix.Normalize([]netip.Prefix{netip.MustParsePrefix("2001:db8::/56")})
	second := prefix.Normalize([]netip.Prefix{netip.MustParsePrefix("2001:db8:100::/56")})
	if err := reconcile(context.Background(), cfg, log.Default(), runner, first); err != nil {
		t.Fatal(err)
	}
	if err := reconcile(context.Background(), cfg, log.Default(), runner, second); err != nil {
		t.Fatal(err)
	}
	if len(runner.targets) != 1 {
		t.Fatalf("trigger count = %d, want 1", len(runner.targets))
	}
}

func testConfig(t *testing.T) config.Config {
	t.Helper()
	dir := t.TempDir()
	return config.Config{
		Mode:      config.ModePDRoute,
		PrefixLen: 56,
		Interface: "wan0",
		Target:    "networkd-prefix-changed.target",
		StateFile: filepath.Join(dir, "state.json"),
		EnvFile:   filepath.Join(dir, "prefix.env"),
		Debounce:  time.Millisecond,
	}
}
