package watch

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Emyrk/screeps-watcher/watch/auth"
	"github.com/Emyrk/screeps-watcher/watch/market"
	"github.com/Emyrk/screeps-watcher/watch/memcollector"
	"github.com/Emyrk/screeps-watcher/watch/profiling"
	"github.com/Emyrk/screeps-watcher/watch/screepssocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

var _ prometheus.Collector = (*Watcher)(nil)

type WatchConfig struct {
	Pyroscope PyroscopeSettings `yaml:"pyroscope"`
	Servers   []WatcherOptions  `yaml:"servers"`
}

type PyroscopeSettings struct {
	Address string `yaml:"address"`
}

type WatcherOptions struct {
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Token    string `yaml:"token"`

	// Each target is a single scrape endpoint
	MemorySegments    []MemoryTargets `yaml:"targets"`
	Markets           []MarketTargets `yaml:"markets"`
	MetricsInterval   time.Duration   `yaml:"metrics_scrape_interval"`
	MarketInterval    time.Duration   `yaml:"market_scrape_interval"`
	WebsocketChannels []string        `yaml:"websocket_channels"`
}

type ProfileTarget struct {
	Shard     string `yaml:"shard"`
	SegmentID int    `yaml:"segment"`
}

type MemoryTargets struct {
	Shard       string            `yaml:"shard"`
	Metrics     *int              `yaml:"metrics_segment"`
	Profile     *int              `yaml:"profile_segment"`
	ConstLabels prometheus.Labels `yaml:"constant_labels"`

	serverName string
	collector  *memcollector.Collector
}

func (m MemoryTargets) MetricSegment() int {
	if m.Metrics == nil {
		return -1
	}
	return *m.Metrics
}

func (m MemoryTargets) ProfileSegment() int {
	if m.Profile == nil {
		return -1
	}
	return *m.Profile
}

type MarketTargets struct {
	ResourceType string `yaml:"resource_type"`
	Shard        string `yaml:"shard"`
}

// Watcher will watch a screeps server and it's configured shards for
// memory stats and logs.
type Watcher struct {
	Name           string
	Username       string
	URL            *url.URL
	MemorySegments []*MemoryTargets
	Markets        []MarketTargets

	// TODO:
	AuthMethod auth.Method
	cli        *http.Client

	logger            zerolog.Logger
	memoryInterval    time.Duration
	profileInterval   time.Duration
	marketInterval    time.Duration
	reg               *prometheus.Registry
	websocketChannels []string

	// For backing off rate limits
	memorySegmentRateLimitUntil time.Time
	marketApiRateLimitUntil     time.Time
}

func New(global WatchConfig, opts WatcherOptions, logger zerolog.Logger) (*Watcher, error) {
	if opts.URL == "" {
		return nil, fmt.Errorf("missing url field for server")
	}

	if len(opts.MemorySegments) == 0 {
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

	if opts.MetricsInterval == 0 {
		opts.MetricsInterval = time.Minute
	}

	if opts.MarketInterval == 0 {
		opts.MarketInterval = time.Hour * 4
	}

	reg := prometheus.NewRegistry()
	var pusher *profiling.PyroscopePusher
	if global.Pyroscope.Address != "" {
		pusher, err = profiling.NewPusher(global.Pyroscope.Address, logger.With().Str("server", "pyroscope_pusher").Logger())
		if err != nil {
			return nil, fmt.Errorf("could not create profiling pusher: %w", err)
		}
	}

	tgts := make([]*MemoryTargets, 0, len(opts.MemorySegments))
	for _, t := range opts.MemorySegments {
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

		tgt := &MemoryTargets{
			Shard:      t.Shard,
			Metrics:    t.Metrics,
			Profile:    t.Profile,
			serverName: opts.Name,
			collector: memcollector.New(logger.
				With().
				Str("username", t.Shard).
				Str("shard", t.Shard).
				Logger(), "screeps_memory", constantLabels).WithPusher(pusher),
		}
		tgts = append(tgts, tgt)
		err := reg.Register(tgt.collector)
		if err != nil {
			return nil, fmt.Errorf("register tgt shard=%q", tgt.Shard)
		}
	}

	return &Watcher{
		Name:              opts.Name,
		Username:          opts.Username,
		URL:               u,
		MemorySegments:    tgts,
		Markets:           opts.Markets,
		AuthMethod:        authMethod,
		cli:               http.DefaultClient,
		memoryInterval:    opts.MetricsInterval,
		marketInterval:    opts.MarketInterval,
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
	go w.WatchMetrics(ctx)
	go w.WatchMarket(ctx)
	go w.WatchWebsocket(ctx)
}

func (w *Watcher) WatchWebsocket(ctx context.Context) {
	if len(w.websocketChannels) == 0 {
		w.logger.Info().Msg(fmt.Sprintf("no websocket channels configured for server %s, skipping", w.Name))
		return
	}

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

func (w *Watcher) WatchMetrics(ctx context.Context) {
	ticker := time.NewTicker(w.memoryInterval)
	logger := w.logger.With().Str("data", "metrics-memory-segment").Logger()
	for {
		if w.memorySegmentRateLimitUntil.After(time.Now()) {
			// Skipping due to rate limit
			logger.Warn().Time("reset", w.memorySegmentRateLimitUntil).Msg("rate limit hit, skipping scrape")
			continue
		}

		for _, target := range w.MemorySegments {
			var metricCount, metricSize = -1, -1
			var profileCount, profileSize = -1, -1
			if target.MetricSegment() >= 0 {
				metricCount, metricSize = w.scrapeMetrics(ctx, target)
			}
			if target.ProfileSegment() >= 0 {
				profileCount, profileSize = w.scrapeProfile(ctx, target)
			}
			logger.Info().
				Int("metric_segment_size", metricSize).
				Int("metric_count", metricCount).
				Int("profile_segment_size", profileSize).
				Int("profile_count", profileCount).
				Int("metrics_segment", target.MetricSegment()).
				Int("profile_segment", target.ProfileSegment()).
				Str("shard", target.Shard).
				Msg("scrape target complete")
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

func (w *Watcher) WatchMarket(ctx context.Context) {
	if len(w.Markets) == 0 {
		w.logger.Info().Msg("no market targets configured, skipping market scrape")
		return
	}

	marketStatsAvgPrice := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "screeps",
		Subsystem: "market",
		Name:      "resource_daily_avg_price",
		Help:      "Average price of the resource for the day",
		ConstLabels: prometheus.Labels{
			"username": w.Username,
			"server":   w.Name,
		},
	}, []string{
		"resource_type", "shard",
	})

	marketStatsStdDevPrice := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "screeps",
		Subsystem: "market",
		Name:      "resource_daily_std_dev_price",
		Help:      "Standard Deviation of the resource for the day",
		ConstLabels: prometheus.Labels{
			"username": w.Username,
			"server":   w.Name,
		},
	}, []string{
		"resource_type", "shard",
	})

	marketStatsTransactionCount := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "screeps",
		Subsystem: "market",
		Name:      "resource_daily_transaction_count",
		Help:      "Total transactions for the day",
		ConstLabels: prometheus.Labels{
			"username": w.Username,
			"server":   w.Name,
		},
	}, []string{
		"resource_type", "shard",
	})

	marketStatsVolume := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "screeps",
		Subsystem: "market",
		Name:      "resource_daily_volume",
		Help:      "Total volume",
		ConstLabels: prometheus.Labels{
			"username": w.Username,
			"server":   w.Name,
		},
	}, []string{
		"resource_type", "shard",
	})

	w.reg.MustRegister(marketStatsAvgPrice)
	w.reg.MustRegister(marketStatsStdDevPrice)
	w.reg.MustRegister(marketStatsTransactionCount)
	w.reg.MustRegister(marketStatsVolume)

	ticker := time.NewTicker(w.marketInterval)
	logger := w.logger.With().Str("data", "market").Logger()
	for {
		if w.marketApiRateLimitUntil.After(time.Now()) {
			// Skipping due to rate limit
			logger.Warn().Time("reset", w.marketApiRateLimitUntil).Msg("rate limit hit, skipping scrape")
			continue
		}

		for _, target := range w.Markets {
			logger = logger.With().Str("resource_type", target.ResourceType).Str("shard", target.Shard).Logger()
			stat, err := w.scrapeMarket(ctx, &target)
			if err != nil {
				logger.Err(err).
					Msg("failed to scrape market")
				continue
			}

			marketStatsAvgPrice.WithLabelValues(stat.ResourceType, target.Shard).Set(stat.AvgPrice)
			marketStatsStdDevPrice.WithLabelValues(stat.ResourceType, target.Shard).Set(stat.StddevPrice)
			marketStatsTransactionCount.WithLabelValues(stat.ResourceType, target.Shard).Set(float64(stat.Transactions))
			marketStatsVolume.WithLabelValues(stat.ResourceType, target.Shard).Set(float64(stat.Volume))
		}
		logger.Info().Msg("scrape markets complete")

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

func (w *Watcher) scrapeMarket(ctx context.Context, target *MarketTargets) (*market.Stats, error) {
	data, err := w.Market(ctx, target.ResourceType, target.Shard)
	if err != nil {
		return nil, fmt.Errorf("failed to get market data: %w", err)
	}

	stats, err := market.ParseMarketResponse(data)
	if err != nil {
		return nil, fmt.Errorf("parse market data: %w", err)
	}
	return stats.Today()
}

func (w *Watcher) scrapeProfile(ctx context.Context, target *MemoryTargets) (int, int) {
	logger := w.logger.With().
		Str("shard", target.Shard).
		Int("segment", target.ProfileSegment()).Logger()

	if !target.collector.SupportsProfiling() {
		logger.Error().Msg("profile collector not supported")
	}

	data, size, err := w.MemorySegment(ctx, target.ProfileSegment(), target.Shard)
	if err != nil {
		logger.Error().Msg("failed to get profile memory segment")
		return 0, size
	}

	count, err := target.collector.SetProfileMemory(fmt.Sprintf("screeps_%s_%s", target.serverName, target.Shard), data)
	if err != nil {
		logger.Error().
			Err(err).
			Int("decoded_size", size).
			Msg("failed to set memory metrics")
	}
	return count, size
}

func (w *Watcher) scrapeMetrics(ctx context.Context, target *MemoryTargets) (int, int) {
	logger := w.logger.With().
		Str("shard", target.Shard).
		Int("segment", target.MetricSegment()).Logger()

	data, size, err := w.MemorySegment(ctx, target.MetricSegment(), target.Shard)
	if err != nil {
		logger.Error().Msg("failed to get metric memory segment")
		return 0, size
	}

	count, err := target.collector.SetMetricMemory(data)
	if err != nil {
		logger.Error().
			Err(err).
			Int("decoded_size", size).
			Msg("failed to set memory metrics")
	}
	return count, size
}
