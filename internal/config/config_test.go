package config

import (
	"strings"
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	valid := Config{
		Mode:      ModePDRoute,
		PrefixLen: 56,
		Interface: "wan0",
		Target:    "networkd-prefix-changed.target",
		StateFile: "/tmp/state.json",
		EnvFile:   "/tmp/prefix.env",
		Debounce:  time.Second,
	}

	for name, mutate := range map[string]func(*Config){
		"valid pd": func(*Config) {},
		"valid ra": func(c *Config) { c.Mode = ModeRAAddress; c.PrefixLen = 64 },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := valid
			mutate(&cfg)
			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}

	for name, tc := range map[string]struct {
		mutate func(*Config)
		want   string
	}{
		"missing mode":       {func(c *Config) { c.Mode = "" }, "--mode is required"},
		"bad mode":           {func(c *Config) { c.Mode = "route" }, "--mode must be"},
		"missing prefix len": {func(c *Config) { c.PrefixLen = -1 }, "--prefix-len is required"},
		"bad prefix len":     {func(c *Config) { c.PrefixLen = 129 }, "--prefix-len must be"},
		"missing interface":  {func(c *Config) { c.Interface = "" }, "--interface is required"},
		"negative debounce":  {func(c *Config) { c.Debounce = -time.Second }, "--debounce must not be negative"},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := valid
			tc.mutate(&cfg)
			err := cfg.Validate()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() error = %v, want containing %q", err, tc.want)
			}
		})
	}
}
