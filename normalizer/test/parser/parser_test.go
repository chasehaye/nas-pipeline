package parser_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/chasehaye/nas-pipeline/processor/internal/fixm"
)

var update = flag.Bool("update", false, "regenerate golden files")

func mustRead(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return b
}

// TestParseEnvelope_Fields asserts the specific field extractions we care about,
// each as a named subtest so a failure names the exact field. The location /
// speed / velocity cases are regression guards for real bugs where nested or
// child-element values were being dropped.
func TestParseEnvelope_Fields(t *testing.T) {
	msgs, err := fixm.ParseEnvelope(mustRead(t, filepath.Join("testdata", "sample_envelope.xml")))
	if err != nil {
		t.Fatalf("ParseEnvelope: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("parsed %d flights, want 2", len(msgs))
	}

	f := msgs[0].Flight
	pos := f.EnRoute.Positions[0]

	cases := []struct{ name, got, want string }{
		{"gufi", f.Gufi.Code, "1d0a7ca1-5fdb-4f28-951b-a3890705fa54"},
		{"callsign", f.FlightIdentification.AircraftIdentification, "N995LA"},
		{"status", f.FlightStatus.FDPSFlightStatus, "ACTIVE"},
		{"altitude.value", pos.Altitude.Value, "8500.0"},
		{"altitude.uom", pos.Altitude.UOM, "FEET"},
		{"location.pos", pos.Location.Pos, "43.915278 -69.638056"},         // nested <position><location>
		{"speed.value", pos.ActualSpeed.Surveillance.Value, "156.0"},       // child <surveillance>
		{"speed.uom", pos.ActualSpeed.Surveillance.UOM, "KNOTS"},
		{"velocity.x", pos.TrackVelocity.X.Value, "-135.0"},                // child <x>
		{"velocity.y", pos.TrackVelocity.Y.Value, "-75.0"},                 // child <y>
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
			}
		})
	}
}

// TestParseEnvelope_Golden locks the full XML->JSON shape against a checked-in
// golden file so any drift (a dropped field, a renamed key, changed nesting)
// shows up as a diff. Regenerate intentionally with:
//
//	go test ./internal/fixm -run Golden -update
func TestParseEnvelope_Golden(t *testing.T) {
	msgs, err := fixm.ParseEnvelope(mustRead(t, filepath.Join("testdata", "sample_envelope.xml")))
	if err != nil {
		t.Fatalf("ParseEnvelope: %v", err)
	}

	got, err := json.MarshalIndent(msgs, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')

	golden := filepath.Join("testdata", "sample_envelope.golden.json")
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("updated %s", golden)
		return
	}

	if want := mustRead(t, golden); !bytes.Equal(got, want) {
		t.Errorf("output drifted from golden — run `go test ./internal/fixm -run Golden -update` if intended")
	}
}

func TestParseEnvelope_Malformed(t *testing.T) {
	if _, err := fixm.ParseEnvelope([]byte("<flight><unclosed>")); err == nil {
		t.Fatal("expected an error on malformed XML, got nil")
	}
}
