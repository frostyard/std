// Package problems writes problem records (panics and coded errors) of a
// frostyard app to the local systemd journal, and reads them back.
//
// The contract is frostyard/core docs/specs/problem-report.md (schema
// frostyard.problem/1, kind S8 app_problem, and the journal record
// registry). The package never sends anything over a network: a record is 1
// datagram to the local journal socket, and a record that does not fit, or
// that the journal cannot take at once, is dropped.
//
// A Recorder writes records. Read and ParseJSON read them back from
// journalctl, and Validate is the one validation path that every reader
// uses: anyone can write FROSTYARD_* fields to the journal, so every record
// is untrusted until it passes Validate.
package problems

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// MessageID is the journal MESSAGE_ID of every problem record.
const MessageID = "8915bdd215c84cbab0c31c05a1a33e7e"

// Schema is the value of FROSTYARD_SCHEMA: the schema version whose
// patterns the record's values follow.
const Schema = "frostyard.problem/1"

// Component is the part of an app that recorded a problem (S8 component).
type Component string

// The S8 component enum.
const (
	ComponentDaemon Component = "daemon"
	ComponentCLI    Component = "cli"
	ComponentGUI    Component = "gui"
	ComponentShell  Component = "shell"
	ComponentOther  Component = "other"
)

// Limits from the spec and this package.
const (
	maxFrames      = 32
	maxFrameLen    = 160
	maxLocationLen = 128
	maxMessageLen  = 2048
	otherFrame     = "other"
)

// Journal field names of the registry.
const (
	fieldMessageID  = "MESSAGE_ID"
	fieldPriority   = "PRIORITY"
	fieldMessage    = "MESSAGE"
	fieldSchema     = "FROSTYARD_SCHEMA"
	fieldAppID      = "FROSTYARD_APP_ID"
	fieldAppVersion = "FROSTYARD_APP_VERSION"
	fieldRuntime    = "FROSTYARD_RUNTIME"
	fieldComponent  = "FROSTYARD_COMPONENT"
	fieldClass      = "FROSTYARD_CLASS"
	fieldCode       = "FROSTYARD_CODE"
	fieldFrames     = "FROSTYARD_FRAMES"
	fieldLocation   = "FROSTYARD_LOCATION"
)

// Patterns copied exactly from core docs/specs/schemas/problem-report/v1.json.
var (
	semverRe   = regexp.MustCompile(`^(0|[1-9][0-9]{0,5})\.(0|[1-9][0-9]{0,5})\.(0|[1-9][0-9]{0,5})(-(alpha|beta|rc)\.(0|[1-9][0-9]{0,3}))?$`)
	codeRe     = regexp.MustCompile(`^[a-z][a-z0-9_.]{0,63}$`)
	frameRe    = regexp.MustCompile(`^(other|(github\.com/frostyard(/[A-Za-z0-9_-]+){1,6}|(github\.com/emersion/go-imap/v2|archive|bufio|bytes|cmp|compress|container|context|crypto|database|debug|embed|encoding|errors|expvar|flag|fmt|go|hash|html|image|index|internal|io|iter|log|main|maps|math|mime|net|os|path|plugin|reflect|regexp|runtime|slices|sort|strconv|strings|structs|sync|syscall|testing|text|time|unicode|unique|unsafe|vendor|weak)(/[a-z0-9_]+){0,4})\.[A-Za-z0-9_.()*\[\]-]+)$`)
	locationRe = regexp.MustCompile(`^src/([a-z0-9_]+/){0,8}[a-z0-9_]+\.rs:[1-9][0-9]{0,5}$`)
)

// appIDs is the S8 app ID allowlist (closed).
var appIDs = map[string]bool{
	"org.frostyard.ChairLift":  true,
	"org.frostyard.nsl":        true,
	"org.frostyard.intuneme":   true,
	"org.frostyard.pilothouse": true,
	"org.frostyard.Frostmail":  true,
	"org.frostyard.Frostnotes": true,
	"org.frostyard.Frostread":  true,
}

var (
	components = map[string]bool{"daemon": true, "cli": true, "gui": true, "shell": true, "other": true}
	runtimes   = map[string]bool{"go": true, "rust": true}
	classes    = map[string]bool{"panic": true, "error": true}
	// registry holds every FROSTYARD_* field that the journal record
	// registry names; any other FROSTYARD_* field makes a record invalid.
	registry = map[string]bool{
		fieldSchema: true, fieldAppID: true, fieldAppVersion: true, fieldRuntime: true,
		fieldComponent: true, fieldClass: true, fieldCode: true, fieldFrames: true, fieldLocation: true,
	}
)

// MapFrame returns name when it is an allowed Go frame of schema v1 (it
// matches the v1 frame pattern and is at most 160 bytes), and "other"
// otherwise.
func MapFrame(name string) string {
	if allowedFrame(name) {
		return name
	}
	return otherFrame
}

// allowedFrame reports whether name is a frame that schema v1 allows: it
// matches the v1 frame pattern and is at most 160 bytes. MapFrame (the
// writer) and Validate (the reader) both use it.
func allowedFrame(name string) bool {
	return len(name) <= maxFrameLen && frameRe.MatchString(name)
}

// Detail holds the S8 app_problem detail fields of a valid record. Its JSON
// form is the S8 detail object.
type Detail struct {
	AppID      string   `json:"app_id"`
	AppVersion string   `json:"app_version"`
	Runtime    string   `json:"runtime"`
	Component  string   `json:"component"`
	Class      string   `json:"class"`
	Code       string   `json:"code,omitempty"`
	Frames     []string `json:"frames,omitempty"`
	Location   string   `json:"location,omitempty"`
}

// ErrInvalid is wrapped by every error that Validate returns.
var ErrInvalid = errors.New("problems: invalid record")

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInvalid}, args...)...)
}

// Validate checks the journal fields of 1 record against schema v1 and the
// journal record registry, and returns its S8 detail. It is the one
// validation path of every reader: a record that fails is dropped. Fields
// that do not start with FROSTYARD_ (other than MESSAGE_ID) are not checked.
func Validate(fields map[string]string) (Detail, error) {
	var d Detail
	if fields[fieldMessageID] != MessageID {
		return d, invalid("wrong MESSAGE_ID")
	}
	for name := range fields {
		if strings.HasPrefix(name, "FROSTYARD_") && !registry[name] {
			return d, invalid("field %s is not in the registry", name)
		}
	}
	if fields[fieldSchema] != Schema {
		return d, invalid("unknown schema")
	}
	d.AppID = fields[fieldAppID]
	if !appIDs[d.AppID] {
		return d, invalid("app id not in the allowlist")
	}
	d.AppVersion = fields[fieldAppVersion]
	if !semverRe.MatchString(d.AppVersion) {
		return d, invalid("bad app version")
	}
	d.Runtime = fields[fieldRuntime]
	if !runtimes[d.Runtime] {
		return d, invalid("unknown runtime")
	}
	d.Component = fields[fieldComponent]
	if !components[d.Component] {
		return d, invalid("unknown component")
	}
	d.Class = fields[fieldClass]
	if !classes[d.Class] {
		return d, invalid("unknown class")
	}

	code, hasCode := fields[fieldCode]
	frames, hasFrames := fields[fieldFrames]
	location, hasLocation := fields[fieldLocation]

	switch d.Class {
	case "error":
		if !hasCode {
			return d, invalid("error without a code")
		}
		if hasFrames || hasLocation {
			return d, invalid("error with frames or location")
		}
	case "panic":
		if hasCode {
			return d, invalid("panic with a code")
		}
	}
	if hasCode {
		if !codeRe.MatchString(code) {
			return d, invalid("bad code")
		}
		d.Code = code
	}
	if hasFrames {
		if d.Runtime != "go" {
			return d, invalid("frames on a %s record", d.Runtime)
		}
		if frames == "" {
			return d, invalid("empty frames")
		}
		list := strings.Split(frames, "\n")
		if len(list) > maxFrames {
			return d, invalid("more than %d frames", maxFrames)
		}
		for _, f := range list {
			if !allowedFrame(f) {
				return d, invalid("bad frame")
			}
		}
		d.Frames = list
	}
	if hasLocation {
		if d.Runtime != "rust" {
			return d, invalid("location on a %s record", d.Runtime)
		}
		if len(location) > maxLocationLen || !locationRe.MatchString(location) {
			return d, invalid("bad location")
		}
		d.Location = location
	}
	return d, nil
}
