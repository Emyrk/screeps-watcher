package profile

import (
	"bytes"
	"fmt"

	"github.com/Emyrk/screeps-watcher/watch/profile/callgrind"
)

func ConvertCallgrind() {
	p := callgrind.NewCallgrindParser(bytes.NewBufferString(callgrind.Example))
	profiler, err := p.Parse()
	if err != nil {
		panic(err)
	}
	fmt.Println(profiler)
}
