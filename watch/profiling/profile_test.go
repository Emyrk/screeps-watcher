package profiling_test

import (
	_ "embed"
	"fmt"
	"os"
	"testing"

	"github.com/Emyrk/screeps-watcher/watch/profiling"
	"github.com/Emyrk/screeps-watcher/watch/profiling/eluded"
	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
)

//go:embed pserver.prof
var profileExample []byte

func TestEluded(t *testing.T) {
	converter := profiling.New()
	converted := converter.Convert(eluded.Example)
	data, err := converter.Encode()
	require.NoError(t, err)

	var _ = converted
	os.WriteFile("created.pprof", data, 0644)
	//fmt.Print(converted.String())

	for _, sample := range converted.Sample {
		str := "| "
		for i := len(sample.Location) - 1; i >= 0; i-- {
			loc := sample.Location[i]
			f := profiling.FindFunction(converted, loc.ID)
			str += " |> " + f.Name
		}
		fmt.Println(str)
	}

	_, err = profile.ParseData(data)
	require.NoError(t, err)
}

func TestEluded2(t *testing.T) {
	prof, err := profile.ParseData(profileExample)
	require.NoError(t, err)

	for _, sample := range prof.Sample {
		str := "| "
		for i := len(sample.Location) - 1; i >= 0; i-- {
			loc := sample.Location[i]
			f := profiling.FindFunction(prof, loc.ID)
			str += " |> " + f.Name
		}
		fmt.Println(str)
	}
	//
	//err = profiling.PyroscopePush(profile)
	//require.NoError(t, err)
}
