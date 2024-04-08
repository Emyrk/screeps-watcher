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

// next recursively walks the JSON data and extracts all the metrics.
// One decision made is that the metric name will have all its labels at the
// final recursion level. An alternative would be to parse them, and pass the
// parsed labels down the recursion chain. That would be more efficient, but
// this was easier to debug with all the data in one place.
// And this scrape interval is infrequent, so the performance hit does
// not matter.
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
		// Each metric level can have labels.
		// So find all the labels and remove them from the metric name.
		rawLabels := labels.FindAllString(metricName, -1)
		// Remove the labels from the metric name.
		metricName = labels.ReplaceAllString(metricName, "")

		// handle extracted labels
		labelList := make([]string, 0, len(rawLabels))
		for _, labels := range rawLabels {
			// Get it down to just the key=value pairs.
			labels = strings.Trim(labels, "{}")
			// Append key=value pairs to the list.
			labelList = append(labelList, strings.Split(labels, ",")...)
		}

		labels := make(map[string]string)
		for _, l := range labelList {
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
