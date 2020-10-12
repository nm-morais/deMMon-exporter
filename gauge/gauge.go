package gauge

import (
	"sync"
	"time"

	lineProto "github.com/influxdata/line-protocol"
)

type GaugeOpts struct {
	Batch              bool
	Name               string           // "e.g" cpu_temperature_celsius
	Tags               []*lineProto.Tag // e.g. {"cpu": "cpu-total", "host": "host1", "region": "eu-west"}
	EmmissionFrequency time.Duration
}

type Gauge struct {
	mu sync.RWMutex
	GaugeOpts
	fields []*lineProto.Field // e.g. {"idle": 0.1, "busy": 0.9}
}

func NewGauge(opts GaugeOpts, firstValue []*lineProto.Field) *Gauge {
	return &Gauge{
		mu:        sync.RWMutex{},
		GaugeOpts: opts,
		fields:    firstValue,
	}
}

func (g *Gauge) Set(fields []*lineProto.Field) {
	g.mu.Lock()
	g.fields = fields
	g.mu.Unlock()
}

func (g *Gauge) Get() []*lineProto.Field {
	g.mu.Lock()
	defer g.mu.Unlock()
	fieldsCopy := make([]*lineProto.Field, 0, len(g.fields))
	for _, f := range g.fields {
		fCopy := *f
		fieldsCopy = append(fieldsCopy, &fCopy)
	}
	return fieldsCopy
}
