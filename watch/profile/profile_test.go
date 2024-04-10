package profile_test

import (
	"runtime/pprof"
	"testing"
)

func TestProfile(t *testing.T) {
	pprof.NewProfile()
}
