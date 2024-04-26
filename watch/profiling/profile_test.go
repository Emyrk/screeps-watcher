package profiling_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/Emyrk/screeps-watcher/watch/profiling"
	"github.com/Emyrk/screeps-watcher/watch/profiling/eluded"
	"github.com/stretchr/testify/require"
)

func TestEluded(t *testing.T) {
	converter := profiling.New()
	profile := converter.Convert(eluded.Example)
	data, err := converter.Encode()
	require.NoError(t, err)

	var _ = profile
	os.WriteFile("created.pprof", data, 0644)
	fmt.Print(profile.String())
	//
	//err = profiling.PyroscopePush(profile)
	//require.NoError(t, err)
}
