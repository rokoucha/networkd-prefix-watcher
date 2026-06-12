package state

import (
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rokoucha/networkd-prefix-watcher/internal/prefix"
)

func TestLoadNotFound(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Load() error = %v, want ErrNotFound", err)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	p := netip.MustParsePrefix("2001:db8::/56")
	want := New("pd", "wan0", 56, prefix.Normalize([]netip.Prefix{p}))
	if err := Save(path, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.UpdatedAt == "" {
		t.Fatalf("UpdatedAt is empty")
	}
	got.UpdatedAt = ""
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
}

func TestLoadBrokenJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("{"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatalf("Load() error = nil, want error")
	}
}
