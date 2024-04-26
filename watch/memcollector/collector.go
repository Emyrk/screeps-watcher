package memcollector

import (
	"encoding/json"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"github.com/Emyrk/screeps-watcher/watch/profiling"
	"github.com/Emyrk/screeps-watcher/watch/profiling/eluded"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

var _ prometheus.Collector = (*Collector)(nil)

type Collector struct {
	logger      zerolog.Logger
	namespace   string
	constLabels prometheus.Labels

	metricCount prometheus.Gauge
	segmentSize *prometheus.GaugeVec
	lastUpdated prometheus.Gauge
	metrics     atomic.Pointer[map[string][]prometheusMetric]
	now         func() time.Time

	profilePusher *profiling.PyroscopePusher
}

// New
// labels are the label constants on all metrics.
func New(logger zerolog.Logger, namespace string, labels prometheus.Labels) *Collector {
	return &Collector{
		logger:      logger,
		namespace:   namespace,
		constLabels: labels,
		now:         time.Now,
		segmentSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "watcher",
			Name:        "segment_size",
			Help:        "Size of the memory segment in bytes.",
			ConstLabels: labels,
		}, []string{"type"}),
		lastUpdated: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "watcher",
			Name:        "last_updated_unix_s",
			Help:        "Timestamp in unix seconds of the last memory update.",
			ConstLabels: labels,
		}),
		metricCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "watcher",
			Name:        "metric_count",
			Help:        "Number of metrics in the memory segment.",
			ConstLabels: labels,
		}),
	}
}

func (c *Collector) SupportsProfiling() bool {
	return c.profilePusher != nil
}

func (c *Collector) WithPusher(pusher *profiling.PyroscopePusher) *Collector {
	c.profilePusher = pusher
	return c
}

func (c *Collector) SetNow(f func() time.Time) {
	c.now = f
}

func (c *Collector) Describe(descs chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, descs)
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	metrics := c.metrics.Load()
	if metrics == nil {
		return
	}

	for k, v := range *metrics {
		for _, metric := range v {
			descLabels := make([]string, 0)
			labelValues := make([]string, 0)
			for lk, lv := range metric.Labels {
				labelValues = append(labelValues, lv)
				descLabels = append(descLabels, lk)
			}

			// Truncating to get consistency for tests.
			value := math.Trunc(metric.Value*10000) / 10000
			pm, err := prometheus.NewConstMetric(
				prometheus.NewDesc(
					fmt.Sprintf("%s_%s", c.namespace, k),
					"Metric from screeps memory segment.",
					descLabels,
					c.constLabels,
				),
				prometheus.GaugeValue,
				value,
				labelValues...,
			)
			if err != nil {
				c.logger.Warn().
					Str("metric_name", k).
					Strs("labels", labelValues).
					Err(err).
					Msg("failed to create metric")
				continue
			}
			ch <- pm
		}
	}

	ch <- c.lastUpdated
	c.segmentSize.Collect(ch)
	ch <- c.metricCount
}

func (c *Collector) SetMetricMemory(memory json.RawMessage) (int, error) {
	c.segmentSize.WithLabelValues(fmt.Sprintf("metrics")).Set(float64(len(memory)))

	metrics, err := memoryMetrics(memory)
	if err != nil {
		return 0, fmt.Errorf("read memory metrics: %w", err)
	}

	count := 0
	for _, v := range metrics {
		count += len(v)
	}

	c.metricCount.Set(float64(count))
	c.lastUpdated.Set(float64(c.now().Unix()))
	c.metrics.Store(&metrics)
	return count, nil
}

func (c *Collector) SetProfileMemory(name string, memory json.RawMessage) (int, error) {
	c.segmentSize.WithLabelValues(fmt.Sprintf("profile")).Set(float64(len(memory)))

	var profile []eluded.Profile
	err := json.Unmarshal(memory, &profile)
	if err != nil {
		return -1, fmt.Errorf("failed to unmarshal memory profile: %w", err)
	}

	// convert to pprof
	proto := profiling.New().Convert(profile)
	err = c.profilePusher.Push(name, proto)
	return len(proto.Sample), err
}
