package fixm

import (
	"encoding/xml"
	"fmt"
)


func ProcessEnvelope(data []byte) error {
	var envelope MessageCollection
	if err := xml.Unmarshal(data, &envelope); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	for _, record := range envelope.Messages {
		f := record.Flight
		_ = f // TODO: branch on f.Source, write Redis + Postgres
	}

	return nil
}