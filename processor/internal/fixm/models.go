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

	// Rarer flight-level elements observed in the census.
	RouteToRevisedDestination    RouteToRevisedDestination `xml:"routeToRevisedDestination" json:"routeToRevisedDestination,omitempty"`
	FlightIdentificationPrevious FlightIdentification      `xml:"flightIdentificationPrevious" json:"flightIdentificationPrevious,omitempty"`
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

	TakeoffAlternateAerodrome Aerodrome         `xml:"takeoffAlternateAerodrome" json:"takeoffAlternateAerodrome,omitempty"`
	DepartureAerodrome        AerodromeLocation `xml:"departureAerodrome" json:"departureAerodrome,omitempty"`
}

type Arrival struct {
	ArrivalPoint string `xml:"arrivalPoint,attr" json:"arrivalPoint,omitempty"`

	RunwayPositionAndTime RunwayPositionAndTime `xml:"runwayPositionAndTime" json:"runwayPositionAndTime,omitempty"`

	// [x2] under a singular <arrival>, so a slice.
	ArrivalAerodromeAlternate []Aerodrome `xml:"arrivalAerodromeAlternate" json:"arrivalAerodromeAlternate,omitempty"`

	// Very rare (2 in 462k flights); the arrival-airport coordinates.
	ArrivalAerodrome AerodromeLocation `xml:"arrivalAerodrome" json:"arrivalAerodrome,omitempty"`
}

// AerodromeLocation carries an airport's coordinates, used by both
// departureAerodrome and arrivalAerodrome.
type AerodromeLocation struct {
	Point Point `xml:"point" json:"point,omitempty"`
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
	// xsi:nil="true" marks the time as explicitly absent (seen on controlled).
	Nil string `xml:"nil,attr" json:"nil,omitempty"`
}

// Aerodrome is an airport reference (code + optional name), used for the
// several alternate/takeoff aerodrome elements.
type Aerodrome struct {
	Code string `xml:"code,attr" json:"code,omitempty"`
	Name string `xml:"name,attr" json:"name,omitempty"`
}

// ---------------- En Route ----------------

type EnRoute struct {
	Positions         []Position        `xml:"position" json:"positions,omitempty"`
	BoundaryCrossings BoundaryCrossings `xml:"boundaryCrossings" json:"boundaryCrossings,omitempty"`

	// [x5] under a singular <enRoute>, so a slice.
	AlternateAerodrome []Aerodrome `xml:"alternateAerodrome" json:"alternateAerodrome,omitempty"`

	Pointout                     Pointout             `xml:"pointout" json:"pointout,omitempty"`
	BeaconCodeAssignment         BeaconCodeAssignment `xml:"beaconCodeAssignment" json:"beaconCodeAssignment,omitempty"`
	Cleared                      Cleared              `xml:"cleared" json:"cleared,omitempty"`
	ExpectedFurtherClearanceTime TimeValue            `xml:"expectedFurtherClearanceTime" json:"expectedFurtherClearanceTime,omitempty"`
}

type Position struct {
	PositionTime       string `xml:"positionTime,attr" json:"positionTime,omitempty"`
	TargetPositionTime string `xml:"targetPositionTime,attr" json:"targetPositionTime,omitempty"`
	ReportSource       string `xml:"reportSource,attr" json:"reportSource,omitempty"`
	CoastIndicator     string `xml:"coastIndicator,attr" json:"coastIndicator,omitempty"`

	ActualSpeed ActualSpeed `xml:"actualSpeed" json:"actualSpeed,omitempty"`

	Altitude Measurement `xml:"altitude" json:"altitude,omitempty"`

	// The actual position sits under a nested <position> wrapper
	// (position/position/location/pos), not as a direct child, so it needs the
	// path expression. Bound to "location" alone it silently came back empty
	// while targetPosition (a real direct child) populated.
	Location Location `xml:"position>location" json:"location"`

	TargetAltitude Measurement `xml:"targetAltitude" json:"targetAltitude,omitempty"`

	TargetPosition Location `xml:"targetPosition" json:"targetPosition,omitempty"`

	TrackVelocity TrackVelocity `xml:"trackVelocity" json:"trackVelocity,omitempty"`
}

type ActualSpeed struct {
	Surveillance Measurement `xml:"surveillance" json:"surveillance,omitempty"`
}

type TrackVelocity struct {
	X Measurement `xml:"x" json:"x,omitempty"`
	Y Measurement `xml:"y" json:"y,omitempty"`
}

type Location struct {
	SRSName string `xml:"srsName,attr" json:"srsName,omitempty"`
	Pos     string `xml:"pos" json:"pos,omitempty"`
}

type Measurement struct {
	Value string `xml:",chardata" json:"value,omitempty"`
	UOM   string `xml:"uom,attr" json:"uom,omitempty"`
	// Some measurements carry markers: targetAltitude has @invalid, several
	// altitudes carry xsi:nil="true".
	Invalid string `xml:"invalid,attr" json:"invalid,omitempty"`
	Nil     string `xml:"nil,attr" json:"nil,omitempty"`
}

// Pointout: originating unit hands attention to one or more receiving units.
type Pointout struct {
	OriginatingUnit Unit `xml:"originatingUnit" json:"originatingUnit,omitempty"`
	// [x2] under a singular <pointout>, so a slice.
	ReceivingUnit []Unit `xml:"receivingUnit" json:"receivingUnit,omitempty"`
}

type BeaconCodeAssignment struct {
	CurrentBeaconCode    string `xml:"currentBeaconCode" json:"currentBeaconCode,omitempty"`
	ReassignedBeaconCode string `xml:"reassignedBeaconCode" json:"reassignedBeaconCode,omitempty"`
	PreviousBeaconCode   string `xml:"previousBeaconCode" json:"previousBeaconCode,omitempty"`
}

type Cleared struct {
	ClearanceSpeed   string `xml:"clearanceSpeed,attr" json:"clearanceSpeed,omitempty"`
	ClearanceHeading string `xml:"clearanceHeading,attr" json:"clearanceHeading,omitempty"`
	ClearanceText    string `xml:"clearanceText,attr" json:"clearanceText,omitempty"`
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

// Fix is a fix reference with optional distance/radial offset and location.
// Used by coordinationFix and holdFix.
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

	Vfr          Measurement `xml:"vfr" json:"vfr,omitempty"`
	VFRPlus      Measurement `xml:"vfrPlus" json:"vfrPlus,omitempty"`
	VFROnTopPlus Measurement `xml:"vfrOnTopPlus" json:"vfrOnTopPlus,omitempty"`

	AltFixAlt AltFixAlt `xml:"altFixAlt" json:"altFixAlt,omitempty"`
}

type BlockAltitude struct {
	Above Measurement `xml:"above" json:"above,omitempty"`

	Below Measurement `xml:"below" json:"below,omitempty"`
}

// AltFixAlt: cross a fix stepping between a pre-fix and post-fix altitude.
type AltFixAlt struct {
	Point Point       `xml:"point" json:"point,omitempty"`
	Post  Measurement `xml:"post" json:"post,omitempty"`
	Pre   Measurement `xml:"pre" json:"pre,omitempty"`
}

type RequestedAltitude struct {
	Simple Measurement `xml:"simple" json:"simple,omitempty"`

	Vfr          Measurement `xml:"vfr" json:"vfr,omitempty"`
	VFRPlus      Measurement `xml:"vfrPlus" json:"vfrPlus,omitempty"`
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

	// [x36] under a singular <route>, so a slice.
	EstimatedElapsedTime []EstimatedElapsedTime `xml:"estimatedElapsedTime" json:"estimatedElapsedTime,omitempty"`

	NASAdaptedArrivalRoute       NASAdaptedRoute `xml:"nasadaptedArrivalRoute" json:"nasadaptedArrivalRoute,omitempty"`
	AdaptedDepartureRoute        NASAdaptedRoute `xml:"adaptedDepartureRoute" json:"adaptedDepartureRoute,omitempty"`
	AdaptedArrivalDepartureRoute NASAdaptedRoute `xml:"adaptedArrivalDepartureRoute" json:"adaptedArrivalDepartureRoute,omitempty"`

	HoldFix CoordinationFix `xml:"holdFix" json:"holdFix,omitempty"`
}

type ExpandedRoute struct {
	RoutePoints []RoutePoint `xml:"routePoint" json:"routePoints,omitempty"`
}

type RoutePoint struct {
	EstimatedTime string `xml:"estimatedTime,attr" json:"estimatedTime,omitempty"`

	// Singular: one <point> per <routePoint>. The high [xN] in the census is
	// the per-flight total across many routePoints, not a per-parent slice.
	Point Point `xml:"point" json:"point,omitempty"`
}

type Point struct {
	Fix string `xml:"fix,attr" json:"fix,omitempty"`

	Location Location `xml:"location" json:"location,omitempty"`

	Distance Measurement `xml:"distance" json:"distance,omitempty"`

	Radial Measurement `xml:"radial" json:"radial,omitempty"`
}

// EstimatedElapsedTime: time to reach a boundary point/region along the route.
type EstimatedElapsedTime struct {
	ElapsedTime string      `xml:"elapsedTime,attr" json:"elapsedTime,omitempty"`
	Location    EETLocation `xml:"location" json:"location,omitempty"`
}

// EETLocation is a choice of a region, a point, or a longitude line.
type EETLocation struct {
	Region    Region `xml:"region" json:"region,omitempty"`
	Point     Point  `xml:"point" json:"point,omitempty"`
	Longitude string `xml:"longitude" json:"longitude,omitempty"`
}

type Region struct {
	AirspaceType string `xml:"airspaceType,attr" json:"airspaceType,omitempty"`
	Code         string `xml:",chardata" json:"code,omitempty"`
}

// NASAdaptedRoute covers the nasadaptedArrivalRoute / adaptedDepartureRoute /
// adaptedArrivalDepartureRoute variants (nasFavNumber only on some).
type NASAdaptedRoute struct {
	NASRouteIdentifier   string `xml:"nasRouteIdentifier,attr" json:"nasRouteIdentifier,omitempty"`
	NASRouteAlphanumeric string `xml:"nasRouteAlphanumeric,attr" json:"nasRouteAlphanumeric,omitempty"`
	NASFavNumber         string `xml:"nasFavNumber" json:"nasFavNumber,omitempty"`
}

// RouteToRevisedDestination uses a plain routeText attribute, distinct from the
// agreed route's nasRouteText.
type RouteToRevisedDestination struct {
	Route RevisedRoute `xml:"route" json:"route,omitempty"`
}

type RevisedRoute struct {
	RouteText string `xml:"routeText,attr" json:"routeText,omitempty"`
}

// ---------------- Aircraft ----------------

type AircraftDescription struct {
	EquipmentQualifier           string `xml:"equipmentQualifier,attr" json:"equipmentQualifier,omitempty"`
	WakeTurbulence               string `xml:"wakeTurbulence,attr" json:"wakeTurbulence,omitempty"`
	Registration                 string `xml:"registration,attr" json:"registration,omitempty"`
	AircraftAddress              string `xml:"aircraftAddress,attr" json:"aircraftAddress,omitempty"`
	AircraftPerformance          string `xml:"aircraftPerformance,attr" json:"aircraftPerformance,omitempty"`
	TFMSSpecialAircraftQualifier string `xml:"tfmsSpecialAircraftQualifier,attr" json:"tfmsSpecialAircraftQualifier,omitempty"`
	AircraftQuantity             string `xml:"aircraftQuantity,attr" json:"aircraftQuantity,omitempty"`

	AircraftType AircraftType `xml:"aircraftType" json:"aircraftType,omitempty"`

	Capabilities Capabilities `xml:"capabilities" json:"capabilities,omitempty"`

	Accuracy Accuracy `xml:"accuracy" json:"accuracy,omitempty"`
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
	SelectiveCallingCode           string `xml:"selectiveCallingCode,attr" json:"selectiveCallingCode,omitempty"`
	OtherDataLinkCapabilities      string `xml:"otherDataLinkCapabilities,attr" json:"otherDataLinkCapabilities,omitempty"`
	OtherCommunicationCapabilities string `xml:"otherCommunicationCapabilities,attr" json:"otherCommunicationCapabilities,omitempty"`

	CommunicationCode string `xml:"communicationCode" json:"communicationCode,omitempty"`

	DataLinkCode string `xml:"dataLinkCode" json:"dataLinkCode,omitempty"`
}

type Navigation struct {
	OtherNavigationCapabilities string `xml:"otherNavigationCapabilities,attr" json:"otherNavigationCapabilities,omitempty"`

	NavigationCode string `xml:"navigationCode" json:"navigationCode,omitempty"`

	PerformanceBasedCode string `xml:"performanceBasedCode" json:"performanceBasedCode,omitempty"`
}

type Surveillance struct {
	OtherSurveillanceCapabilities string `xml:"otherSurveillanceCapabilities,attr" json:"otherSurveillanceCapabilities,omitempty"`

	SurveillanceCode string `xml:"surveillanceCode" json:"surveillanceCode,omitempty"`
}

// Accuracy holds required navigation performance figures per flight phase.
type Accuracy struct {
	// [x3] under a singular <accuracy>, so a slice.
	CmsFieldType []CmsFieldType `xml:"cmsFieldType" json:"cmsFieldType,omitempty"`
}

type CmsFieldType struct {
	Phase string `xml:"phase,attr" json:"phase,omitempty"`
	// Plain "type" attribute here (not xsi:type); matched by local name.
	Type  string `xml:"type,attr" json:"type,omitempty"`
	UOM   string `xml:"uom,attr" json:"uom,omitempty"`
	Value string `xml:",chardata" json:"value,omitempty"`
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
