package memory

import (
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type memoryResponse struct {
	Data string `json:"data"`
}

// Decode the base64 gzipped payload from the memory endpoint.
func Decode(data []byte) ([]byte, error) {
	var memResp memoryResponse
	err := json.Unmarshal(data, &memResp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal resp: %w", err)
	}

	if len(memResp.Data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	if len(memResp.Data) < 3 {
		return []byte(memResp.Data), nil
	}

	if memResp.Data[:3] != "gz:" {
		return []byte(memResp.Data), nil
	}

	memResp.Data = strings.Split(memResp.Data, "gz:")[1]
	decoded := base64.NewDecoder(base64.StdEncoding, strings.NewReader(memResp.Data))
	r, err := gzip.NewReader(decoded)
	if err != nil {
		return nil, fmt.Errorf("new gzip reader: %w", err)
	}

	all, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read all: %w", err)
	}

	return all, nil
}
