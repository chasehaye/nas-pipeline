package fixm

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Crafted fixtures are minimal and namespace-free (encoding/xml matches by local
// name), so we can assert exact values on fully-controlled input.

// One complete flight covering every keep-set field, incl. the nested
// position/position/location/pos path.
const fullFlight = `<MessageCollection>
  <message>
    <flight timestamp="2026-10-09T19:36:49.262Z">
      <flightIdentification aircraftIdentification="N683SL"/>
      <flightStatus fdpsFlightStatus="ACTIVE"/>
      <gufi>845aa726-48dc-44ed-a227-1629f135e057</gufi>
      <departure departurePoint="KMMU">
        <runwayPositionAndTime><runwayTime><actual time="2026-10-09T19:08:00Z"/></runwayTime></runwayPositionAndTime>
      </departure>
      <arrival arrivalPoint="KFOK">
        <runwayPositionAndTime><runwayTime><actual time="2026-10-09T19:36:00Z"/></runwayTime></runwayPositionAndTime>
      </arrival>
      <enRoute>
        <position positionTime="2026-10-09T19:36:40Z">
          <altitude uom="FEET">1500.0</altitude>
          <position>
            <location><pos>40.906944 -72.553611</pos></location>
          </position>
          <trackVelocity>
            <x uom="KNOTS">-108.0</x>
            <y uom="KNOTS">-111.0</y>
          </trackVelocity>
        </position>
      </enRoute>
      <aircraftDescription registration="N683SL">
        <aircraftType><icaoModelIdentifier>M20P</icaoModelIdentifier></aircraftType>
      </aircraftDescription>
    </flight>
  </message>
</MessageCollection>`

// Proposed flight, no position (~12% of the feed).
const positionlessFlight = `<MessageCollection>
  <message>
    <flight timestamp="2026-10-09T19:36:49.262Z">
      <flightIdentification aircraftIdentification="AAL123"/>
      <flightStatus fdpsFlightStatus="PROPOSED"/>
      <gufi>00000000-0000-0000-0000-000000000001</gufi>
    </flight>
  </message>
</MessageCollection>`

// Airline call sign, no aircraftDescription, so empty registration.
const noRegistrationFlight = `<MessageCollection>
  <message>
    <flight timestamp="2026-10-09T19:36:49.262Z">
      <flightIdentification aircraftIdentification="KAP519"/>
      <flightStatus fdpsFlightStatus="ACTIVE"/>
      <gufi>12e6fbea-e427-4da4-a183-06ac8920000d</gufi>
    </flight>
  </message>
</MessageCollection>`

const twoFlights = `<MessageCollection>
  <message><flight timestamp="t"><flightIdentification aircraftIdentification="A"/><flightStatus fdpsFlightStatus="ACTIVE"/><gufi>g1</gufi></flight></message>
  <message><flight timestamp="t"><flightIdentification aircraftIdentification="B"/><flightStatus fdpsFlightStatus="ACTIVE"/><gufi>g2</gufi></flight></message>
</MessageCollection>`

func eq(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", field, got, want)
	}
}

func TestParseEnvelope_FullFlight(t *testing.T) {
	msgs, err := ParseEnvelope([]byte(fullFlight))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("got %d flights, want 1", len(msgs))
	}
	f := msgs[0].Flight

	eq(t, "timestamp", f.Timestamp, "2026-10-09T19:36:49.262Z")
	eq(t, "aircraftIdentification", f.FlightIdentification.AircraftIdentification, "N683SL")
	eq(t, "fdpsFlightStatus", f.FlightStatus.FDPSFlightStatus, "ACTIVE")
	eq(t, "gufi", f.Gufi.Code, "845aa726-48dc-44ed-a227-1629f135e057")

	eq(t, "departurePoint", f.Departure.DeparturePoint, "KMMU")
	eq(t, "departure actual time", f.Departure.RunwayPositionAndTime.RunwayTime.Actual.Time, "2026-10-09T19:08:00Z")
	eq(t, "arrivalPoint", f.Arrival.ArrivalPoint, "KFOK")
	eq(t, "arrival actual time", f.Arrival.RunwayPositionAndTime.RunwayTime.Actual.Time, "2026-10-09T19:36:00Z")

	eq(t, "registration", f.AircraftDescription.Registration, "N683SL")
	eq(t, "icaoModelIdentifier", f.AircraftDescription.AircraftType.ICAOModelIdentifier, "M20P")

	if len(f.EnRoute.Positions) != 1 {
		t.Fatalf("got %d positions, want 1", len(f.EnRoute.Positions))
	}
	p := f.EnRoute.Positions[0]
	eq(t, "positionTime", p.PositionTime, "2026-10-09T19:36:40Z")
	eq(t, "altitude value", p.Altitude.Value, "1500.0")
	eq(t, "altitude uom", p.Altitude.UOM, "FEET")
	// Nested position/position/location/pos path.
	eq(t, "location pos", p.Location.Pos, "40.906944 -72.553611")
	eq(t, "trackVelocity x", p.TrackVelocity.X.Value, "-108.0")
	eq(t, "trackVelocity y", p.TrackVelocity.Y.Value, "-111.0")
}

func TestParseEnvelope_Positionless(t *testing.T) {
	msgs, err := ParseEnvelope([]byte(positionlessFlight))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("got %d flights, want 1", len(msgs))
	}
	f := msgs[0].Flight
	eq(t, "fdpsFlightStatus", f.FlightStatus.FDPSFlightStatus, "PROPOSED")
	if len(f.EnRoute.Positions) != 0 {
		t.Errorf("got %d positions, want 0 for a positionless flight", len(f.EnRoute.Positions))
	}
}

func TestParseEnvelope_NoRegistration(t *testing.T) {
	msgs, err := ParseEnvelope([]byte(noRegistrationFlight))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f := msgs[0].Flight
	eq(t, "aircraftIdentification", f.FlightIdentification.AircraftIdentification, "KAP519")
	eq(t, "registration", f.AircraftDescription.Registration, "") // absent, not an error
}

func TestParseEnvelope_MultipleFlights(t *testing.T) {
	msgs, err := ParseEnvelope([]byte(twoFlights))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("got %d flights, want 2", len(msgs))
	}
	eq(t, "flight 0 gufi", msgs[0].Flight.Gufi.Code, "g1")
	eq(t, "flight 1 gufi", msgs[1].Flight.Gufi.Code, "g2")
}

func TestParseEnvelope_Malformed(t *testing.T) {
	_, err := ParseEnvelope([]byte(`<MessageCollection><message><flight timestamp="t"`))
	if err == nil {
		t.Fatal("expected an error for malformed XML, got nil")
	}
}

// Real captured record (49 flights): parses cleanly and every flight has a GUFI.
func TestParseEnvelope_RealSample(t *testing.T) {
	data, err := os.ReadFile("testdata/sample.xml")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	msgs, err := ParseEnvelope(data)
	if err != nil {
		t.Fatalf("real sample failed to parse: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("no flights parsed from the real sample")
	}
	for i, m := range msgs {
		if strings.TrimSpace(m.Flight.Gufi.Code) == "" {
			t.Errorf("flight %d has an empty gufi", i)
		}
	}
}

// EncodeOne must produce the JSON key paths the writers unmarshal; a drifted
// json tag makes them read zero values silently.
func TestEncodeOne_JSONContract(t *testing.T) {
	msgs, err := ParseEnvelope([]byte(fullFlight))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	data, err := EncodeOne(msgs[0])
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	// Mirror of the shape the writers parse, so we assert against the real
	// contract, not just "some JSON came out".
	var out struct {
		Flight struct {
			Gufi                 struct{ Code string `json:"code"` }                              `json:"gufi"`
			FlightIdentification struct{ AircraftIdentification string `json:"aircraftIdentification"` } `json:"flightIdentification"`
			FlightStatus         struct{ FDPSFlightStatus string `json:"fdpsFlightStatus"` }      `json:"flightStatus"`
			EnRoute              struct {
				Positions []struct {
					Altitude struct {
						Value string `json:"value"`
						UOM   string `json:"uom"`
					} `json:"altitude"`
					Location struct {
						Pos string `json:"pos"`
					} `json:"location"`
				} `json:"positions"`
			} `json:"enRoute"`
		} `json:"flight"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("encoded JSON did not unmarshal to the downstream shape: %v\njson: %s", err, data)
	}

	eq(t, "json gufi.code", out.Flight.Gufi.Code, "845aa726-48dc-44ed-a227-1629f135e057")
	eq(t, "json aircraftIdentification", out.Flight.FlightIdentification.AircraftIdentification, "N683SL")
	eq(t, "json fdpsFlightStatus", out.Flight.FlightStatus.FDPSFlightStatus, "ACTIVE")
	if len(out.Flight.EnRoute.Positions) != 1 {
		t.Fatalf("json positions = %d, want 1", len(out.Flight.EnRoute.Positions))
	}
	eq(t, "json location.pos", out.Flight.EnRoute.Positions[0].Location.Pos, "40.906944 -72.553611")
	eq(t, "json altitude.value", out.Flight.EnRoute.Positions[0].Altitude.Value, "1500.0")
}
