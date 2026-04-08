package promtext

import (
	"bytes"

	"github.com/prometheus/common/expfmt"
)

// ParseMetricValue parses Prometheus text format and extracts the first sample for metricName.
// If the metric has labels, the first matching sample is used.
func ParseMetricValue(body []byte, metricName string) (float64, bool, error) {
	var parser expfmt.TextParser
	mf, err := parser.TextToMetricFamilies(bytes.NewReader(body))
	if err != nil {
		return 0, false, err
	}
	m := mf[metricName]
	if m == nil {
		return 0, false, nil
	}
	for _, mm := range m.Metric {
		if mm.Gauge != nil {
			return mm.Gauge.GetValue(), true, nil
		}
		if mm.Counter != nil {
			return mm.Counter.GetValue(), true, nil
		}
		if mm.Untyped != nil {
			return mm.Untyped.GetValue(), true, nil
		}
		if mm.Summary != nil && mm.Summary.SampleCount != nil {
			return float64(mm.Summary.GetSampleCount()), true, nil
		}
		if mm.Histogram != nil && mm.Histogram.SampleCount != nil {
			return float64(mm.Histogram.GetSampleCount()), true, nil
		}
	}
	return 0, false, nil
}
