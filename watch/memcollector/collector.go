package memcollector

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

var _ prometheus.Collector = (*Collector)(nil)

type Collector struct {
	logger      zerolog.Logger
	namespace   string
	constLabels prometheus.Labels

	lastUpdated prometheus.Gauge
	metrics     atomic.Pointer[map[string][]prometheusMetric]
}

// New
// labels are the label constants on all metrics.
func New(logger zerolog.Logger, namespace string, labels prometheus.Labels) *Collector {
	return &Collector{
		logger:      logger,
		namespace:   namespace,
		constLabels: labels,
		lastUpdated: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "watcher",
			Name:        "last_updated_unix_s",
			Help:        "Timestamp in unix seconds of the last memory update.",
			ConstLabels: labels,
		}),
	}
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
			pm, err := prometheus.NewConstMetric(
				prometheus.NewDesc(
					fmt.Sprintf("%s_%s", c.namespace, k),
					"Metric from screeps memory segment.",
					descLabels,
					c.constLabels,
				),
				prometheus.GaugeValue,
				metric.Value,
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
}

func (c *Collector) SetMemory(memory json.RawMessage) (int, error) {
	metrics, err := memoryMetrics(memory)
	if err != nil {
		return 0, fmt.Errorf("read memory metrics: %w", err)
	}

	c.lastUpdated.Set(float64(time.Now().Unix()))
	c.metrics.Store(&metrics)

	count := 0
	for _, v := range metrics {
		count += len(v)
	}
	return count, nil
}
