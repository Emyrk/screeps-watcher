package cmd

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/Emyrk/screeps-watcher/watch"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"

	"github.com/coder/serpent"
)

func (r *Root) WatchCmd() *serpent.Command {
	var (
		cliOpts = new(cliWatcherConfig)
	)
	cmd := &serpent.Command{
		Use:     "watch",
		Options: serpent.OptionSet{},
		Handler: func(i *serpent.Invocation) error {
			logger := r.Logger(i)
			ctx := i.Context()

			watchers, err := configureWatchers(cliOpts, logger)
			if err != nil {
				return err
			}

			reg := prometheus.NewRegistry()
			for _, watcher := range watchers {
				go watcher.Watch(ctx)
				err := reg.Register(watcher)
				if err != nil {
					logger.Error().Err(err).Str("server", watcher.Name).Msg("register watcher")
					return fmt.Errorf("register watcher: %w", err)
				}
			}

			handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{
				Registry: reg,
			})

			//go func() {
			//	mux := http.NewServeMux()
			//	mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
			//	err := http.ListenAndServe(":6060", mux)
			//	if err != nil {
			//		logger.Error().Err(err).Msg("pprof")
			//	}
			//}()

			return http.ListenAndServe(":2112", handler)
		},
	}

	cliOpts.Attach(cmd)
	return cmd
}

type cliWatcherConfig struct {
	ConfigPath string
	Additional serpent.Struct[watch.WatcherOptions]

	single       bool
	SelectServer string
	SelectShard  string
}

// SingleWatcher the caller expects just 1 watcher to be returned.
// This is helpful for 1 off api calls.
func (c *cliWatcherConfig) SingleWatcher() *cliWatcherConfig {
	c.single = true
	return c
}

func (c *cliWatcherConfig) Attach(cmd *serpent.Command) {
	cmd.Options = append(cmd.Options, serpent.Option{
		Name:          "config",
		Description:   "YAML config file to use.",
		Required:      false,
		Flag:          "config",
		FlagShorthand: "c",
		Default:       "config.yaml",
		Value:         serpent.StringOf(&c.ConfigPath),
	},
		// Manual configuration for a single server. Used for 1 off commands.
		serpent.Option{
			Name:        "server-config",
			Description: "Full server configuration outside config yaml. Used for 1 off commands.",
			Required:    false,
			Flag:        "extra",
			Default:     "",
			Value:       &c.Additional,
		},
	)

	if c.single {
		cmd.Options = append(cmd.Options,
			serpent.Option{
				Name:          "server",
				Description:   "Which server to pull from.",
				Required:      false,
				Flag:          "server",
				FlagShorthand: "",
				Default:       "",
				Value:         serpent.StringOf(&c.SelectServer),
			},
			serpent.Option{
				Name:          "shard",
				Description:   "Which shard.",
				Required:      false,
				Flag:          "shard",
				FlagShorthand: "",
				Default:       "",
				Value:         serpent.StringOf(&c.SelectShard),
			})
	}
}

func configureWatchers(opts *cliWatcherConfig, logger zerolog.Logger) ([]*watch.Watcher, error) {
	watchConfigs := make([]watch.WatcherOptions, 0)
	if opts.Additional.Value.URL != "" {
		if opts.single {
			if opts.Additional.Value.Name == "" {
				opts.Additional.Value.Name = "manual"
			}
			if opts.SelectServer == "" {
				opts.SelectServer = opts.Additional.Value.Name
			}
		}
		watchConfigs = append(watchConfigs, opts.Additional.Value)
	}

	_, err := os.Stat(opts.ConfigPath)
	var config watch.WatchConfig
	// If the config exists, parse it.
	if !os.IsNotExist(err) {
		yamlData, err := os.ReadFile(opts.ConfigPath)
		if err != nil {
			logger.Error().Err(err).Str("config", opts.ConfigPath).Msg("read config")
			return nil, fmt.Errorf("read config: %w", err)
		}

		err = yaml.Unmarshal(yamlData, &config)
		if err != nil {
			logger.Error().Err(err).Str("config", opts.ConfigPath).Msg("unmarshal config")
			return nil, fmt.Errorf("unmarshal config: %w", err)
		}
	} else {
		if opts.Additional.Value.URL == "" {
			return nil, fmt.Errorf("config file does not exist: %s", opts.ConfigPath)
		}
	}

	allConfigs := append(watchConfigs, config.Servers...)
	watchers := make([]*watch.Watcher, 0, len(allConfigs))
	for _, server := range allConfigs {
		watcher, err := watch.New(config, server, logger.With().Str("service", "watcher").Logger())
		if err != nil {
			logger.Error().Err(err).Str("server", server.Name).Msg("new watcher")
			return nil, fmt.Errorf("new watcher: %w", err)
		}
		watchers = append(watchers, watcher)
	}

	if opts.single {
		if opts.SelectServer != "" {
			for _, w := range watchers {
				if w.Name == opts.SelectServer {
					watchers = []*watch.Watcher{w}
					break
				}
			}
		}

		// Ambiguous server selection.
		if len(watchers) > 1 {
			return nil, fmt.Errorf("more than 1 watcher found, must specify --server")
		}
	}

	logger.Info().
		Int("num_watchers", len(watchers)).
		Msg("watching")

	return watchers, nil
}
