package flight

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Position is the current location of a flight, pulled from the latest
// enRoute position report in a FIXM message.
type Position struct {
	Lat          float64
	Lon          float64
	AltValue     string
	AltUOM       string
	PositionTime string
}

// Flight is the slim projection we store per GUFI. It intentionally mirrors
// only the fields the active-flights view needs, not the full FIXM model.
type Flight struct {
	Gufi         string
	CallSign     string
	Registration string
	Status       string
	Timestamp    string
	Position     Position
	HasPosition  bool
}

// element mirrors just the subset of the fixm.Flight JSON we read. Field names
// track the json tags in processor/internal/fixm/models.go.
type element struct {
	Flight struct {
		Timestamp            string `json:"timestamp"`
		FlightIdentification struct {
			AircraftIdentification string `json:"aircraftIdentification"`
		} `json:"flightIdentification"`
		FlightStatus struct {
			FDPSFlightStatus string `json:"fdpsFlightStatus"`
		} `json:"flightStatus"`
		Gufi struct {
			Code string `json:"code"`
		} `json:"gufi"`
		AircraftDescription struct {
			Registration string `json:"registration"`
		} `json:"aircraftDescription"`
		EnRoute struct {
			Positions []struct {
				PositionTime string `json:"positionTime"`
				Altitude     struct {
					Value string `json:"value"`
					UOM   string `json:"uom"`
				} `json:"altitude"`
				Location struct {
					SRSName string `json:"srsName"`
					Pos     string `json:"pos"`
				} `json:"location"`
			} `json:"positions"`
		} `json:"enRoute"`
	} `json:"flight"`
}

func Parse(payload []byte) (Flight, error) {
	var e element
	if err := json.Unmarshal(payload, &e); err != nil {
		return Flight{}, err
	}

	f := Flight{
		Gufi:         e.Flight.Gufi.Code,
		CallSign:     e.Flight.FlightIdentification.AircraftIdentification,
		Registration: e.Flight.AircraftDescription.Registration,
		Status:       e.Flight.FlightStatus.FDPSFlightStatus,
		Timestamp:    e.Flight.Timestamp,
	}

	// A message can carry several position reports; take the most recent one
	// that actually has a location. positionTime is RFC3339, so lexical
	// comparison orders it correctly.
	var best struct {
		time string
		lat  float64
		lon  float64
		alt  string
		uom  string
		ok   bool
	}
	for _, p := range e.Flight.EnRoute.Positions {
		lat, lon, ok := parsePos(p.Location.Pos)
		if !ok {
			continue
		}
		if best.ok && p.PositionTime < best.time {
			continue
		}
		best.time = p.PositionTime
		best.lat, best.lon = lat, lon
		best.alt, best.uom = p.Altitude.Value, p.Altitude.UOM
		best.ok = true
	}

	if best.ok {
		f.HasPosition = true
		f.Position = Position{
			Lat:          best.lat,
			Lon:          best.lon,
			AltValue:     best.alt,
			AltUOM:       best.uom,
			PositionTime: best.time,
		}
	}

	return f, nil
}

// parsePos reads a GML pos string. FIXM positions use EPSG:4326, which is
// latitude-then-longitude order, e.g. "40.63980 -73.77890".
func parsePos(pos string) (lat, lon float64, ok bool) {
	fields := strings.Fields(pos)
	if len(fields) < 2 {
		return 0, 0, false
	}
	lat, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, 0, false
	}
	lon, err = strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return 0, 0, false
	}
	return lat, lon, true
}
