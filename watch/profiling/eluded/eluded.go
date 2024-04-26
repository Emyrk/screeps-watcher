package eluded

import (
	_ "embed"
	"encoding/json"
)

var Example []Profile

func init() {
	err := json.Unmarshal([]byte(ExampleData), &Example)
	if err != nil {
		panic(err)
	}
}

//go:embed dump.json
var ExampleData string

type Profile struct {
	Key       string    `json:"key"`
	Start     float64   `json:"start"`
	CPU       float64   `json:"cpu"`
	Children  []Profile `json:"children"`
	UnixMilli int64     `json:"um,omitempty"`
}

// CPUNano returns the number of nanoseconds. Probably too much
// precision.
// Default is in ms, so just bump by a factor of 6.
func (p Profile) CPUNano() int64 {
	return int64(p.CPU * 1e6)
}

func (p Profile) SelfCostNano() int64 {
	total := p.CPU
	for _, child := range p.Children {
		total -= child.CPU
	}
	return int64(total * 1e6)
}
