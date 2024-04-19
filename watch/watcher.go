package watch

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Emyrk/screeps-watcher/watch/auth"
	"github.com/Emyrk/screeps-watcher/watch/memcollector"
	"github.com/Emyrk/screeps-watcher/watch/screepssocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

var _ prometheus.Collector = (*Watcher)(nil)

type WatcherOptions struct {
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Token    string `yaml:"token"`

	// Each target is a single scrape endpoint
	Targets           []Target      `yaml:"targets"`
	ScrapeInterval    time.Duration `yaml:"scrape_interval"`
	WebsocketChannels []string      `yaml:"websocket_channels"`
}

type Target struct {
	Shard       string            `yaml:"shard"`
	SegmentID   int               `yaml:"segment"`
	ConstLabels prometheus.Labels `yaml:"constant_labels"`

	collector *memcollector.Collector
}

// Watcher will watch a screeps server and it's configured shards for
// memory stats and logs.
type Watcher struct {
	Name    string
	URL     *url.URL
	Targets []*Target

	// TODO:
	AuthMethod auth.Method
	cli        *http.Client
	// rateLimitUntil is set when rate limits are hit.
	rateLimitUntil    time.Time
	logger            zerolog.Logger
	interval          time.Duration
	reg               *prometheus.Registry
	websocketChannels []string
}

func New(opts WatcherOptions, logger zerolog.Logger) (*Watcher, error) {
	if opts.URL == "" {
		return nil, fmt.Errorf("missing url field for server")
	}

	if len(opts.Targets) == 0 {
		return nil, fmt.Errorf("no targets configured for %q", opts.Name)
	}

	u, err := url.Parse(opts.URL)
	if err != nil {
		return nil, err
	}

	if opts.Password != "" && opts.Token != "" {
		return nil, fmt.Errorf("cannot provide both toke and password fields for %q", opts.Name)
	}

	var authMethod auth.Method
	if opts.Password != "" {
		authMethod = &auth.Password{
			Username: opts.Username,
			Password: opts.Password,
		}
	} else {
		authMethod = &auth.Token{
			Username:  opts.Username,
			AuthToken: opts.Token,
		}
	}

	if opts.ScrapeInterval == 0 {
		opts.ScrapeInterval = time.Minute
	}

	reg := prometheus.NewRegistry()

	tgts := make([]*Target, 0, len(opts.Targets))
	for _, t := range opts.Targets {
		if t.Shard == "" {
			t.Shard = "none"
		}
		constantLabels := prometheus.Labels{
			// Watcher labels
			"username": opts.Username,
			"server":   opts.Name,
			// Target label
			"shard": t.Shard,
		}
		for k, v := range t.ConstLabels {
			constantLabels[k] = v
		}

		tgt := &Target{
			Shard:     t.Shard,
			SegmentID: t.SegmentID,
			collector: memcollector.New(logger.
				With().
				Str("username", t.Shard).
				Str("shard", t.Shard).
				Logger(), "screeps_memory", constantLabels),
		}
		tgts = append(tgts, tgt)
		err := reg.Register(tgt.collector)
		if err != nil {
			return nil, fmt.Errorf("register tgt shard=%q", tgt.Shard)
		}
	}

	return &Watcher{
		Name:              opts.Name,
		URL:               u,
		Targets:           tgts,
		AuthMethod:        authMethod,
		cli:               http.DefaultClient,
		interval:          opts.ScrapeInterval,
		reg:               reg,
		websocketChannels: opts.WebsocketChannels,
		logger: logger.With().
			Str("username", opts.Username).
			Str("server", opts.Name).
			Logger(),
	}, nil
}

func (w *Watcher) Describe(descs chan<- *prometheus.Desc) {
	w.reg.Describe(descs)
}

func (w *Watcher) Collect(metrics chan<- prometheus.Metric) {
	w.reg.Collect(metrics)
}

func (w *Watcher) Watch(ctx context.Context) {
	go w.WatchMemory(ctx)
	go w.WatchWebsocket(ctx)
}

func (w *Watcher) WatchWebsocket(ctx context.Context) {
	sock, err := screepssocket.New(ctx, w.URL, w.logger, w.cli, w.AuthMethod, w.websocketChannels, prometheus.Labels{
		"server":   w.Name,
		"username": w.AuthMethod.GetUsername(),
	})
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to create websocket")
		return
	}

	go sock.Run(ctx)
	w.reg.MustRegister(sock)
}

func (w *Watcher) WatchMemory(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	for {
		if w.rateLimitUntil.After(time.Now()) {
			// Skipping due to rate limit
			w.logger.Warn().Time("reset", w.rateLimitUntil).Msg("rate limit hit, skipping scrape")
			continue
		}

		for _, target := range w.Targets {
			count, size := w.scrapeTarget(ctx, target)
			w.logger.Info().
				Int("segment_size", size).
				Int("segment", target.SegmentID).
				Str("shard", target.Shard).
				Int("metric_count", count).
				Msg("scrape target complete")
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

func (w *Watcher) scrapeTarget(ctx context.Context, target *Target) (int, int) {
	logger := w.logger.With().
		Str("shard", target.Shard).
		Int("segment", target.SegmentID).Logger()

	data, size, err := w.MemorySegment(ctx, target.SegmentID, target.Shard)
	if err != nil {
		logger.Error().Msg("failed to get memory segment")
		return 0, size
	}

	count, err := target.collector.SetMemory(data)
	if err != nil {
		logger.Error().
			Err(err).
			Int("decoded_size", size).
			Msg("failed to set memory metrics")
	}
	return count, size
}
