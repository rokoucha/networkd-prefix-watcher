package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"time"

	"github.com/rokoucha/networkd-prefix-watcher/internal/atomicfile"
	"github.com/rokoucha/networkd-prefix-watcher/internal/prefix"
)

var ErrNotFound = errors.New("state not found")

type State struct {
	Mode      string   `json:"mode"`
	Interface string   `json:"interface"`
	PrefixLen int      `json:"prefix_len"`
	Prefixes  []string `json:"prefixes"`
	UpdatedAt string   `json:"updated_at"`
}

func Load(path string) (State, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return State{}, ErrNotFound
	}
	if err != nil {
		return State{}, err
	}
	var st State
	if err := json.Unmarshal(b, &st); err != nil {
		return State{}, err
	}
	return st, nil
}

func Save(path string, st State) error {
	st.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return atomicfile.Write(path, b, 0644)
}

func New(mode, iface string, prefixLen int, set prefix.Set) State {
	return State{
		Mode:      mode,
		Interface: iface,
		PrefixLen: prefixLen,
		Prefixes:  set.Strings(),
	}
}

func (s State) PrefixSet() (prefix.Set, error) {
	items := make([]netip.Prefix, 0, len(s.Prefixes))
	for _, raw := range s.Prefixes {
		p, err := netip.ParsePrefix(raw)
		if err != nil {
			return nil, fmt.Errorf("parse stored prefix %q: %w", raw, err)
		}
		items = append(items, p)
	}
	return prefix.Normalize(items), nil
}
