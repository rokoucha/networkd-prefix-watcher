package detector

import (
	"net/netip"
	"testing"
)

func TestFilterRoute(t *testing.T) {
	good := RouteCandidate{
		DstPrefix: netip.MustParsePrefix("2001:db8::1/56"),
		Type:      RouteTypeUnreach,
		Protocol:  RouteProtocolDHCP,
	}
	p, ok := FilterRoute(good, 56)
	if !ok || p.String() != "2001:db8::/56" {
		t.Fatalf("FilterRoute() = %v, %v", p, ok)
	}

	for name, c := range map[string]RouteCandidate{
		"wrong length": {DstPrefix: netip.MustParsePrefix("2001:db8::/60"), Type: RouteTypeUnreach, Protocol: RouteProtocolDHCP},
		"wrong type":   {DstPrefix: netip.MustParsePrefix("2001:db8::/56"), Type: RouteTypeUnicast, Protocol: RouteProtocolDHCP},
		"wrong proto":  {DstPrefix: netip.MustParsePrefix("2001:db8::/56"), Type: RouteTypeUnreach, Protocol: 0},
		"ipv4":         {DstPrefix: netip.MustParsePrefix("192.0.2.0/24"), Type: RouteTypeUnreach, Protocol: RouteProtocolDHCP},
	} {
		t.Run(name, func(t *testing.T) {
			if _, ok := FilterRoute(c, 56); ok {
				t.Fatalf("FilterRoute() ok = true, want false")
			}
		})
	}
}

func TestFilterAddress(t *testing.T) {
	p, ok := FilterAddress(AddressCandidate{Address: netip.MustParseAddr("2001:db8:1::abcd"), IfIndex: 2}, 2, 64)
	if !ok || p.String() != "2001:db8:1::/64" {
		t.Fatalf("FilterAddress() = %v, %v", p, ok)
	}

	for name, c := range map[string]AddressCandidate{
		"wrong ifindex": {Address: netip.MustParseAddr("2001:db8:1::1"), IfIndex: 3},
		"link local":    {Address: netip.MustParseAddr("fe80::1"), IfIndex: 2},
		"ula":           {Address: netip.MustParseAddr("fd00::1"), IfIndex: 2},
		"ipv4":          {Address: netip.MustParseAddr("192.0.2.1"), IfIndex: 2},
	} {
		t.Run(name, func(t *testing.T) {
			if _, ok := FilterAddress(c, 2, 64); ok {
				t.Fatalf("FilterAddress() ok = true, want false")
			}
		})
	}
}
