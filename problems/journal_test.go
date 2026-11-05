package problems

import (
	"encoding/binary"
	"reflect"
	"strings"
	"testing"
)

func TestEncodeDecodeEntry(t *testing.T) {
	fields := []field{
		{"MESSAGE", "line 1\nline 2=x"},
		{"FROSTYARD_FRAMES", "main.main\nruntime.main"},
		{"EMPTY", ""},
	}
	b := encodeEntry(fields)
	got, err := DecodeEntry(b)
	if err != nil {
		t.Fatalf("DecodeEntry: %v", err)
	}
	want := map[string]string{"MESSAGE": "line 1\nline 2=x", "FROSTYARD_FRAMES": "main.main\nruntime.main", "EMPTY": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DecodeEntry = %q, want %q", got, want)
	}
	if !strings.HasPrefix(string(b), "MESSAGE\n") || b[len(b)-1] != '\n' {
		t.Errorf("encodeEntry = %q, want the binary form", b)
	}
}

func TestDecodeEntryPlainAndMixed(t *testing.T) {
	b := []byte("PRIORITY=3\nMESSAGE=a=b\n")
	b = append(b, encodeEntry([]field{{"CODE", "x\ny"}})...)
	got, err := DecodeEntry(b)
	if err != nil {
		t.Fatalf("DecodeEntry: %v", err)
	}
	want := map[string]string{"PRIORITY": "3", "MESSAGE": "a=b", "CODE": "x\ny"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DecodeEntry = %q, want %q", got, want)
	}
}

func TestDecodeEntryRejects(t *testing.T) {
	lenOf := func(n uint64) string {
		return string(binary.LittleEndian.AppendUint64(nil, n))
	}
	tests := map[string]string{
		"empty":             "",
		"no value":          "MESSAGE",
		"lowercase name":    "message=x\n",
		"empty name":        "=x\n",
		"underscore name":   "_UID=0\n",
		"repeated plain":    "A=1\nA=2\n",
		"repeated binary":   "A=1\nA\n" + lenOf(1) + "2\n",
		"plain no newline":  "A=1",
		"short length":      "A\n\x01\x00",
		"length too big":    "A\n" + lenOf(100) + "abc\n",
		"length overflow":   "A\n" + lenOf(1<<63) + "abc\n",
		"no final newline":  "A\n" + lenOf(3) + "abc",
		"wrong final byte":  "A\n" + lenOf(2) + "abc\n",
		"name with a space": "A B=1\n",
	}
	for name, in := range tests {
		if m, err := DecodeEntry([]byte(in)); err == nil {
			t.Errorf("%s: DecodeEntry(%q) = %q, want an error", name, in, m)
		}
	}
}
