package problems

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

// jsonLine returns 1 journalctl JSON line for fields; a value is a string or
// any other JSON value.
func jsonLine(t *testing.T, fields map[string]any) string {
	t.Helper()
	b, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func goErrorEntry() map[string]any {
	m := map[string]any{
		"__CURSOR":             "s=abc;i=1",
		"__REALTIME_TIMESTAMP": "1800000000123456",
		"_UID":                 "1000",
		"_COMM":                "nsl",
		"_SYSTEMD_UNIT":        "user@1000.service",
		"_SYSTEMD_USER_UNIT":   "nsl.service",
		"PRIORITY":             "3",
		"MESSAGE":              "open: permission denied",
	}
	for k, v := range validFields() {
		if _, ok := m[k]; !ok {
			m[k] = v
		}
	}
	m["MESSAGE"] = "open: permission denied"
	return m
}

func TestParseJSON(t *testing.T) {
	rust := goErrorEntry()
	rust["FROSTYARD_RUNTIME"] = "rust"
	rust["FROSTYARD_CLASS"] = "panic"
	rust["FROSTYARD_LOCATION"] = "src/main.rs:7"
	delete(rust, "FROSTYARD_CODE")

	bytesMsg := goErrorEntry()
	bytesMsg["MESSAGE"] = []int{0xff, 'a'}
	bytesMsg["FROSTYARD_CODE"] = []int{'i', 'o'}

	goPanic := goErrorEntry()
	goPanic["FROSTYARD_CLASS"] = "panic"
	goPanic["FROSTYARD_FRAMES"] = "main.f\nmain.main"
	delete(goPanic, "FROSTYARD_CODE")
	delete(goPanic, "__REALTIME_TIMESTAMP")

	forged := goErrorEntry()
	forged["FROSTYARD_APP_ID"] = "org.example.app"
	repeated := goErrorEntry()
	repeated["FROSTYARD_CODE"] = []string{"a", "b"}
	repeatedMsg := goErrorEntry()
	repeatedMsg["MESSAGE"] = []string{"a", "b"}
	null := goErrorEntry()
	null["FROSTYARD_FRAMES"] = nil
	badByte := goErrorEntry()
	badByte["MESSAGE"] = []int{256}
	emptyArray := goErrorEntry()
	emptyArray["MESSAGE"] = []int{}
	number := goErrorEntry()
	number["PRIORITY"] = 3
	badTime := goErrorEntry()
	badTime["__REALTIME_TIMESTAMP"] = "soon"

	lines := []string{
		jsonLine(t, goErrorEntry()),
		"",
		jsonLine(t, rust),
		jsonLine(t, bytesMsg),
		jsonLine(t, goPanic),
		jsonLine(t, forged),
		jsonLine(t, repeated),
		jsonLine(t, repeatedMsg),
		jsonLine(t, null),
		jsonLine(t, badByte),
		jsonLine(t, emptyArray),
		jsonLine(t, number),
		jsonLine(t, badTime),
		"[1,2]",
		"null",
		"not json",
	}
	res, err := ParseJSON(strings.NewReader(strings.Join(lines, "\n")))
	if err != nil {
		t.Fatalf("ParseJSON = %v", err)
	}
	if res.Dropped != 11 {
		t.Errorf("Dropped = %d, want 11", res.Dropped)
	}
	if len(res.Records) != 4 {
		t.Fatalf("got %d records, want 4: %+v", len(res.Records), res.Records)
	}

	first := res.Records[0]
	wantTrusted := Trusted{
		UID: "1000", Comm: "nsl", SystemdUnit: "user@1000.service", SystemdUserUnit: "nsl.service",
		Cursor: "s=abc;i=1", Time: time.UnixMicro(1800000000123456).UTC(),
	}
	if first.Trusted != wantTrusted {
		t.Errorf("Trusted = %+v, want %+v", first.Trusted, wantTrusted)
	}
	if first.Message != "open: permission denied" || first.Detail.Code != "a.b" {
		t.Errorf("record = %+v", first)
	}
	if d := res.Records[1].Detail; d.Runtime != "rust" || d.Location != "src/main.rs:7" {
		t.Errorf("rust detail = %+v", d)
	}
	if r := res.Records[2]; r.Message != "\xffa" || r.Detail.Code != "io" {
		t.Errorf("byte array record = %+v", r)
	}
	if r := res.Records[3]; !reflect.DeepEqual(r.Detail.Frames, []string{"main.f", "main.main"}) || !r.Trusted.Time.IsZero() {
		t.Errorf("go panic record = %+v", r)
	}
}

type failReader struct{}

func (failReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestParseJSONReadError(t *testing.T) {
	if _, err := ParseJSON(failReader{}); err == nil {
		t.Error("ParseJSON(failing reader): want an error")
	}
	res, err := ParseJSON(strings.NewReader(""))
	if err != nil || len(res.Records) != 0 || res.Dropped != 0 {
		t.Errorf("ParseJSON(empty) = %+v, %v", res, err)
	}
}
