package memcollector

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var _ prometheus.Collector = (*Collector)(nil)

type Collector struct {
	namespace   string
	constLabels prometheus.Labels

	lastUpdated prometheus.Gauge
	metrics     atomic.Pointer[map[string][]prometheusMetric]
}

// New
// labels are the label constants on all metrics.
func New(namespace string, labels prometheus.Labels) *Collector {
	return &Collector{
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
			labels := make([]string, 0)
			for lk, lv := range metric.Labels {
				labels = append(labels, lk, lv)
			}
			pm, err := prometheus.NewConstMetric(
				prometheus.NewDesc(
					fmt.Sprintf("%s_%s", c.namespace, k),
					"Metric from screeps memory segment.",
					nil,
					c.constLabels,
				),
				prometheus.GaugeValue,
				metric.Value,
				labels...,
			)
			if err != nil {
				// Log?
				continue
			}
			ch <- pm
		}
	}

	ch <- c.lastUpdated
}

func (c *Collector) SetMemory(memory json.RawMessage) error {
	metrics, err := memoryMetrics(memory)
	if err != nil {
		return fmt.Errorf("read memory metrics: %w", err)
	}

	c.lastUpdated.Set(float64(time.Now().Unix()))
	c.metrics.Store(&metrics)
	return nil
}
