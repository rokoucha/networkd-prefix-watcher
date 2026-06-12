package envfile

import (
	"strings"

	"github.com/rokoucha/networkd-prefix-watcher/internal/atomicfile"
	"github.com/rokoucha/networkd-prefix-watcher/internal/prefix"
)

func Write(path string, current, previous prefix.Set) error {
	var b strings.Builder
	b.WriteString("PREFIXES=")
	writeQuoted(&b, current.Join())
	b.WriteByte('\n')
	b.WriteString("PREVIOUS_PREFIXES=")
	writeQuoted(&b, previous.Join())
	b.WriteByte('\n')
	return atomicfile.Write(path, []byte(b.String()), 0644)
}

func writeQuoted(b *strings.Builder, value string) {
	b.WriteByte('"')
	for _, r := range value {
		if r == '\\' || r == '"' || r == '$' || r == '`' {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	b.WriteByte('"')
}
