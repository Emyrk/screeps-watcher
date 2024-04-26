package eluded

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

var Example []Profile

func init() {
	var err error
	Example, err = ParseProfileData([]byte(ExampleData))
	if err != nil {
		panic(err)
	}
	var _ = err
}

//go:embed dump2.json
var ExampleData string

type Profile struct {
	Key       string    `json:"key"`
	Start     float64   `json:"start"`
	CPU       float64   `json:"cpu"`
	Children  []Profile `json:"children"`
	UnixMilli int64     `json:"um,omitempty"`
}

type MinProfile struct {
	Key       string       `json:"k"`
	Start     float64      `json:"s"`
	CPU       float64      `json:"u"`
	Children  []MinProfile `json:"c"`
	UnixMilli int64        `json:"um,omitempty"`
}

func (m MinProfile) Profile() Profile {
	children := make([]Profile, 0, len(m.Children))
	for _, child := range m.Children {
		children = append(children, child.Profile())
	}
	return Profile{
		Key:       m.Key,
		Start:     m.Start,
		CPU:       m.CPU,
		Children:  children,
		UnixMilli: m.UnixMilli,
	}
}

func ParseProfileData(data []byte) ([]Profile, error) {
	if strings.Contains(string(data[0:10]), `"k":`) {
		// Use vs
		var min []MinProfile
		err := json.Unmarshal(data, &min)
		if err != nil {
			return nil, fmt.Errorf("minified profile: %w", err)
		}

		actual := make([]Profile, 0, len(min))
		for i := range min {
			actual = append(actual, min[i].Profile())
		}
		return actual, nil
	}

	var profile []Profile
	err := json.Unmarshal(data, &profile)
	return profile, err
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
