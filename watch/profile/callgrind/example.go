package callgrind

import _ "embed"

//go:embed data/callgrind.out.1570
var Example string

//go:embed data/example.pprof
var ExamplePprof string

//go:embed data/pprof-demo.pprof
var PProfDemo string
