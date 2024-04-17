package cmd

import (
	"bytes"
	"fmt"
	"runtime/pprof"

	"github.com/Emyrk/screeps-watcher/cmd/workdemo"

	"github.com/coder/serpent"
)

func (r *Root) pprofDemo() *serpent.Command {
	return &serpent.Command{
		Use: "pprofdemo",
		Handler: func(i *serpent.Invocation) error {
			var buf bytes.Buffer
			err := pprof.StartCPUProfile(&buf)
			if err != nil {
				return fmt.Errorf("start cpu profile: %w", err)
			}

			// Do some work
			workdemo.Root()

			// Stop profile
			pprof.StopCPUProfile()

			// Write the profile to output
			_, err = buf.WriteTo(i.Stdout)
			return err
		},
	}
}
