// Package accumulator infers the shape of the fixm.raw feed and holds it as a
// pure, Kafka-free model that can be snapshotted to / loaded from JSON.
package accumulator

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultMaxEnum = 256 // distinct values enumerated before a categorical field is treated as free-text
	maxSamples     = 5   // example values kept per field, for all types
)

type ValueKind string

const (
	KindString    ValueKind = "string"
	KindInt       ValueKind = "int"
	KindFloat     ValueKind = "float"
	KindBool      ValueKind = "bool"
	KindTimestamp ValueKind = "timestamp"
	KindDuration  ValueKind = "duration"
	KindUUID      ValueKind = "uuid"
	KindGeoPair   ValueKind = "geo"
	KindEmpty     ValueKind = "empty"
)

type Field struct {
	Path        string              `json:"path"`
	Kind        string              `json:"kind"` // element | attribute | text
	Occurrences int64               `json:"occurrences"`
	FirstSeen   time.Time           `json:"firstSeen"`
	LastSeen    time.Time           `json:"lastSeen"`
	Types       map[ValueKind]int64 `json:"types,omitempty"` // type -> count; >1 key = mixed/drift
	Distinct    int64               `json:"distinct,omitempty"`
	Truncated   bool                `json:"truncated,omitempty"`
	Values      map[string]int64    `json:"values,omitempty"`  // value -> count, categoricals only
	Samples     []string            `json:"samples,omitempty"` // a few examples, any type
}

func (f *Field) addValue(value string, maxEnum int) {
	if f.Types == nil {
		f.Types = map[ValueKind]int64{}
	}
	k := classify(value)
	f.Types[k]++
	f.addSample(value)

	// Enumerate values only for categorical types; continuous ones are unbounded noise.
	if !categorical(k) {
		return
	}
	if f.Truncated { // already proven high-cardinality
		return
	}
	if f.Values == nil {
		f.Values = map[string]int64{}
	}
	switch {
	case f.Values[value] > 0:
		f.Values[value]++
	case len(f.Values) < maxEnum:
		f.Values[value] = 1
	default:
		// Over the cap: not an enum. Drop the sample, just flag high-cardinality.
		f.Truncated = true
		f.Distinct = int64(len(f.Values))
		f.Values = nil
		return
	}
	f.Distinct = int64(len(f.Values))
}

func (f *Field) addSample(value string) {
	if len(f.Samples) >= maxSamples {
		return
	}
	for _, s := range f.Samples {
		if s == value {
			return
		}
	}
	f.Samples = append(f.Samples, value)
}

// categorical types are worth enumerating; continuous/unique ones are not.
func categorical(k ValueKind) bool {
	switch k {
	case KindString, KindInt, KindBool:
		return true
	default:
		return false
	}
}

// Accumulator holds the model: one writer (Observe), many readers (snapshot/web).
type Accumulator struct {
	mu      sync.RWMutex
	fields  map[string]*Field
	maxEnum int
}

func New() *Accumulator {
	return &Accumulator{fields: map[string]*Field{}, maxEnum: defaultMaxEnum}
}

// Observe folds one raw MessageCollection into the model. Returns a parse error
// on malformed input; never panics.
func (a *Accumulator) Observe(raw []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now().UTC()
	dec := xml.NewDecoder(bytes.NewReader(raw))
	var stack []string

	for {
		tok, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			stack = append(stack, t.Name.Local)
			path := strings.Join(stack, "/")
			a.touch(path, "element", now)

			for _, attr := range t.Attr {
				if attr.Name.Local == "xmlns" || attr.Name.Space == "xmlns" {
					continue // skip namespace declarations
				}
				a.touch(path+"/@"+attr.Name.Local, "attribute", now).addValue(attr.Value, a.maxEnum)
			}

		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text == "" || len(stack) == 0 {
				continue
			}
			// element already touched on StartElement; add text without recounting
			if f := a.fields[strings.Join(stack, "/")]; f != nil {
				f.addValue(text, a.maxEnum)
			}

		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
}

func (a *Accumulator) touch(path, kind string, now time.Time) *Field {
	f := a.fields[path]
	if f == nil {
		f = &Field{Path: path, Kind: kind, FirstSeen: now} // value maps created lazily in addValue
		a.fields[path] = f
	}
	f.Occurrences++
	f.LastSeen = now
	return f
}

// Snapshot is the serialized model, keyed by path (Go sorts map keys in JSON).
type Snapshot struct {
	GeneratedAt time.Time         `json:"generatedAt"`
	Fields      map[string]*Field `json:"fields"`
}

// Snapshot returns the model view (used by tests).
func (a *Accumulator) Snapshot() Snapshot {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return Snapshot{GeneratedAt: time.Now().UTC(), Fields: a.fields}
}

// MarshalSnapshot renders the model as JSON under the read lock (race-free).
func (a *Accumulator) MarshalSnapshot() ([]byte, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return json.MarshalIndent(Snapshot{GeneratedAt: time.Now().UTC(), Fields: a.fields}, "", "  ")
}

// SaveFile writes the model atomically (temp file + rename).
func (a *Accumulator) SaveFile(path string) error {
	data, err := a.MarshalSnapshot()
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadFile restores a prior snapshot; a missing file is not an error.
func (a *Accumulator) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.fields = snap.Fields
	if a.fields == nil {
		a.fields = map[string]*Field{}
	}
	return nil
}

var (
	uuidRe     = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	durationRe = regexp.MustCompile(`^P(\d+Y)?(\d+M)?(\d+D)?(T(\d+H)?(\d+M)?(\d+S)?)?$`)
	geoRe      = regexp.MustCompile(`^-?\d+(\.\d+)?\s+-?\d+(\.\d+)?$`)
)

// classify infers a value's type; order matters (specific patterns before numeric/string).
func classify(s string) ValueKind {
	switch {
	case s == "":
		return KindEmpty
	case uuidRe.MatchString(s):
		return KindUUID
	case isTimestamp(s):
		return KindTimestamp
	case durationRe.MatchString(s) && s != "P":
		return KindDuration
	case geoRe.MatchString(s):
		return KindGeoPair
	case isInt(s):
		return KindInt
	case isFloat(s):
		return KindFloat
	case s == "true" || s == "false":
		return KindBool
	default:
		return KindString
	}
}

func isTimestamp(s string) bool { _, err := time.Parse(time.RFC3339, s); return err == nil }
func isInt(s string) bool       { _, err := strconv.ParseInt(s, 10, 64); return err == nil }
func isFloat(s string) bool     { _, err := strconv.ParseFloat(s, 64); return err == nil }
