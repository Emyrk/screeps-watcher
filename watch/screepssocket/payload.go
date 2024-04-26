package screepssocket

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// batchPayload handles payloads as an array of string serialized json objects.
func batchPayload(ctx context.Context, data []byte) ([]json.RawMessage, error) {
	array := make([]json.RawMessage, 0)
	err := json.Unmarshal(data, &array)
	if err != nil {
		return nil, fmt.Errorf("unmarshal payload as array: %w", err)
	}

	deserialized := make([]json.RawMessage, 0, len(array))
	for _, msgJson := range array {
		var msg json.RawMessage

		unquoted, err := strconv.Unquote(string(msgJson))
		if err != nil {
			// UTF16 really messes this up. Have this dumb backup that works most
			// of the time.
			unquoted = dumbUnquote(string(msgJson))
		}

		err = json.Unmarshal([]byte(unquoted), &msg)
		if err != nil {
			//return nil, fmt.Errorf("unmarshal unquoted message %q: %w", unquoted, err)
			// This is so bad, but string literals are just encoded once.
			// Objects are encoded twice....
			err = json.Unmarshal(msgJson, &msg)
			if err != nil {
				return nil, fmt.Errorf("unmarshal unquoted message %q: %w", unquoted, err)
			}
		}
		deserialized = append(deserialized, msg)
	}

	return deserialized, nil
}

func dumbUnquote(str string) string {
	str = strings.Replace(str, `\"`, `"`, -1)
	return strings.Trim(str, `"`)

}
