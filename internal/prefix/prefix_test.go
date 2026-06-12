package prefix

import (
	"net/netip"
	"reflect"
	"testing"
)

func mustPrefix(t *testing.T, s string) netip.Prefix {
	t.Helper()
	p, err := netip.ParsePrefix(s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestNormalizeMasksSortsAndDeduplicates(t *testing.T) {
	got := Normalize([]netip.Prefix{
		mustPrefix(t, "2001:db8:2::1/56"),
		mustPrefix(t, "2001:db8:1::abcd/56"),
		mustPrefix(t, "2001:db8:1::/56"),
	})
	want := []string{"2001:db8:1::/56", "2001:db8:2::/56"}
	if !reflect.DeepEqual(got.Strings(), want) {
		t.Fatalf("Normalize() = %#v, want %#v", got.Strings(), want)
	}
}
