package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/coder/serpent"
)

func (r *Root) segment() *serpent.Command {
	var (
		cliOpts = new(cliWatcherConfig).SingleWatcher()
		segment int64
		shard   string
		pretty  bool
	)
	cmd := &serpent.Command{
		Use: "pull-segment",
		Options: serpent.OptionSet{
			serpent.Option{
				Name:          "pretty",
				Description:   "Pretty print JSON.",
				Required:      false,
				Flag:          "pretty",
				FlagShorthand: "",
				Default:       "",
				Value:         serpent.BoolOf(&pretty),
			},
			serpent.Option{
				Name:          "segment",
				Description:   "Which segment to pull.",
				Required:      true,
				Flag:          "segment",
				FlagShorthand: "",
				Default:       "",
				Value:         serpent.Int64Of(&segment),
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

			watcher := watchers[0]

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
		},
	}

	cliOpts.Attach(cmd)
	return cmd
}
