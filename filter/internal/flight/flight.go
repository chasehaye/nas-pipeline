package flight

import "encoding/json"


type Ident struct {
	CallSign     string // FIXM aircraftIdentification
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
	} `json:"flight"`
}


type Message struct {
	Raw   json.RawMessage
	Ident Ident
}


func Parse(payload []byte) ([]Message, error) {
	var raws []json.RawMessage
	if err := json.Unmarshal(payload, &raws); err != nil {
		return nil, err
	}

	msgs := make([]Message, 0, len(raws))
	for _, raw := range raws {
		var e element
		if err := json.Unmarshal(raw, &e); err != nil {
			return nil, err
		}
		msgs = append(msgs, Message{
			Raw: raw,
			Ident: Ident{
				CallSign:     e.Flight.FlightIdentification.AircraftIdentification,
				Registration: e.Flight.AircraftDescription.Registration,
			},
		})
	}
	return msgs, nil
}


func Marshal(msgs []Message) ([]byte, error) {
	raws := make([]json.RawMessage, len(msgs))
	for i, m := range msgs {
		raws[i] = m.Raw
	}
	return json.Marshal(raws)
}
