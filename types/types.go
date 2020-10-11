package types

import (
	"time"

	. "github.com/influxdata/line-protocol"
)

// Time() time.Time
// Name() string
// TagList() []*Tag
// FieldList() []*Field

type metric struct {
	name      string   // e.g. "cpu_usage"
	tags      []*Tag   // e.g. {"cpu": "cpu-total", "host": "host1", "region": "eu-west"}
	fields    []*Field // e.g. {"idle": 0.1, "busy": 0.9}
	timestamp time.Time
}

func NewMetric(name string, tags []*Tag, fields []*Field, Timestamp time.Time) Metric {
	return &metric{
		name:      name,
		fields:    fields,
		tags:      tags,
		timestamp: Timestamp,
	}
}

func (m *metric) Time() time.Time {
	return m.timestamp
}

func (m *metric) Name() string {
	return m.name
}

func (m *metric) TagList() []*Tag {
	return m.tags
}

func (m *metric) FieldList() []*Field {
	return m.fields
}
