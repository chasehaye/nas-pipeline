package fixm

import (
	"encoding/xml"
	"fmt"
)

func ParseEnvelope(data []byte) ([]Message, error) {
	var envelope MessageCollection

	if err := xml.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return envelope.Messages, nil
}