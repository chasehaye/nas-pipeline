package accumulator

import (
	"path/filepath"
	"testing"
)

const sampleMsg = `<MessageCollection>
  <message>
    <flight timestamp="2026-10-09T19:36:49Z">
      <flightStatus fdpsFlightStatus="ACTIVE"/>
      <gufi>845aa726-48dc-44ed-a227-1629f135e057</gufi>
      <altitude uom="FEET">6000.0</altitude>
    </flight>
  </message>
</MessageCollection>`

// Observe records paths with the right kind, inferred type, and values.
func TestObserve_RecordsShape(t *testing.T) {
	a := New()
	if err := a.Observe([]byte(sampleMsg)); err != nil {
		t.Fatal(err)
	}
	f := a.Snapshot().Fields

	if got := f["MessageCollection/message/flight/@timestamp"]; got == nil || got.Types[KindTimestamp] == 0 {
		t.Errorf("timestamp attr not classified as timestamp: %+v", got)
	}
	if got := f["MessageCollection/message/flight/gufi"]; got == nil || got.Types[KindUUID] == 0 {
		t.Errorf("gufi text not classified as uuid: %+v", got)
	}
	if got := f["MessageCollection/message/flight/altitude"]; got == nil || got.Types[KindFloat] == 0 {
		t.Errorf("altitude text not classified as float: %+v", got)
	}
	if got := f["MessageCollection/message/flight/altitude/@uom"]; got == nil || got.Values["FEET"] == 0 {
		t.Errorf("uom value FEET not recorded: %+v", got)
	}
	if got := f["MessageCollection/message/flight/flightStatus/@fdpsFlightStatus"]; got == nil || got.Kind != "attribute" {
		t.Errorf("fdpsFlightStatus not recorded as an attribute: %+v", got)
	}
}

// Only categorical fields enumerate values; continuous/unique ones don't.
func TestObserve_OnlyEnumeratesCategorical(t *testing.T) {
	a := New()
	if err := a.Observe([]byte(sampleMsg)); err != nil {
		t.Fatal(err)
	}
	f := a.Snapshot().Fields

	// Categorical string: values enumerated.
	if got := f["MessageCollection/message/flight/altitude/@uom"]; len(got.Values) == 0 {
		t.Errorf("categorical @uom should enumerate values, got none")
	}
	// Continuous/unique: no value set.
	for _, p := range []string{
		"MessageCollection/message/flight/@timestamp", // timestamp
		"MessageCollection/message/flight/gufi",       // uuid
		"MessageCollection/message/flight/altitude",   // float
	} {
		got := f[p]
		if len(got.Values) != 0 {
			t.Errorf("%s (continuous) should not enumerate values, got %v", p, got.Values)
		}
		if len(got.Samples) == 0 {
			t.Errorf("%s should still keep example samples", p)
		}
	}
}

// SaveFile then LoadFile into a fresh accumulator reproduces the model.
func TestSaveLoad_RoundTrip(t *testing.T) {
	a := New()
	if err := a.Observe([]byte(sampleMsg)); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "census.json")
	if err := a.SaveFile(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	b := New()
	if err := b.LoadFile(path); err != nil {
		t.Fatalf("load: %v", err)
	}

	want, got := a.Snapshot().Fields, b.Snapshot().Fields
	if len(got) != len(want) {
		t.Fatalf("loaded %d fields, want %d", len(got), len(want))
	}
	if got["MessageCollection/message/flight/gufi"] == nil {
		t.Error("gufi field missing after round-trip")
	}
}

// A missing snapshot file on load is not an error (first run).
func TestLoadFile_MissingIsOK(t *testing.T) {
	if err := New().LoadFile(filepath.Join(t.TempDir(), "absent.json")); err != nil {
		t.Errorf("missing file should load cleanly, got %v", err)
	}
}

func TestClassify(t *testing.T) {
	cases := map[string]ValueKind{
		"845aa726-48dc-44ed-a227-1629f135e057": KindUUID,
		"2026-10-09T19:36:49Z":                 KindTimestamp,
		"2026-10-09T19:36:00.000Z":             KindTimestamp,
		"P0Y0M0DT0H3M0S":                       KindDuration,
		"41.161111 -73.08":                     KindGeoPair,
		"6000":                                 KindInt,
		"6000.0":                               KindFloat,
		"true":                                 KindBool,
		"ACTIVE":                               KindString,
		"N683SL":                               KindString,
		"":                                     KindEmpty,
	}
	for in, want := range cases {
		if got := classify(in); got != want {
			t.Errorf("classify(%q) = %q, want %q", in, got, want)
		}
	}
}
