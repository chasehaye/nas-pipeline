package fixm

import "encoding/xml"

// Normalized FIXM model: the lean keep-set.
//
// The FAA FIXM NasFlightType has dozens of subtrees (routes, coordination,
// capabilities, accuracy, and a long tail that keeps growing), but only the
// fields below are consumed downstream (filter, cache-writer, database-writer;
// the api reads only what the writers stored). Everything else is intentionally
// not mapped.
//
// This is safe and low-maintenance: Go's encoding/xml silently ignores any XML
// element not mapped here, so unused or newly-appearing FIXM fields never cause
// an error, they are simply not captured. The model grows only when a consumer
// needs a new field. See docs/data-model.md for the full keep/drop rationale.

type MessageCollection struct {
	XMLName  xml.Name  `xml:"MessageCollection" json:"-"`
	Messages []Message `xml:"message" json:"messages,omitempty"`
}

type Message struct {
	Flight Flight `xml:"flight" json:"flight"`
}

type Flight struct {
	Timestamp string `xml:"timestamp,attr" json:"timestamp"`

	FlightIdentification FlightIdentification `xml:"flightIdentification" json:"flightIdentification"`
	FlightStatus         FlightStatus         `xml:"flightStatus" json:"flightStatus"`
	Gufi                 Gufi                 `xml:"gufi" json:"gufi"`

	Departure Departure `xml:"departure" json:"departure,omitempty"`
	Arrival   Arrival   `xml:"arrival" json:"arrival,omitempty"`

	EnRoute EnRoute `xml:"enRoute" json:"enRoute,omitempty"`

	AircraftDescription AircraftDescription `xml:"aircraftDescription" json:"aircraftDescription,omitempty"`
}

type FlightIdentification struct {
	AircraftIdentification string `xml:"aircraftIdentification,attr" json:"aircraftIdentification"`
}

type FlightStatus struct {
	FDPSFlightStatus string `xml:"fdpsFlightStatus,attr" json:"fdpsFlightStatus"`
}

type Gufi struct {
	Code string `xml:",chardata" json:"code"`
}

// ---------------- Departure / Arrival ----------------

type Departure struct {
	DeparturePoint        string                `xml:"departurePoint,attr" json:"departurePoint,omitempty"`
	RunwayPositionAndTime RunwayPositionAndTime `xml:"runwayPositionAndTime" json:"runwayPositionAndTime,omitempty"`
	DepartureAerodrome    AerodromeLocation     `xml:"departureAerodrome" json:"departureAerodrome,omitempty"`
}

type Arrival struct {
	ArrivalPoint          string                `xml:"arrivalPoint,attr" json:"arrivalPoint,omitempty"`
	RunwayPositionAndTime RunwayPositionAndTime `xml:"runwayPositionAndTime" json:"runwayPositionAndTime,omitempty"`
	ArrivalAerodrome      AerodromeLocation     `xml:"arrivalAerodrome" json:"arrivalAerodrome,omitempty"`
}

// Only the actual (achieved) runway time is consumed; estimated and controlled
// are dropped.
type RunwayPositionAndTime struct {
	RunwayTime RunwayTime `xml:"runwayTime" json:"runwayTime,omitempty"`
}

type RunwayTime struct {
	Actual TimeValue `xml:"actual" json:"actual,omitempty"`
}

type TimeValue struct {
	Time string `xml:"time,attr" json:"time,omitempty"`
}

// AerodromeLocation carries an airport's coordinates, used by departureAerodrome
// and arrivalAerodrome.
type AerodromeLocation struct {
	Point Point `xml:"point" json:"point,omitempty"`
}

type Point struct {
	Location Location `xml:"location" json:"location,omitempty"`
}

// ---------------- En Route ----------------

type EnRoute struct {
	Positions []Position `xml:"position" json:"positions,omitempty"`
}

type Position struct {
	PositionTime string `xml:"positionTime,attr" json:"positionTime,omitempty"`

	Altitude Measurement `xml:"altitude" json:"altitude,omitempty"`

	// The actual position sits under a nested <position> wrapper
	// (position/position/location/pos), not as a direct child, so it needs the
	// path expression. Bound to "location" alone it silently comes back empty
	// while targetPosition (a real direct child) would populate.
	Location Location `xml:"position>location" json:"location"`

	TrackVelocity TrackVelocity `xml:"trackVelocity" json:"trackVelocity,omitempty"`
}

type TrackVelocity struct {
	X Measurement `xml:"x" json:"x,omitempty"`
	Y Measurement `xml:"y" json:"y,omitempty"`
}

type Location struct {
	Pos string `xml:"pos" json:"pos,omitempty"`
}

type Measurement struct {
	Value string `xml:",chardata" json:"value,omitempty"`
	UOM   string `xml:"uom,attr" json:"uom,omitempty"`
}

// ---------------- Aircraft ----------------

type AircraftDescription struct {
	Registration string       `xml:"registration,attr" json:"registration,omitempty"`
	AircraftType AircraftType `xml:"aircraftType" json:"aircraftType,omitempty"`
}

type AircraftType struct {
	ICAOModelIdentifier string `xml:"icaoModelIdentifier" json:"icaoModelIdentifier,omitempty"`
}
