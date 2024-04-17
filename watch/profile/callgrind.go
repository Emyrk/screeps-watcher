package profile

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/Emyrk/screeps-watcher/watch/profile/callgrind"

	"github.com/google/pprof/profile"
)

type Converter struct {
	fid       uint64
	functions map[string]*profile.Function
	locations map[string]*profile.Location

	callsRecorded map[uint64]bool
	protobuf      *profile.Profile
}

func New() *Converter {
	return &Converter{
		functions: make(map[string]*profile.Function),
		locations: make(map[string]*profile.Location),
		protobuf: &profile.Profile{
			SampleType: []*profile.ValueType{
				{Type: "cpu", Unit: "cycles"},
				//{Type: "calls", Unit: "count"},
			},
			DefaultSampleType: "cpu",
			Sample:            []*profile.Sample{},
			Mapping:           []*profile.Mapping{},
			Location:          []*profile.Location{},
			Function:          []*profile.Function{},
			Comments:          []string{},
			// Ignore these regex filters for now
			DropFrames: "",
			KeepFrames: "",

			// TODO: @emyrk get a more accurate timestamp from the client.
			TimeNanos: 0,
			// TODO: @emyrk this should be included to indicate how long the
			// 		profile ran. The amount of time run should be the CPU_LIMIT.
			//		If the tick exceeds the profile limit, that tick full duration
			//		should be included.
			//DurationNanos:     0,
			// TODO: @emyrk This will be helpful when we know the periodic nature
			// 	of the profile. For example, if 100 ticks are profiled every 1000 ticks,
			// 	that information can be encoded here.
			//PeriodType:        nil,
			//Period:            0,
		},
	}
}

func ConvertCallgrind(input string) ([]byte, error) {
	p := callgrind.NewCallgrindParser(strings.NewReader(input))
	profiler, err := p.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse callgrind: %w", err)
	}

	converter := New()
	converter.Convert(profiler)

	var buf bytes.Buffer
	converter.protobuf.Write(&buf)
	fmt.Println(converter.protobuf.String())
	return buf.Bytes(), nil
}

func (c *Converter) Convert(cg *callgrind.Profile) {
	for _, root := range cg.Roots() {
		// Roots have no cost by default. So range through their calls.
		calls := root.Calls()
		for _, call := range calls {
			fmt.Println(call.Cost)
		}
		//c.recurseFunctions(cg, root, nil)
	}
}

func (c *Converter) recurseFunctions(cg *callgrind.Profile, f *callgrind.Function, sample *profile.Sample) {
	fn, loc := c.function(f.Name)
	// If no sample exists, bootstrap the first sample.
	if sample == nil {
		sample = &profile.Sample{
			Location: []*profile.Location{loc},
			Value:    []int64{f.Cost.Milliseconds()},
			Label:    nil,
			NumLabel: nil,
			NumUnit:  nil,
		}
		c.protobuf.Sample = append(c.protobuf.Sample, sample)
	} else {
		// Add this function call to the existing sample.
		// Prepend location, as location[0] is the leaf node.
		sample.Location = append([]*profile.Location{loc}, sample.Location...)
	}

	var _ = fn
	// For each call, recurse and add to the sample.
	calls := f.Calls()
	if len(calls) == 0 {
		return
	}

	for _, call := range calls {
		callFn, callLoc := c.function(call.CalleeId)
		callSample := &profile.Sample{
			Location: []*profile.Location{callLoc},
			Value:    []int64{call.Cost.Milliseconds()},
		}
		c.protobuf.Sample = append(c.protobuf.Sample, callSample)
		cgf, _ := cg.GetFunction(call.CalleeId)
		c.recurseFunctions(cg, cgf, callSample)
		var _ = callFn
		fmt.Println(callFn.Name)
	}
}

func (c *Converter) function(name string) (*profile.Function, *profile.Location) {
	if fn, found := c.functions[name]; found {
		return fn, c.locations[name]
	}

	c.fid++
	fn := &profile.Function{
		ID:         c.fid,
		Name:       name,
		SystemName: name,
		// TODO: get this info maybe?
		Filename:  "main.ts",
		StartLine: 1,
	}

	c.functions[name] = fn
	c.protobuf.Function = append(c.protobuf.Function, fn)

	loc := &profile.Location{
		ID:      c.fid,
		Mapping: nil,
		Address: 0,
		Line: []profile.Line{
			{
				Function: fn,
				// TODO: Should we get his info?
				Line:   fn.StartLine,
				Column: 0,
			},
		},
		IsFolded: false,
	}
	c.locations[name] = loc
	c.protobuf.Location = append(c.protobuf.Location, loc)
	return fn, loc
}

func prepend[T any](s []T, x T) []T {
	return append([]T{x}, s...)
}
