// Package fixm parses the FAA's FIXM US Extension 3.0 flight data.
//
// It imports nothing outside the standard library: parsing is a pure
// function over bytes, so it can be tested against captured files with no
// Kafka, Redis or Postgres running.
package fixm

import "encoding/xml"

// ---------- envelope ----------
type MessageCollection struct {
	XMLName  xml.Name  `xml:"MessageCollection"`
	Messages []Message `xml:"message"`
}

// Message is a thin wrapper. It carries no data of its own.
type Message struct {
	Flight Flight `xml:"flight"`
}

// ---------- flight ----------

type Flight struct {
	Source     string `xml:"source,attr"`
	Centre     string `xml:"centre,attr"`
	System     string `xml:"system,attr"`
	Timestamp  string `xml:"timestamp,attr"`
	FlightType string `xml:"flightType,attr"`

	GUFI           string               `xml:"gufi"`
	Identification Identification       `xml:"flightIdentification"`
	Status         Status               `xml:"flightStatus"`
	Arrival        *Endpoint            `xml:"arrival"`
	Departure      *Endpoint            `xml:"departure"`
	EnRoute        *EnRoute             `xml:"enRoute"`
	ControllingIn  *Unit                `xml:"controllingUnit"`
	Operator       *Operator            `xml:"operator"`
	Aircraft       *AircraftDescription `xml:"aircraftDescription"`
	Supplemental   *Supplemental        `xml:"supplementalData"`
}

type Identification struct {
	AircraftID string `xml:"aircraftIdentification,attr"`
	ComputerID string `xml:"computerId,attr"`
}

type Status struct {
	FdpsStatus string `xml:"fdpsFlightStatus,attr"`
}

// Endpoint covers both arrival and departure. The point codes are free text
// by design: ERAM cannot distinguish an airport from a fix or a
// fix-radial-distance, so values like SAF003045 appear alongside KPHX.
type Endpoint struct {
	ArrivalPoint   string `xml:"arrivalPoint,attr"`
	DeparturePoint string `xml:"departurePoint,attr"`
}

type Unit struct {
	UnitIdentifier string `xml:"unitIdentifier,attr"`
	// "00" means unknown, not sector zero.
	SectorIdentifier string `xml:"sectorIdentifier,attr"`
}

// ---------- position ----------
type EnRoute struct {
	Position *Position `xml:"position"`
}

type Position struct {
	// Optional: absent on roughly 2% of position messages. Fall back to the
	// flight-level Timestamp when empty.
	PositionTime       string `xml:"positionTime,attr"`
	TargetPositionTime string `xml:"targetPositionTime,attr"`
	ReportSource       string `xml:"reportSource,attr"`

	// "C" means the track is being extrapolated because no radar return
	// arrived. A coasting position is a guess, not an observation.
	CoastIndicator string `xml:"coastIndicator,attr"`

	// Zero means no altitude available, not sea level. Whether the value is
	// MSL or a flight level is genuinely ambiguous in this feed.
	Altitude    *Measure  `xml:"altitude"`
	ActualSpeed *Speed    `xml:"actualSpeed"`
	Location    *Location `xml:"position"`
}

// Location holds the coordinate pair as a single space-separated string,
// e.g. "35.375278 -100.479444". Needs a manual split.
type Location struct {
	Pos string `xml:"location>pos"`
}

type Measure struct {
	UOM   string  `xml:"uom,attr"`
	Value float64 `xml:",chardata"`
}

type Speed struct {
	Surveillance *Measure `xml:"surveillance"`
}

// ---------- operator and aircraft ----------

// Operator needs this nesting because encoding/xml rejects a ">" path chain
// combined with the attr flag. Absent on ~24% of flights: general aviation
// has no operating organization.
type Operator struct {
	OperatingOrganization OperatingOrganization `xml:"operatingOrganization"`
}

type OperatingOrganization struct {
	Organization Organization `xml:"organization"`
}

type Organization struct {
	Name string `xml:"name,attr"`
}

// AircraftDescription appears only on flight plan messages, so aircraft
// identity arrives separately from position data and must be merged rather
// than overwritten.
type AircraftDescription struct {
	Address      string `xml:"aircraftAddress,attr"`
	Registration string `xml:"registration,attr"`
	Wake         string `xml:"wakeTurbulence,attr"`
	Model        string `xml:"aircraftType>icaoModelIdentifier"`
}

// ---------- supplemental ----------

type Supplemental struct {
	Values []NameValue `xml:"additionalFlightInformation>nameValue"`
}

type NameValue struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// Lookup flattens the nameValue list. This is where FIXM hides MSG_SEQ_NO
// and FDPS_GUFI, the latter being the flight identifier that survives an
// SFDPS failover when the UUID GUFI does not.
func (s *Supplemental) Lookup(key string) string {
	if s == nil {
		return ""
	}
	for _, nv := range s.Values {
		if nv.Name == key {
			return nv.Value
		}
	}
	return ""
}