package exporter

import (
	"io"
	"time"

	"github.com/nm-morais/demmon-common/exporters"
	"github.com/nm-morais/go-babel/pkg/peer"
	"github.com/nm-morais/go-babel/pkg/protocol"
	"github.com/nm-morais/go-babel/pkg/protocolManager"
)

type ExporterConf struct {
	ImporterAddr    peer.Peer
	MaxRedials      int
	RedialTimeout   time.Duration
	ExportFrequency time.Duration
}

type Exporter struct {
	proto protocol.Protocol
	confs ExporterConf
	set   *exporters.Set
}

func New(confs ExporterConf, babel protocolManager.ProtocolManager) *Exporter {
	e := &Exporter{
		confs: confs,
		set:   exporters.NewSet(),
	}
	e.proto = NewExporterProto(confs, e, babel)
	return e
}

// Proto returns the babel proto of the exporter.
func (e *Exporter) Proto() protocol.Protocol {
	return e.proto
}

// Proto returns the babel proto of the exporter.
func (e *Exporter) NewCounter(name string) *exporters.Counter {
	return e.set.NewCounter(name)
}

// Proto returns the babel proto of the exporter.
func (e *Exporter) NewGauge(name string, f func() float64) *exporters.Gauge {
	return e.set.NewGauge(name, f)
}

// Proto returns the babel proto of the exporter.
func (e *Exporter) NewHistogram(name string) *exporters.Histogram {
	return e.set.NewHistogram(name)
}

func (e *Exporter) WriteMetrics(w io.Writer) {
	e.set.WriteMetrics(w)
}
