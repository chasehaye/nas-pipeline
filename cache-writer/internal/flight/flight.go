package flight

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
)


type Position struct {
	Lat          float64
	Lon          float64
	AltValue     string
	AltUOM       string
	PositionTime string

	Heading    float64
	SpeedKt    float64
	HasHeading bool
}

type Flight struct {
	Gufi         string
	CallSign     string
	Registration string
	Status       string
	Timestamp    string
	Position     Position
	HasPosition  bool
}


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
				TrackVelocity struct {
					X struct {
						Value string `json:"value"`
					} `json:"x"`
					Y struct {
						Value string `json:"value"`
					} `json:"y"`
				} `json:"trackVelocity"`
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

	var best struct {
		time string
		lat  float64
		lon  float64
		alt  string
		uom  string
		vx   string
		vy   string
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
		best.vx, best.vy = p.TrackVelocity.X.Value, p.TrackVelocity.Y.Value
		best.ok = true
	}

	if best.ok {
		f.HasPosition = true
		pos := Position{
			Lat:          best.lat,
			Lon:          best.lon,
			AltValue:     best.alt,
			AltUOM:       best.uom,
			PositionTime: best.time,
		}
		if hdg, spd, ok := velocity(best.vx, best.vy); ok {
			pos.Heading = hdg
			pos.SpeedKt = spd
			pos.HasHeading = true
		}
		f.Position = pos
	}

	return f, nil
}

func velocity(xs, ys string) (heading, speedKt float64, ok bool) {
	x, err := strconv.ParseFloat(strings.TrimSpace(xs), 64)
	if err != nil {
		return 0, 0, false
	}
	y, err := strconv.ParseFloat(strings.TrimSpace(ys), 64)
	if err != nil {
		return 0, 0, false
	}
	if x == 0 && y == 0 {
		return 0, 0, false
	}
	h := math.Atan2(x, y) * 180 / math.Pi
	if h < 0 {
		h += 360
	}
	return math.Round(h*10) / 10, math.Round(math.Hypot(x, y)*10) / 10, true
}

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
