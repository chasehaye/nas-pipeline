package fixm

import "encoding/xml"

type MessageCollection struct {
	XMLName  xml.Name  `xml:"MessageCollection" json:"-"`
	Messages []Message `xml:"message" json:"messages,omitempty"`
}

type Message struct {
	Flight Flight `xml:"flight" json:"flight"`
}

type Flight struct {
	Timestamp  string `xml:"timestamp,attr" json:"timestamp"`
	Centre     string `xml:"centre,attr" json:"centre"`
	Source     string `xml:"source,attr" json:"source"`
	System     string `xml:"system,attr" json:"system"`
	FlightType string `xml:"flightType,attr" json:"flightType,omitempty"`

	FlightIdentification FlightIdentification `xml:"flightIdentification" json:"flightIdentification"`
	FlightStatus         FlightStatus         `xml:"flightStatus" json:"flightStatus"`

	Gufi Gufi `xml:"gufi" json:"gufi"`

	Departure Departure `xml:"departure" json:"departure,omitempty"`
	Arrival   Arrival   `xml:"arrival" json:"arrival,omitempty"`

	EnRoute EnRoute `xml:"enRoute" json:"enRoute,omitempty"`

	Operator Operator `xml:"operator" json:"operator,omitempty"`

	SupplementalData SupplementalData `xml:"supplementalData" json:"supplementalData,omitempty"`

	Coordination Coordination `xml:"coordination" json:"coordination,omitempty"`

	FlightPlan FlightPlan `xml:"flightPlan" json:"flightPlan,omitempty"`

	ControllingUnit ControllingUnit `xml:"controllingUnit" json:"controllingUnit,omitempty"`

	AssignedAltitude AssignedAltitude `xml:"assignedAltitude" json:"assignedAltitude,omitempty"`

	Agreed Agreed `xml:"agreed" json:"agreed,omitempty"`

	AircraftDescription AircraftDescription `xml:"aircraftDescription" json:"aircraftDescription,omitempty"`

	RequestedAirspeed RequestedAirspeed `xml:"requestedAirspeed" json:"requestedAirspeed,omitempty"`

	Originator Originator `xml:"originator" json:"originator,omitempty"`

	InterimAltitude Measurement `xml:"interimAltitude" json:"interimAltitude,omitempty"`

	RequestedAltitude RequestedAltitude `xml:"requestedAltitude" json:"requestedAltitude,omitempty"`

	SpecialHandling string `xml:"specialHandling" json:"specialHandling,omitempty"`
}

type FlightIdentification struct {
	ComputerID             string `xml:"computerId,attr" json:"computerId,omitempty"`
	AircraftIdentification string `xml:"aircraftIdentification,attr" json:"aircraftIdentification"`
	SiteSpecificPlanID     string `xml:"siteSpecificPlanId,attr" json:"siteSpecificPlanId,omitempty"`
}

type FlightStatus struct {
	FDPSFlightStatus string `xml:"fdpsFlightStatus,attr" json:"fdpsFlightStatus"`
	AirborneHold     string `xml:"airborneHold,attr" json:"airborneHold,omitempty"`
}

type Gufi struct {
	Code      string `xml:",chardata" json:"code"`
	CodeSpace string `xml:"codeSpace,attr" json:"codeSpace,omitempty"`
}

// ---------------- Departure / Arrival ----------------

type Departure struct {
	DeparturePoint string `xml:"departurePoint,attr" json:"departurePoint,omitempty"`

	RunwayPositionAndTime RunwayPositionAndTime `xml:"runwayPositionAndTime" json:"runwayPositionAndTime,omitempty"`
}

type Arrival struct {
	ArrivalPoint string `xml:"arrivalPoint,attr" json:"arrivalPoint,omitempty"`

	RunwayPositionAndTime RunwayPositionAndTime `xml:"runwayPositionAndTime" json:"runwayPositionAndTime,omitempty"`
}

type RunwayPositionAndTime struct {
	RunwayTime RunwayTime `xml:"runwayTime" json:"runwayTime,omitempty"`
}

type RunwayTime struct {
	Actual     TimeValue `xml:"actual" json:"actual,omitempty"`
	Estimated  TimeValue `xml:"estimated" json:"estimated,omitempty"`
	Controlled TimeValue `xml:"controlled" json:"controlled,omitempty"`
}

type TimeValue struct {
	Time string `xml:"time,attr" json:"time,omitempty"`
}

// ---------------- En Route ----------------

type EnRoute struct {
	Positions          []Position           `xml:"position" json:"positions,omitempty"`
	BoundaryCrossings  BoundaryCrossings    `xml:"boundaryCrossings" json:"boundaryCrossings,omitempty"`
	AlternateAerodrome []AlternateAerodrome `xml:"alternateAerodrome" json:"alternateAerodrome,omitempty"`
}

type Position struct {
	PositionTime       string `xml:"positionTime,attr" json:"positionTime,omitempty"`
	TargetPositionTime string `xml:"targetPositionTime,attr" json:"targetPositionTime,omitempty"`
	ReportSource       string `xml:"reportSource,attr" json:"reportSource,omitempty"`
	CoastIndicator     string `xml:"coastIndicator,attr" json:"coastIndicator,omitempty"`

	ActualSpeed ActualSpeed `xml:"actualSpeed" json:"actualSpeed,omitempty"`

	Altitude Measurement `xml:"altitude" json:"altitude,omitempty"`

	Location Location `xml:"location" json:"location"`

	TargetAltitude Measurement `xml:"targetAltitude" json:"targetAltitude,omitempty"`

	TargetPosition Location `xml:"targetPosition" json:"targetPosition,omitempty"`

	TrackVelocity TrackVelocity `xml:"trackVelocity" json:"trackVelocity,omitempty"`
}

type ActualSpeed struct {
	Surveillance string `xml:",chardata" json:"surveillance,omitempty"`
	UOM          string `xml:"uom,attr" json:"uom,omitempty"`
}

type TrackVelocity struct {
	X   string `xml:",chardata" json:"x,omitempty"`
	Y   string `xml:",chardata" json:"y,omitempty"`
	UOM string `xml:"uom,attr" json:"uom,omitempty"`
}

type Location struct {
	SRSName string `xml:"srsName,attr" json:"srsName,omitempty"`
	Pos     string `xml:"pos" json:"pos,omitempty"`
}

type Measurement struct {
	Value string `xml:",chardata" json:"value,omitempty"`
	UOM   string `xml:"uom,attr" json:"uom,omitempty"`
}

// ---------------- Operator ----------------

type Operator struct {
	OperatingOrganization OperatingOrganization `xml:"operatingOrganization" json:"operatingOrganization,omitempty"`
}

type OperatingOrganization struct {
	Organization Organization `xml:"organization" json:"organization,omitempty"`
}

type Organization struct {
	Name string `xml:"name,attr" json:"name,omitempty"`
}

// ---------------- Supplemental Data ----------------

type SupplementalData struct {
	AdditionalFlightInformation AdditionalFlightInformation `xml:"additionalFlightInformation" json:"additionalFlightInformation,omitempty"`
}

type AdditionalFlightInformation struct {
	NameValue []NameValue `xml:"nameValue" json:"nameValue,omitempty"`
}

type NameValue struct {
	Name  string `xml:"name,attr" json:"name,omitempty"`
	Value string `xml:"value,attr" json:"value,omitempty"`
}

// ---------------- Coordination ----------------

type Coordination struct {
	CoordinationTimeHandling string `xml:"coordinationTimeHandling,attr" json:"coordinationTimeHandling,omitempty"`
	CoordinationTime         string `xml:"coordinationTime,attr" json:"coordinationTime,omitempty"`
	DelayTimeToAbsorb        string `xml:"delayTimeToAbsorb,attr" json:"delayTimeToAbsorb,omitempty"`

	CoordinationFix CoordinationFix `xml:"coordinationFix" json:"coordinationFix,omitempty"`
}

type CoordinationFix struct {
	Fix string `xml:"fix,attr" json:"fix,omitempty"`

	Distance Measurement `xml:"distance" json:"distance,omitempty"`

	Radial Measurement `xml:"radial" json:"radial,omitempty"`

	Location Location `xml:"location" json:"location,omitempty"`
}

// ---------------- Flight Plan ----------------

type FlightPlan struct {
	Identifier string `xml:"identifier,attr" json:"identifier,omitempty"`

	FlightPlanRemarks string `xml:"flightPlanRemarks,attr" json:"flightPlanRemarks,omitempty"`
}

// ---------------- Control ----------------

type ControllingUnit struct {
	UnitIdentifier   string `xml:"unitIdentifier,attr" json:"unitIdentifier,omitempty"`
	SectorIdentifier string `xml:"sectorIdentifier,attr" json:"sectorIdentifier,omitempty"`
}

// ---------------- Altitude ----------------

type AssignedAltitude struct {
	Simple Measurement `xml:"simple" json:"simple,omitempty"`

	Block BlockAltitude `xml:"block" json:"block,omitempty"`

	VFRPlus Measurement `xml:"vfrPlus" json:"vfrPlus,omitempty"`

	VFROnTopPlus Measurement `xml:"vfrOnTopPlus" json:"vfrOnTopPlus,omitempty"`
}

type BlockAltitude struct {
	Above Measurement `xml:"above" json:"above,omitempty"`

	Below Measurement `xml:"below" json:"below,omitempty"`
}

type RequestedAltitude struct {
	Simple Measurement `xml:"simple" json:"simple,omitempty"`

	VFRPlus Measurement `xml:"vfrPlus" json:"vfrPlus,omitempty"`

	VFROnTopPlus Measurement `xml:"vfrOnTopPlus" json:"vfrOnTopPlus,omitempty"`
}

// ---------------- Routes ----------------

type Agreed struct {
	Route Route `xml:"route" json:"route,omitempty"`
}

type Route struct {
	NASRouteText       string `xml:"nasRouteText,attr" json:"nasRouteText,omitempty"`
	LocalIntendedRoute string `xml:"localIntendedRoute,attr" json:"localIntendedRoute,omitempty"`
	ATCIntendedRoute   string `xml:"atcIntendedRoute,attr" json:"atcIntendedRoute,omitempty"`

	InitialFlightRules string `xml:"initialFlightRules,attr" json:"initialFlightRules,omitempty"`

	FlightDuration string `xml:"flightDuration,attr" json:"flightDuration,omitempty"`

	ExpandedRoute ExpandedRoute `xml:"expandedRoute" json:"expandedRoute,omitempty"`
}

type ExpandedRoute struct {
	RoutePoints []RoutePoint `xml:"routePoint" json:"routePoints,omitempty"`
}

type RoutePoint struct {
	EstimatedTime string `xml:"estimatedTime,attr" json:"estimatedTime,omitempty"`

	Point Point `xml:"point" json:"point,omitempty"`
}

type Point struct {
	Fix string `xml:"fix,attr" json:"fix,omitempty"`

	Location Location `xml:"location" json:"location,omitempty"`

	Distance Measurement `xml:"distance" json:"distance,omitempty"`

	Radial Measurement `xml:"radial" json:"radial,omitempty"`
}

// ---------------- Aircraft ----------------

type AircraftDescription struct {
	EquipmentQualifier string `xml:"equipmentQualifier,attr" json:"equipmentQualifier,omitempty"`
	WakeTurbulence     string `xml:"wakeTurbulence,attr" json:"wakeTurbulence,omitempty"`
	Registration       string `xml:"registration,attr" json:"registration,omitempty"`
	AircraftAddress    string `xml:"aircraftAddress,attr" json:"aircraftAddress,omitempty"`

	AircraftType AircraftType `xml:"aircraftType" json:"aircraftType,omitempty"`

	Capabilities Capabilities `xml:"capabilities" json:"capabilities,omitempty"`
}

type AircraftType struct {
	ICAOModelIdentifier string `xml:"icaoModelIdentifier" json:"icaoModelIdentifier,omitempty"`

	OtherModelData string `xml:"otherModelData" json:"otherModelData,omitempty"`
}

type Capabilities struct {
	StandardCapabilities string `xml:"standardCapabilities,attr" json:"standardCapabilities,omitempty"`

	Communication Communication `xml:"communication" json:"communication,omitempty"`

	Navigation Navigation `xml:"navigation" json:"navigation,omitempty"`

	Surveillance Surveillance `xml:"surveillance" json:"surveillance,omitempty"`
}

type Communication struct {
	SelectiveCallingCode string `xml:"selectiveCallingCode,attr" json:"selectiveCallingCode,omitempty"`

	CommunicationCode string `xml:"communicationCode" json:"communicationCode,omitempty"`

	DataLinkCode string `xml:"dataLinkCode" json:"dataLinkCode,omitempty"`
}

type Navigation struct {
	NavigationCode string `xml:"navigationCode" json:"navigationCode,omitempty"`

	PerformanceBasedCode string `xml:"performanceBasedCode" json:"performanceBasedCode,omitempty"`
}

type Surveillance struct {
	SurveillanceCode string `xml:"surveillanceCode" json:"surveillanceCode,omitempty"`
}

// ---------------- Requested Speed ----------------

type RequestedAirspeed struct {
	NASAirspeed Measurement `xml:"nasAirspeed" json:"nasAirspeed,omitempty"`
}

// ---------------- Originator ----------------

type Originator struct {
	AFTNAddress string `xml:"aftnAddress" json:"aftnAddress,omitempty"`

	FlightOriginator string `xml:"flightOriginator" json:"flightOriginator,omitempty"`
}

// ---------------- Boundary Crossings ----------------

type BoundaryCrossings struct {
	Handoff []Handoff `xml:"handoff" json:"handoff,omitempty"`
}

type Handoff struct {
	Event string `xml:"event,attr" json:"event,omitempty"`

	ReceivingUnit Unit `xml:"receivingUnit" json:"receivingUnit,omitempty"`

	TransferringUnit Unit `xml:"transferringUnit" json:"transferringUnit,omitempty"`

	AcceptingUnit Unit `xml:"acceptingUnit" json:"acceptingUnit,omitempty"`
}

type Unit struct {
	UnitIdentifier string `xml:"unitIdentifier,attr" json:"unitIdentifier,omitempty"`

	SectorIdentifier string `xml:"sectorIdentifier,attr" json:"sectorIdentifier,omitempty"`
}

type AlternateAerodrome struct {
	Code string `xml:"code,attr" json:"code,omitempty"`

	Name string `xml:"name,attr" json:"name,omitempty"`
}
