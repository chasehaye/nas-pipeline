package flight

import "encoding/json"


type Ident struct {
	CallSign     string
	Registration string
}

type element struct {
	Flight struct {
		FlightIdentification struct {
			AircraftIdentification string `json:"aircraftIdentification"`
		} `json:"flightIdentification"`
		AircraftDescription struct {
			Registration string `json:"registration"`
		} `json:"aircraftDescription"`
		Gufi struct {
			Code string `json:"code"`
		} `json:"gufi"`
	} `json:"flight"`
}


type Message struct {
	Raw   json.RawMessage
	Ident Ident
	Gufi  string
}


func Parse(payload []byte) (Message, error) {
	var e element
	if err := json.Unmarshal(payload, &e); err != nil {
		return Message{}, err
	}
	return Message{
		Raw: payload,
		Ident: Ident{
			CallSign:     e.Flight.FlightIdentification.AircraftIdentification,
			Registration: e.Flight.AircraftDescription.Registration,
		},
		Gufi: e.Flight.Gufi.Code,
	}, nil
}
