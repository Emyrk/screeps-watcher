package cmd

import (
	"fmt"
	"net/http"
	"os"

	"github.com/Emyrk/screeps-watcher/watch"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v3"

	"github.com/coder/serpent"
)

type WatchConfig struct {
	Servers []watch.WatcherOptions `yaml:"servers"`
}

func (r *Root) WatchCmd() *serpent.Command {
	var (
		configPath string
	)
	return &serpent.Command{
		Use: "watch",
		Options: serpent.OptionSet{
			serpent.Option{
				Name:          "config",
				Description:   "YAML config file to use.",
				Required:      false,
				Flag:          "config",
				FlagShorthand: "c",
				Default:       "config.yaml",
				Value:         serpent.StringOf(&configPath),
			},
		},
		Handler: func(i *serpent.Invocation) error {
			logger := r.Logger(i)
			ctx := i.Context()

			yamlData, err := os.ReadFile(configPath)
			if err != nil {
				logger.Error().Err(err).Str("config", configPath).Msg("read config")
				return fmt.Errorf("read config: %w", err)
			}

			var config WatchConfig
			err = yaml.Unmarshal(yamlData, &config)
			if err != nil {
				logger.Error().Err(err).Str("config", configPath).Msg("unmarshal config")
				return fmt.Errorf("unmarshal config: %w", err)
			}

			watchers := make([]*watch.Watcher, 0, len(config.Servers))
			for _, server := range config.Servers {
				watcher, err := watch.New(server, logger.With().Str("service", "watcher").Logger())
				if err != nil {
					logger.Error().Err(err).Str("server", server.Name).Msg("new watcher")
					return fmt.Errorf("new watcher: %w", err)
				}
				watchers = append(watchers, watcher)
			}

			logger.Info().
				Int("num_watchers", len(watchers)).
				Msg("watching")

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
			return http.ListenAndServe(":2112", handler)
		},
	}
}
