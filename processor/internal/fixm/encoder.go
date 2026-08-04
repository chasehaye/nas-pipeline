package fixm

import (
	"encoding/json"
	"fmt"
)

func EncodeJSON(messages []Message) ([]byte, error) {
	data, err := json.Marshal(messages)
	if err != nil {
		return nil, fmt.Errorf("json encode: %w", err)
	}

	return data, nil
}