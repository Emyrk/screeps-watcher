package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/coder/serpent"
)

func (r *Root) rooms() *serpent.Command {
	var (
		cliOpts = new(cliWatcherConfig)
		server  string
		segment int64
		shard   string
		pretty  bool
	)
	cmd := &serpent.Command{
		Use: "room",
		Options: serpent.OptionSet{
			serpent.Option{
				Name:        "room",
				Description: "Room name to download.",
				Required:    true,
				Flag:        "room",
			},
			serpent.Option{
				Name:        "pretty",
				Description: "Pretty print JSON.",
				Required:    false,
				Flag:        "pretty",
				Value:       serpent.BoolOf(&pretty),
			},
			serpent.Option{
				Name:        "server",
				Description: "Which server to pull from.",
				Required:    true,
				Flag:        "server",
				Value:       serpent.StringOf(&server),
			},
			serpent.Option{
				Name:        "shard",
				Description: "Which shard.",
				Required:    false,
				Flag:        "shard",
				Value:       serpent.StringOf(&shard),
			},
		},
		Handler: func(i *serpent.Invocation) error {
			logger := r.Logger(i)
			ctx := i.Context()

			watchers, err := configureWatchers(cliOpts, logger)
			if err != nil {
				return err
			}

			for _, watcher := range watchers {
				if watcher.Name != server {
					continue
				}
				if len(watcher.MemorySegments) == 1 {
					shard = watcher.MemorySegments[0].Shard
				}
				if shard == "" {
					return fmt.Errorf("must choose a --shard")
				}
				data, _, err := watcher.MemorySegment(ctx, int(segment), shard)
				if err != nil {
					return fmt.Errorf("fetch memory segment: %w", err)
				}

				if pretty {
					data, _ = json.MarshalIndent(data, "", "\t")
				}
				fmt.Println(string(data))
				return nil
			}

			return fmt.Errorf("not found")
		},
	}

	cliOpts.Attach(cmd)
	return cmd
}
