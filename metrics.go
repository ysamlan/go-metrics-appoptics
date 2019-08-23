package appoptics

import (
	"fmt"
	"github.com/rcrowley/go-metrics"
	"log"
	"regexp"
	"sort"
	"strings"
)

var tagNameRegex = regexp.MustCompile(`[^-.:_\w]`)
var tagValueRegex = regexp.MustCompile(`[^-.:_\\/\w\? ]`)

type metric struct {
	name string
	tags map[string]string
	sampleFunc func() metrics.Sample
}

func Metric(name string) *metric {
	return &metric{name: name, tags: map[string]string{}}
}

func (m *metric) Tag(name string, value interface{}) *metric {
	tagName := sanitizeTagName(name)
	tagValue := sanitizeTagValue(fmt.Sprintf("%v", value))

	if tagName == "" || tagValue == "" {
		log.Printf("Empty tag name or value: name=%v value=%v", tagName, tagValue)
		return m
	}

	m.tags[tagName] = tagValue
	return m
}

func (m *metric) WithSample(s func() metrics.Sample) *metric {
	m.sampleFunc = s
	return m
}

func (m *metric) String() string {
	sb := strings.Builder{}

	sb.WriteString(m.name)

	if len(m.tags) > 0 {
		sb.WriteString("#")
	}

	// Sort tag map for consistent ordering in encoded string
	var keys []string
	for key := range m.tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for i, key := range keys {
		if i != 0 {
			sb.WriteString(",")
		}
		sb.WriteString(key + "=" + m.tags[key])
	}

	return sb.String()
}

func (m *metric) Counter() metrics.Counter {
	return metrics.GetOrRegisterCounter(m.String(), metrics.DefaultRegistry)
}

func (m *metric) Meter() metrics.Meter {
	return metrics.GetOrRegisterMeter(m.String(), metrics.DefaultRegistry)
}

func (m *metric) Timer() metrics.Timer {
	return metrics.GetOrRegisterTimer(m.String(), metrics.DefaultRegistry)
}

func (m *metric) Histogram() metrics.Histogram {
	var sample func() metrics.Sample
	if m.sampleFunc != nil {
		sample = m.sampleFunc
	} else {
		sample = func() metrics.Sample {
			return metrics.NewExpDecaySample(1028, 0.015)
		}
	}

	return metrics.GetOrRegister(m.String(), func() metrics.Histogram {return metrics.NewHistogram(sample())}).(metrics.Histogram)
}

func (m *metric) Gauge() metrics.Gauge {
	return metrics.GetOrRegisterGauge(m.String(), metrics.DefaultRegistry)
}

func (m *metric) Gauge64() metrics.GaugeFloat64 {
	return metrics.GetOrRegisterGaugeFloat64(m.String(), metrics.DefaultRegistry)
}

// decodeMetricName decodes the metricName#a=foo,b=bar format and returns the metric name
// as a string and the tags as a map
func decodeMetricName(encoded string) (string, map[string]string) {
	split := strings.SplitN(encoded, "#", 2)
	name := split[0]
	if len(split) == 1 {
		return name, map[string]string{}
	}

	tagPart := split[1]
	pairs := strings.Split(tagPart, ",")

	tags := map[string]string{}
	for _, pair := range pairs {
		pairList := strings.SplitN(pair, "=", 2)
		if len(pairList) != 2 {
			log.Printf("Tag name `%v` is missing its value", pairList[0])
			continue
		}
		tags[pairList[0]] = pairList[1]
	}

	return name, tags
}

func sanitizeTagName(value string) string {
	if len(value) > 64 {
		value = value[:64]
	}
	value = strings.ToLower(value)
	return tagNameRegex.ReplaceAllString(value, "_")
}

func sanitizeTagValue(value string) string {
	if len(value) > 255 {
		value = value[:252] + "..."
	}
	value = strings.ToLower(value)
	return tagValueRegex.ReplaceAllString(value, "_")
}
