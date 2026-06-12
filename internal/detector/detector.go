package detector

import (
	"context"
	"net/netip"

	"github.com/rokoucha/networkd-prefix-watcher/internal/config"
	"github.com/rokoucha/networkd-prefix-watcher/internal/prefix"
)

type Detector interface {
	Snapshot(context.Context) (prefix.Set, error)
	Wait(context.Context) error
	Close() error
}

type RouteCandidate struct {
	DstPrefix netip.Prefix
	Type      uint8
	Protocol  uint8
}

type AddressCandidate struct {
	Address netip.Addr
	IfIndex int
}

const (
	RouteProtocolDHCP = 16
	RouteTypeUnspec   = 0
	RouteTypeUnicast  = 1
	RouteTypeUnreach  = 7
)

func FilterRoute(c RouteCandidate, prefixLen int) (netip.Prefix, bool) {
	if !c.DstPrefix.IsValid() || !c.DstPrefix.Addr().Is6() {
		return netip.Prefix{}, false
	}
	if c.Type != RouteTypeUnreach || c.Protocol != RouteProtocolDHCP {
		return netip.Prefix{}, false
	}
	if c.DstPrefix.Bits() != prefixLen {
		return netip.Prefix{}, false
	}
	return c.DstPrefix.Masked(), true
}

func FilterAddress(c AddressCandidate, wantIfIndex int, prefixLen int) (netip.Prefix, bool) {
	if c.IfIndex != wantIfIndex {
		return netip.Prefix{}, false
	}
	if !isWatchedGlobalIPv6(c.Address) {
		return netip.Prefix{}, false
	}
	p := netip.PrefixFrom(c.Address, prefixLen)
	return p.Masked(), true
}

func isWatchedGlobalIPv6(addr netip.Addr) bool {
	return addr.IsValid() &&
		addr.Is6() &&
		addr.IsGlobalUnicast() &&
		!addr.IsPrivate()
}

func New(cfg config.Config) (Detector, error) {
	return newNetlinkDetector(cfg)
}
