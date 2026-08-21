package flight

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

// Coord is opportunistically-parsed (rarely present).
type Coord struct {
	Lat float64
	Lon float64
	OK  bool
}

type Position struct {
	Lat        float64
	Lon        float64
	Alt        float64
	HasAlt     bool
	Heading    float64
	SpeedKt    float64
	HasHeading bool
	Time       string
}

type Flight struct {
	Gufi         string
	CallSign     string
	Registration string
	AircraftType string
	Origin       string
	Destination  string
	Status       string
	Timestamp    string

	ActualDepartureTime string
	ActualArrivalTime   string

	OriginCoord      Coord
	DestinationCoord Coord

	Position    Position
	HasPosition bool
}

type runwayTime struct {
	RunwayTime struct {
		Actual struct {
			Time string `json:"time"`
		} `json:"actual"`
	} `json:"runwayTime"`
}

type aerodrome struct {
	Point struct {
		Location struct {
			Pos string `json:"pos"`
		} `json:"location"`
	} `json:"point"`
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
			AircraftType struct {
				IcaoModelIdentifier string `json:"icaoModelIdentifier"`
			} `json:"aircraftType"`
		} `json:"aircraftDescription"`
		Departure struct {
			DeparturePoint        string     `json:"departurePoint"`
			RunwayPositionAndTime runwayTime `json:"runwayPositionAndTime"`
			DepartureAerodrome    aerodrome  `json:"departureAerodrome"`
		} `json:"departure"`
		Arrival struct {
			ArrivalPoint          string     `json:"arrivalPoint"`
			RunwayPositionAndTime runwayTime `json:"runwayPositionAndTime"`
			ArrivalAerodrome      aerodrome  `json:"arrivalAerodrome"`
		} `json:"arrival"`
		EnRoute struct {
			Positions []struct {
				PositionTime string `json:"positionTime"`
				Altitude     struct {
					Value string `json:"value"`
					UOM   string `json:"uom"`
				} `json:"altitude"`
				Location struct {
					Pos string `json:"pos"`
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
	fl := e.Flight

	f := Flight{
		Gufi:                fl.Gufi.Code,
		CallSign:            fl.FlightIdentification.AircraftIdentification,
		Registration:        fl.AircraftDescription.Registration,
		AircraftType:        fl.AircraftDescription.AircraftType.IcaoModelIdentifier,
		Origin:              fl.Departure.DeparturePoint,
		Destination:         fl.Arrival.ArrivalPoint,
		Status:              fl.FlightStatus.FDPSFlightStatus,
		Timestamp:           fl.Timestamp,
		ActualDepartureTime: fl.Departure.RunwayPositionAndTime.RunwayTime.Actual.Time,
		ActualArrivalTime:   fl.Arrival.RunwayPositionAndTime.RunwayTime.Actual.Time,
	}

	if lat, lon, ok := parsePos(fl.Departure.DepartureAerodrome.Point.Location.Pos); ok {
		f.OriginCoord = Coord{Lat: lat, Lon: lon, OK: true}
	}
	if lat, lon, ok := parsePos(fl.Arrival.ArrivalAerodrome.Point.Location.Pos); ok {
		f.DestinationCoord = Coord{Lat: lat, Lon: lon, OK: true}
	}

	var best struct {
		time string
		lat  float64
		lon  float64
		alt  string
		vx   string
		vy   string
		ok   bool
	}
	for _, p := range fl.EnRoute.Positions {
		lat, lon, ok := parsePos(p.Location.Pos)
		if !ok {
			continue
		}
		if best.ok && p.PositionTime < best.time {
			continue
		}
		best.time = p.PositionTime
		best.lat, best.lon = lat, lon
		best.alt = p.Altitude.Value
		best.vx, best.vy = p.TrackVelocity.X.Value, p.TrackVelocity.Y.Value
		best.ok = true
	}

	if best.ok {
		f.HasPosition = true
		pos := Position{Lat: best.lat, Lon: best.lon, Time: best.time}
		if alt, err := strconv.ParseFloat(strings.TrimSpace(best.alt), 64); err == nil {
			pos.Alt = alt
			pos.HasAlt = true
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
