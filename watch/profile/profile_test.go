package profile_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	profile2 "github.com/Emyrk/screeps-watcher/watch/profile"
	"github.com/Emyrk/screeps-watcher/watch/profile/callgrind"
	"github.com/Emyrk/screeps-watcher/watch/profile/eluded"
	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
)

func TestEluded(t *testing.T) {
	converter := profile2.New()
	converter.Convert(eluded.Example)
	data, err := converter.Encode()
	require.NoError(t, err)

	os.WriteFile("created.pprof", data, 0644)
}

func TestParsePProf(t *testing.T) {
	proto, err := profile.Parse(bytes.NewBufferString(callgrind.PProfDemo))
	require.NoError(t, err)
	fmt.Println(proto.String())
	fmt.Println("")
}

var testM = []*profile.Mapping{
	{
		ID:              1,
		Start:           1,
		Limit:           10,
		Offset:          0,
		File:            "file1",
		BuildID:         "buildid1",
		HasFunctions:    true,
		HasFilenames:    true,
		HasLineNumbers:  true,
		HasInlineFrames: true,
	},
	{
		ID:              2,
		Start:           10,
		Limit:           30,
		Offset:          9,
		File:            "file1",
		BuildID:         "buildid2",
		HasFunctions:    true,
		HasFilenames:    true,
		HasLineNumbers:  true,
		HasInlineFrames: true,
	},
}

var func1 = &profile.Function{ID: 1, Name: "func1", SystemName: "func1", Filename: "file1"}
var func2 = &profile.Function{ID: 2, Name: "func2", SystemName: "func2", Filename: "file1"}
var func3 = &profile.Function{ID: 3, Name: "func3", SystemName: "func3", Filename: "file2"}
var func4 = &profile.Function{ID: 4, Name: "func4", SystemName: "func4", Filename: "file3"}
var func5 = &profile.Function{ID: 5, Name: "func5", SystemName: "func5", Filename: "file4"}

var testL = []*profile.Location{
	{
		ID:      1,
		Address: 1,
		Mapping: testM[0],
		Line: []profile.Line{
			//{
			//	Function: func1,
			//	Line:     2,
			//},
			{
				Function: func2,
				Line:     2222222,
			},
		},
	},
	{
		ID:      2,
		Mapping: testM[1],
		Address: 11,
		Line: []profile.Line{
			{
				Function: func3,
				Line:     2,
			},
		},
	},
	{
		ID:      3,
		Mapping: testM[1],
		Address: 12,
	},
	{
		ID:      4,
		Mapping: testM[1],
		Address: 12,
		Line: []profile.Line{
			{
				Function: func5,
				Line:     6,
			},
			{
				Function: func5,
				Line:     6,
			},
		},
		IsFolded: true,
	},
}

var all = &profile.Profile{
	PeriodType:    &profile.ValueType{Type: "cpu", Unit: "seconds"},
	Period:        10,
	DurationNanos: 100e9,
	SampleType: []*profile.ValueType{
		{Type: "cpu", Unit: "cycles"},
		{Type: "object", Unit: "count"},
	},
	Sample: []*profile.Sample{
		{
			Location: []*profile.Location{testL[0], testL[1]},
			Label: map[string][]string{
				"key1": {"value1"},
				"key2": {"value2"},
			},
			Value: []int64{33, 20},
		},
		{
			Location: []*profile.Location{testL[0]},
			Label: map[string][]string{
				"key1": {"value1"},
				"key2": {"value2"},
			},
			Value: []int64{15, 20},
		},
		//{
		//	Location: []*profile.Location{testL[1], testL[2], testL[0], testL[1]},
		//	Value:    []int64{30, 40},
		//	Label: map[string][]string{
		//		"key1": {"value1"},
		//		"key2": {"value2"},
		//	},
		//	NumLabel: map[string][]int64{
		//		"key1":      {1, 2},
		//		"key2":      {3, 4},
		//		"bytes":     {3, 4},
		//		"requests":  {1, 1, 3, 4, 5},
		//		"alignment": {3, 4},
		//	},
		//	NumUnit: map[string][]string{
		//		"requests":  {"", "", "seconds", "", "s"},
		//		"alignment": {"kilobytes", "kilobytes"},
		//	},
		//},
		//{
		//	Location: []*profile.Location{testL[1], testL[2], testL[0], testL[1]},
		//	Value:    []int64{30, 40},
		//	NumLabel: map[string][]int64{
		//		"size": {0},
		//	},
		//	NumUnit: map[string][]string{
		//		"size": {"bytes"},
		//	},
		//},
	},
	Function: []*profile.Function{func1, func2, func3, func4, func5},
	Mapping:  testM,
	Location: testL,
	Comments: []string{"Comment 1", "Comment 2"},
}
