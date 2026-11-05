package problems

import (
	"encoding/binary"
	"errors"
	"fmt"
	"regexp"
)

// field is 1 journal field of a record, in write order.
type field struct {
	name, value string
}

// fieldNameRe is the shape of a journal field name that a client may write.
var fieldNameRe = regexp.MustCompile(`^[A-Z0-9_]+$`)

// encodeEntry encodes fields in the binary form of the journal native
// protocol: for each field, NAME, a newline, the value length as a
// little-endian uint64, the value, and a newline.
func encodeEntry(fields []field) []byte {
	size := 0
	for _, f := range fields {
		size += len(f.name) + 1 + 8 + len(f.value) + 1
	}
	b := make([]byte, 0, size)
	for _, f := range fields {
		b = append(b, f.name...)
		b = append(b, '\n')
		b = binary.LittleEndian.AppendUint64(b, uint64(len(f.value)))
		b = append(b, f.value...)
		b = append(b, '\n')
	}
	return b
}

// DecodeEntry decodes 1 journal native protocol datagram. It reads the
// binary form (NAME\n, a little-endian uint64 length, the value, \n) and the
// plain form (NAME=value\n). It rejects an empty entry, a repeated field, a
// bad length, a missing final newline, and a field name that is not
// ^[A-Z0-9_]+$ or that starts with "_".
func DecodeEntry(b []byte) (map[string]string, error) {
	if len(b) == 0 {
		return nil, errors.New("problems: empty entry")
	}
	out := make(map[string]string)
	for len(b) > 0 {
		i := 0
		for i < len(b) && b[i] != '\n' && b[i] != '=' {
			i++
		}
		if i == len(b) {
			return nil, errors.New("problems: field without a value")
		}
		name := string(b[:i])
		if !fieldNameRe.MatchString(name) || name[0] == '_' {
			return nil, fmt.Errorf("problems: bad field name %q", name)
		}
		if _, dup := out[name]; dup {
			return nil, fmt.Errorf("problems: repeated field %s", name)
		}
		var value string
		if b[i] == '=' {
			rest := b[i+1:]
			j := 0
			for j < len(rest) && rest[j] != '\n' {
				j++
			}
			if j == len(rest) {
				return nil, fmt.Errorf("problems: field %s has no final newline", name)
			}
			value = string(rest[:j])
			b = rest[j+1:]
		} else {
			rest := b[i+1:]
			if len(rest) < 8 {
				return nil, fmt.Errorf("problems: field %s has a short length", name)
			}
			n := binary.LittleEndian.Uint64(rest)
			rest = rest[8:]
			if n >= uint64(len(rest)) || rest[n] != '\n' {
				return nil, fmt.Errorf("problems: field %s has a bad length", name)
			}
			value = string(rest[:n])
			b = rest[n+1:]
		}
		out[name] = value
	}
	return out, nil
}
