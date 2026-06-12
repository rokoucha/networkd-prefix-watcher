//go:build !linux

package detector

import (
	"context"
	"fmt"

	"github.com/rokoucha/networkd-prefix-watcher/internal/config"
	"github.com/rokoucha/networkd-prefix-watcher/internal/prefix"
)

type unsupportedDetector struct{}

func newNetlinkDetector(config.Config) (Detector, error) {
	return unsupportedDetector{}, nil
}

func (unsupportedDetector) Snapshot(context.Context) (prefix.Set, error) {
	return nil, fmt.Errorf("netlink detector is only supported on Linux")
}

func (unsupportedDetector) Wait(context.Context) error {
	return fmt.Errorf("netlink detector is only supported on Linux")
}

func (unsupportedDetector) Close() error { return nil }
