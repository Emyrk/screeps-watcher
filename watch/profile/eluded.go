package profile

import (
	"bytes"

	"github.com/Emyrk/screeps-watcher/watch/profile/eluded"
	"github.com/google/pprof/profile"
)

type Converter struct {
	fid       uint64
	functions map[string]*profile.Function
	locations map[string]*profile.Location

	protobuf *profile.Profile
}

func New() *Converter {
	return &Converter{
		functions: make(map[string]*profile.Function),
		locations: make(map[string]*profile.Location),
		protobuf: &profile.Profile{
			SampleType: []*profile.ValueType{
				{Type: "cpu", Unit: "nanoseconds"},
				{Type: "samples", Unit: "count"},
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

func (c *Converter) Convert(elu []eluded.Profile) *profile.Profile {
	for _, tick := range elu {
		tick.Key = "tick"
		c.ConvertSingle(tick)
	}
	return c.protobuf
}

func (c *Converter) Encode() ([]byte, error) {
	var buf bytes.Buffer
	err := c.protobuf.Write(&buf)
	return buf.Bytes(), err
}

func (c *Converter) ConvertSingle(elu eluded.Profile) {
	c.recurseFunctions(elu, nil)

}

func (c *Converter) recurseFunctions(elu eluded.Profile, sample *profile.Sample) {
	_, loc := c.function(elu.Key)
	// If no sample exists, bootstrap the first sample.
	if sample == nil {
		sample = &profile.Sample{
			Location: []*profile.Location{loc},
			Value:    []int64{elu.SelfCostNano(), 1},
			Label:    nil,
			NumLabel: nil,
			NumUnit:  nil,
		}
		c.protobuf.Sample = append(c.protobuf.Sample, sample)
	} else {
		// Add this function call to the existing sample.
		// Prepend location, as location[0] is the leaf node.
		sample.Location = prepend(loc, sample.Location)
	}

	for _, call := range elu.Children {
		// For each child, prepend the stack and the cost of the child.
		_, callLoc := c.function(call.Key)
		callSample := &profile.Sample{
			Location: prepend(callLoc, sample.Location),
			Value:    []int64{call.SelfCostNano(), 1},
		}
		c.protobuf.Sample = append(c.protobuf.Sample, callSample)
		c.recurseFunctions(call, callSample)
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
