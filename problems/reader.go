package problems

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"
)

// ReadOptions selects the records that Read returns.
type ReadOptions struct {
	// User reads the user journal (--user); else Read reads the system
	// journal (--system).
	User bool
	// Since, when not zero, keeps only records at or after it (--since).
	Since time.Time
	// Limit, when above 0, keeps only the newest Limit entries (-n).
	Limit int
	// Journalctl is the path of journalctl; the default is "journalctl".
	Journalctl string
}

// Result holds the valid records that a read found, and the number of
// entries that it dropped because they failed validation.
type Result struct {
	Records []Record
	Dropped int
}

// Record is 1 valid problem record.
type Record struct {
	// Trusted holds the fields that journald adds itself. A client cannot
	// forge them.
	Trusted Trusted
	// Detail holds the validated S8 detail fields: the only fields that can
	// reach a report.
	Detail Detail
	// Message is the MESSAGE text. It is local only and never goes into a
	// report.
	Message string
}

// Trusted holds the journald fields of a record.
type Trusted struct {
	UID             string    // _UID
	Comm            string    // _COMM
	SystemdUnit     string    // _SYSTEMD_UNIT
	SystemdUserUnit string    // _SYSTEMD_USER_UNIT
	Cursor          string    // __CURSOR
	Time            time.Time // __REALTIME_TIMESTAMP, in UTC
}

// ParseJSON reads `journalctl -o json` output: 1 JSON object for each line.
// It validates each entry with Validate. An entry that is not a JSON object,
// that has a repeated field (journalctl gives it as an array of strings) or
// a null value, or that fails Validate, is dropped and counted. A value that
// journalctl gives as an array of byte numbers (not UTF-8) is decoded first.
// ParseJSON returns an error only when it cannot read r.
func ParseJSON(r io.Reader) (Result, error) {
	var res Result
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			if rec, ok := parseEntry(line); ok {
				res.Records = append(res.Records, rec)
			} else {
				res.Dropped++
			}
		}
		if errors.Is(err, io.EOF) {
			return res, nil
		}
		if err != nil {
			return res, err
		}
	}
}

func parseEntry(line []byte) (Record, bool) {
	var rec Record
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(line, &raw); err != nil || raw == nil {
		return rec, false
	}
	fields := make(map[string]string, len(raw))
	for name, v := range raw {
		s, ok := jsonValue(v)
		if !ok {
			return rec, false
		}
		fields[name] = s
	}
	d, err := Validate(fields)
	if err != nil {
		return rec, false
	}
	rec.Detail = d
	rec.Message = fields[fieldMessage]
	rec.Trusted = Trusted{
		UID:             fields["_UID"],
		Comm:            fields["_COMM"],
		SystemdUnit:     fields["_SYSTEMD_UNIT"],
		SystemdUserUnit: fields["_SYSTEMD_USER_UNIT"],
		Cursor:          fields["__CURSOR"],
	}
	if ts, ok := fields["__REALTIME_TIMESTAMP"]; ok {
		us, err := strconv.ParseInt(ts, 10, 64)
		if err != nil {
			return rec, false
		}
		rec.Trusted.Time = time.UnixMicro(us).UTC()
	}
	return rec, true
}

// jsonValue decodes 1 journalctl JSON field value: a string, or an array of
// byte numbers. Any other value (null, an array of strings for a repeated
// field, a number) is not accepted.
func jsonValue(v json.RawMessage) (string, bool) {
	if string(v) == "null" {
		return "", false
	}
	var s string
	if err := json.Unmarshal(v, &s); err == nil {
		return s, true
	}
	var list []json.RawMessage
	if err := json.Unmarshal(v, &list); err != nil || len(list) == 0 {
		return "", false
	}
	var b strings.Builder
	for _, e := range list {
		n, err := strconv.ParseUint(string(e), 10, 8)
		if err != nil {
			return "", false
		}
		b.WriteByte(byte(n))
	}
	return b.String(), true
}
