package profile

import (
	"fmt"

	"github.com/Emyrk/screeps-watcher/watch/profile/pprofproto"
	"google.golang.org/protobuf/proto"
)

//go:generate protoc --go_opt=paths=source_relative --go_out=. ./proto/profile.proto
func DemoPprof() {
	p := pprofproto.Profile{}

	p.Sample = append(p.Sample, &pprofproto.Sample{
		LocationId: nil,
		Value:      nil,
		Label:      nil,
	})

	data, err := proto.Marshal(&p)
	if err != nil {
		panic(err)
	}
	fmt.Println(data)
}
