package fixm

import (
	"encoding/json"
	"fmt"
)

func EncodeOne(m Message) ([]byte, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("json encode: %w", err)
	}

	return data, nil
}