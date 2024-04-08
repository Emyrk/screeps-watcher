package memcollector

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

func memoryMetrics(data json.RawMessage) (map[string][]prometheusMetric, error) {
	stats := make(map[string]interface{})
	err := json.Unmarshal(data, &stats)
	if err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	// Pull all the metrics from the memory segment.
	metrics := make(map[string][]prometheusMetric)
	for k, v := range stats {
		switch v := v.(type) {
		case map[string]interface{}:
			next(metrics, k, v)
		default:
			// Log an error?
			//return nil, fmt.Errorf("unknown type: %T", v)
		}
	}

	return metrics, nil
}

type prometheusMetric struct {
	Labels map[string]string
	Value  float64
}

var labels, _ = regexp.Compile(`\{[^}]*\}`)

func next(src map[string][]prometheusMetric, parent string, data map[string]interface{}) {
	for k, v := range data {
		// Create the parent metric name with all their labels.
		parent = strings.ReplaceAll(parent, ".", "_")
		metricName := fmt.Sprintf("%s_%s", parent, k)
		var value float64
		switch v := v.(type) {
		case map[string]interface{}:
			next(src, metricName, v)
			continue
		case int:
			value = float64(v)
		case int32:
			value = float64(v)
		case int64:
			value = float64(v)
		case float32:
			value = float64(v)
		case float64:
			value = v
		}
		rawLabels := labels.FindString(metricName)
		rawLabels = strings.Trim(rawLabels, "{}")
		metricName = labels.ReplaceAllString(metricName, "")

		labels := make(map[string]string)
		list := strings.Split(rawLabels, ",")
		for _, l := range list {
			parts := strings.Split(l, "=")
			if len(parts) != 2 {
				continue
			}
			labels[parts[0]] = parts[1]
		}
		src[metricName] = append(src[metricName], prometheusMetric{
			Labels: labels,
			Value:  value,
		})
	}
}
