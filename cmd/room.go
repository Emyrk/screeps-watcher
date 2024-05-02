package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Emyrk/screeps-watcher/watch"

	"github.com/coder/serpent"
)

func (r *Root) roomTerrain() *serpent.Command {
	cmd := r.rooms(func(w *watch.Watcher) func(ctx context.Context, room string, shard string) (json.RawMessage, error) {
		return w.RoomTerrain
	})

	cmd.Use = "room-terrain"
	cmd.Short = "Fetch room terrain data."
	return cmd
}

func (r *Root) roomObjects() *serpent.Command {
	cmd := r.rooms(func(w *watch.Watcher) func(ctx context.Context, room string, shard string) (json.RawMessage, error) {
		return w.RoomTerrain
	})

	cmd.Use = "room-objects"
	cmd.Short = "Fetch room objects data."
	return cmd
}

type roomAPICall = func(w *watch.Watcher) func(ctx context.Context, room string, shard string) (json.RawMessage, error)

func (r *Root) rooms(do roomAPICall) *serpent.Command {
	var (
		cliOpts = new(cliWatcherConfig).SingleWatcher()
		shard   string
		room    string
		pretty  bool
	)
	cmd := &serpent.Command{
		Use: "",
		Options: serpent.OptionSet{
			serpent.Option{
				Name:        "room",
				Description: "Room name to download.",
				Required:    true,
				Flag:        "room",
				Value:       serpent.StringOf(&room),
			},
			serpent.Option{
				Name:        "pretty",
				Description: "Pretty print JSON.",
				Required:    false,
				Flag:        "pretty",
				Value:       serpent.BoolOf(&pretty),
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

			if shard == "" && len(watcher.MemorySegments) == 1 {
				shard = watcher.MemorySegments[0].Shard
			}
			if shard == "" {
				return fmt.Errorf("must choose a --shard")
			}

			data, err := do(watcher)(ctx, room, shard)
			if err != nil {
				return fmt.Errorf("fetch room terrain: %w", err)
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
