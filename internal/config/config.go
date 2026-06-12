package config

import (
	"fmt"
	"time"
)

const (
	ModePDRoute   = "pd"
	ModeRAAddress = "ra"
)

type Config struct {
	Mode           string
	PrefixLen      int
	Interface      string
	Target         string
	StateFile      string
	EnvFile        string
	Debounce       time.Duration
	Once           bool
	TriggerOnStart bool
}

func (c Config) Validate() error {
	switch c.Mode {
	case ModePDRoute, ModeRAAddress:
	case "":
		return fmt.Errorf("--mode is required")
	default:
		return fmt.Errorf("--mode must be %q or %q", ModePDRoute, ModeRAAddress)
	}

	if c.PrefixLen < 0 {
		return fmt.Errorf("--prefix-len is required")
	}
	if c.PrefixLen < 1 || c.PrefixLen > 128 {
		return fmt.Errorf("--prefix-len must be between 1 and 128")
	}
	if c.Interface == "" {
		return fmt.Errorf("--interface is required")
	}
	if c.Target == "" {
		return fmt.Errorf("--target must not be empty")
	}
	if c.StateFile == "" {
		return fmt.Errorf("--state-file must not be empty")
	}
	if c.EnvFile == "" {
		return fmt.Errorf("--env-file must not be empty")
	}
	if c.Debounce < 0 {
		return fmt.Errorf("--debounce must not be negative")
	}
	return nil
}
