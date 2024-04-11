package cmd

import (
	"github.com/Emyrk/screeps-watcher/watch/profile"

	"github.com/coder/serpent"
)

func (r *Root) pprofDemo() *serpent.Command {
	return &serpent.Command{
		Use: "pprofdemo",
		Handler: func(i *serpent.Invocation) error {
			profile.DemoPprof()
			return nil
		},
	}
}
