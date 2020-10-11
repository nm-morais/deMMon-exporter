package exporter

import (
	"time"

	babel "github.com/nm-morais/go-babel/pkg"
	"github.com/nm-morais/go-babel/pkg/errors"
	"github.com/nm-morais/go-babel/pkg/logs"
	"github.com/nm-morais/go-babel/pkg/message"
	"github.com/nm-morais/go-babel/pkg/notification"
	"github.com/nm-morais/go-babel/pkg/peer"
	"github.com/nm-morais/go-babel/pkg/protocol"
	stream "github.com/nm-morais/go-babel/pkg/stream"
	"github.com/nm-morais/go-babel/pkg/timer"
	"github.com/sirupsen/logrus"

	lineProto "github.com/influxdata/line-protocol"

	"github.com/nm-morais/demmon-metrics-client/gauge"
	"github.com/nm-morais/demmon-metrics-client/types"
)

const (
	exporterProtoID = 100
	importerProtoID = 101
	name            = "exporter"
)

type ExporterConf struct {
	RedialTimeout time.Duration
	MaxRedials    int

	target peer.Peer

	DatabaseName           string
	Username               string
	Password               string
	MaxBatchSize           uint64
	BatchEmissionFrequency time.Duration
}

type Exporter struct {
	confs  ExporterConf
	logger *logrus.Logger

	failedDials int

	gauges []*gauge.Gauge

	currBatch []lineProto.Metric
	idx       int
}

func New(confs ExporterConf) *Exporter {
	return &Exporter{
		confs:       confs,
		logger:      logs.NewLogger(name),
		currBatch:   make([]lineProto.Metric, confs.MaxBatchSize),
		gauges:      []*gauge.Gauge{},
		failedDials: 0,
		idx:         0,
	}
}

func (e *Exporter) MessageDelivered(message message.Message, peer peer.Peer) {}

func (e *Exporter) MessageDeliveryErr(message message.Message, peer peer.Peer, error errors.Error) {
}

func (e *Exporter) handleFlushTimer(timer timer.Timer) {
	metricMessage := NewMetricMessage(e.currBatch[:e.idx])
	babel.SendMessage(metricMessage, e.confs.target, importerProtoID, []protocol.ID{exporterProtoID})
	for i := 0; i < e.idx; i++ {
		e.currBatch[i] = nil
	}
	e.idx = 0
}

func (e *Exporter) handleRedialTimer(timer timer.Timer) {
	babel.Dial(e.confs.target, e.ID(), stream.NewTCPDialer())
}

func (e *Exporter) handleMetricNotification(notification notification.Notification) {
	metricNotification := notification.(metricNotification)
	if !metricNotification.Batch {
		metricMessage := NewMetricMessage([]lineProto.Metric{metricNotification.Metric})
		babel.SendMessage(metricMessage, e.confs.target, importerProtoID, []protocol.ID{exporterProtoID})
		return
	}
	e.currBatch = append(e.currBatch, metricNotification.Metric)
	e.idx++
}

func (e *Exporter) ID() protocol.ID {
	return exporterProtoID
}

func (e *Exporter) Name() string {
	return name
}

func (e *Exporter) Logger() *logrus.Logger {
	return e.logger
}

func (e *Exporter) Init() {
	babel.RegisterNotificationHandler(e.ID(), metricNotification{}, e.handleMetricNotification)
	babel.RegisterTimerHandler(e.ID(), redialTimerID, e.handleRedialTimer)
	babel.RegisterTimerHandler(e.ID(), flushTimerID, e.handleFlushTimer)
}

func (e *Exporter) Start() {
	for _, g := range e.gauges {
		go e.handleGauge(g)
	}
	babel.Dial(e.confs.target, e.ID(), stream.NewTCPDialer())
}

func (e *Exporter) DialFailed(p peer.Peer) {
	e.failedDials++

	if e.failedDials > e.confs.MaxRedials {
		e.logger.Panicln("Could not dial importer")
	}

	babel.RegisterTimer(e.ID(), NewRedialTimer(e.confs.RedialTimeout))
}

func (e *Exporter) DialSuccess(sourceProto protocol.ID, peer peer.Peer) bool {
	return sourceProto == e.ID()
}

func (e *Exporter) InConnRequested(peer peer.Peer) bool {
	return false
}

func (e *Exporter) OutConnDown(peer peer.Peer) {
	e.logger.Panic("local conn down")
}

func (e *Exporter) RegisterGauge(g *gauge.Gauge) {
	e.gauges = append(e.gauges, g)
}

func (e *Exporter) handleGauge(g *gauge.Gauge) {
	gaugeTicker := time.NewTicker(g.EmmissionFrequency)
	for {
		<-gaugeTicker.C
		babel.SendNotification(NewMetricNotification(types.NewMetric(g.Name, g.Tags, g.Get(), time.Now()), g.Batch))
	}
}
