package parser_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/chasehaye/nas-pipeline/processor/internal/fixm"
)

func loadCollection(t *testing.T) []fixm.Message {
	t.Helper()
	msgs, err := fixm.ParseEnvelope(mustRead(t, filepath.Join("testdata", "sample_collection.xml")))
	if err != nil {
		t.Fatalf("ParseEnvelope: %v", err)
	}
	return msgs
}

func findByCallsign(t *testing.T, msgs []fixm.Message, callsign string) fixm.Flight {
	t.Helper()
	for _, m := range msgs {
		if m.Flight.FlightIdentification.AircraftIdentification == callsign {
			return m.Flight
		}
	}
	t.Fatalf("no flight with callsign %q in collection", callsign)
	return fixm.Flight{}
}

func afiMap(f fixm.Flight) map[string]string {
	m := make(map[string]string)
	for _, nv := range f.SupplementalData.AdditionalFlightInformation.NameValue {
		m[nv.Name] = nv.Value
	}
	return m
}

func TestParseCollection_Count(t *testing.T) {
	if got := len(loadCollection(t)); got != 33 {
		t.Fatalf("parsed %d flights, want 33", got)
	}
}

func TestParseCollection_FirstFlight(t *testing.T) {
	f := loadCollection(t)[0].Flight
	pos := f.EnRoute.Positions[0]

	cases := []struct{ name, got, want string }{
		{"callsign", f.FlightIdentification.AircraftIdentification, "JIA5201"},
		{"computerId", f.FlightIdentification.ComputerID, "705"},
		{"status", f.FlightStatus.FDPSFlightStatus, "DROPPED"},
		{"gufi", f.Gufi.Code, "78b5ad2f-3733-47db-94db-52d83c89ab55"},
		{"gufi.codeSpace", f.Gufi.CodeSpace, "urn:uuid"},
		{"centre", f.Centre, "ZTL"},
		{"arrival.point", f.Arrival.ArrivalPoint, "KVPS"},
		{"arrival.estimated", f.Arrival.RunwayPositionAndTime.RunwayTime.Estimated.Time, "2026-08-05T22:00:00Z"},
		{"departure.point", f.Departure.DeparturePoint, "KCLT"},
		{"departure.actual", f.Departure.RunwayPositionAndTime.RunwayTime.Actual.Time, "2026-08-05T20:56:00Z"},
		{"controllingUnit.id", f.ControllingUnit.UnitIdentifier, "ZTL"},
		{"controllingUnit.sector", f.ControllingUnit.SectorIdentifier, "20"},
		{"operator", f.Operator.OperatingOrganization.Organization.Name, "JIA"},
		{"assignedAltitude.simple", f.AssignedAltitude.Simple.Value, "34000.0"},
		{"flightPlan.id", f.FlightPlan.Identifier, "KT706954K4"},
		{"altitude.value", pos.Altitude.Value, "29300.0"},
		{"altitude.uom", pos.Altitude.UOM, "FEET"},
		{"location.pos", pos.Location.Pos, "33.48 -82.174722"},
		{"speed.value", pos.ActualSpeed.Surveillance.Value, "416.0"},
		{"velocity.x", pos.TrackVelocity.X.Value, "-330.0"},
		{"velocity.y", pos.TrackVelocity.Y.Value, "-253.0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
			}
		})
	}
}


func TestParseCollection_VFRPlusAltitude(t *testing.T) {
	f := findByCallsign(t, loadCollection(t), "N2417L")

	if got, want := f.AssignedAltitude.VFRPlus.Value, "2500.0"; got != want {
		t.Errorf("vfrPlus.value = %q, want %q", got, want)
	}
	if got, want := f.AssignedAltitude.VFRPlus.UOM, "FEET"; got != want {
		t.Errorf("vfrPlus.uom = %q, want %q", got, want)
	}
	if got := f.AssignedAltitude.Simple.Value; got != "" {
		t.Errorf("simple.value = %q, want empty for a vfrPlus flight", got)
	}
}

func TestParseCollection_BoundaryHandoff(t *testing.T) {
	f := findByCallsign(t, loadCollection(t), "AAL1224")

	handoffs := f.EnRoute.BoundaryCrossings.Handoff
	if len(handoffs) != 1 {
		t.Fatalf("got %d handoffs, want 1", len(handoffs))
	}
	ru := handoffs[0].ReceivingUnit
	if got, want := ru.UnitIdentifier, "ZTL"; got != want {
		t.Errorf("receivingUnit.id = %q, want %q", got, want)
	}
	if got, want := ru.SectorIdentifier, "29"; got != want {
		t.Errorf("receivingUnit.sector = %q, want %q", got, want)
	}
}

func TestParseCollection_OptionalElementsAbsent(t *testing.T) {
	f := findByCallsign(t, loadCollection(t), "N2417L")

	if got := f.Operator.OperatingOrganization.Organization.Name; got != "" {
		t.Errorf("operator name = %q, want empty (flight has no <operator>)", got)
	}
	if got := f.Arrival.RunwayPositionAndTime.RunwayTime.Estimated.Time; got != "" {
		t.Errorf("arrival estimated time = %q, want empty (no runwayPositionAndTime)", got)
	}
	if got, want := f.Arrival.ArrivalPoint, "KSA1"; got != want {
		t.Errorf("arrival point = %q, want %q", got, want)
	}
}

func TestParseCollection_SupplementalNameValues(t *testing.T) {
	f := loadCollection(t)[0].Flight
	afi := afiMap(f)

	want := map[string]string{
		"MSG_SEQ_NO":  "81188493",
		"FDPS_GUFI":   "us.fdps.2026-08-05T19:38:15Z.000/19/4K4",
		"SOURCE_TIME": "21_18_12",
		"ADSB_02M_52B": "-A86A2B",
	}
	for name, wantVal := range want {
		if got := afi[name]; got != wantVal {
			t.Errorf("nameValue[%q] = %q, want %q", name, got, wantVal)
		}
	}
}

func TestParseCollection_Invariants(t *testing.T) {
	for i, m := range loadCollection(t) {
		f := m.Flight
		if f.Gufi.Code == "" {
			t.Errorf("flight[%d]: empty gufi", i)
		}
		if f.Gufi.CodeSpace != "urn:uuid" {
			t.Errorf("flight[%d]: gufi.codeSpace = %q, want urn:uuid", i, f.Gufi.CodeSpace)
		}
		if f.FlightIdentification.AircraftIdentification == "" {
			t.Errorf("flight[%d]: empty callsign", i)
		}
		if f.Centre != "ZTL" {
			t.Errorf("flight[%d]: centre = %q, want ZTL", i, f.Centre)
		}
		if len(f.EnRoute.Positions) == 0 {
			t.Errorf("flight[%d] (%s): no positions", i, f.FlightIdentification.AircraftIdentification)
			continue
		}
		if f.EnRoute.Positions[0].Location.Pos == "" {
			t.Errorf("flight[%d] (%s): empty location pos", i, f.FlightIdentification.AircraftIdentification)
		}
	}
}

func TestParseCollection_Golden(t *testing.T) {
	msgs := loadCollection(t)

	got, err := json.MarshalIndent(msgs, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')

	golden := filepath.Join("testdata", "sample_collection.golden.json")
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("updated %s", golden)
		return
	}

	if want := mustRead(t, golden); !bytes.Equal(got, want) {
		t.Errorf("output drifted from golden — run `go test ./test/parser -run Collection_Golden -update` if intended")
	}
}
