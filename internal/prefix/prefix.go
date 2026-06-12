package prefix

import (
	"net/netip"
	"sort"
	"strings"
)

type Set []netip.Prefix

func Normalize(prefixes []netip.Prefix) Set {
	out := make([]netip.Prefix, 0, len(prefixes))
	seen := map[netip.Prefix]struct{}{}
	for _, p := range prefixes {
		if !p.IsValid() {
			continue
		}
		p = p.Masked()
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].String() < out[j].String()
	})
	return Set(out)
}

func (s Set) Strings() []string {
	out := make([]string, len(s))
	for i, p := range s {
		out[i] = p.String()
	}
	return out
}

func (s Set) Join() string {
	return strings.Join(s.Strings(), " ")
}

func (s Set) Equal(other Set) bool {
	if len(s) != len(other) {
		return false
	}
	for i := range s {
		if s[i] != other[i] {
			return false
		}
	}
	return true
}
