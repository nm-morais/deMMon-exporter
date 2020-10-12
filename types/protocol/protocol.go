package prototypes

import (
	"time"

	babel "github.com/nm-morais/go-babel/pkg"
	"github.com/nm-morais/go-babel/pkg/errors"
	"github.com/nm-morais/go-babel/pkg/logs"
	"github.com/nm-morais/go-babel/pkg/message"
	"github.com/nm-morais/go-babel/pkg/notification"
	"github.com/nm-morais/go-babel/pkg/peer"
	"github.com/nm-morais/go-babel/pkg/protocol"
	"github.com/nm-morais/go-babel/pkg/stream"
	"github.com/nm-morais/go-babel/pkg/timer"
	"github.com/sirupsen/logrus"
)

const (
	exporterProtoID = 100
	importerProtoID = 101
	name            = "exporter"
)

type ExporterProtoConf struct {
	ImporterAddr  peer.Peer
	MaxRedials    int
	RedialTimeout time.Duration
}

type ExporterProto struct {
	confs       ExporterProtoConf
	logger      *logrus.Logger
	failedDials int
}

func NewExporterProto(confs ExporterProtoConf) *ExporterProto {
	return &ExporterProto{
		confs:       confs,
		failedDials: 0,
		logger:      logs.NewLogger(name),
	}
}

func (e *ExporterProto) MessageDelivered(message message.Message, peer peer.Peer) {}

func (e *ExporterProto) MessageDeliveryErr(message message.Message, peer peer.Peer, error errors.Error) {
}

func (e *ExporterProto) handleRedialTimer(timer timer.Timer) {
	babel.Dial(e.confs.ImporterAddr, e.ID(), stream.NewTCPDialer())
}

func (e *ExporterProto) handleMetricNotification(n notification.Notification) {
	metricNotification := n.(MetricNotification)
	metricMessage := NewMetricMessage(metricNotification.Points)
	babel.SendMessage(metricMessage, e.confs.ImporterAddr, importerProtoID, []protocol.ID{exporterProtoID})
}

func (e *ExporterProto) ID() protocol.ID {
	return exporterProtoID
}

func (e *ExporterProto) Name() string {
	return name
}

func (e *ExporterProto) Logger() *logrus.Logger {
	return e.logger
}

func (e *ExporterProto) Init() {
	babel.RegisterNotificationHandler(e.ID(), MetricNotification{}, e.handleMetricNotification)
	babel.RegisterTimerHandler(e.ID(), NewRedialTimer(0).ID(), e.handleRedialTimer)
}

func (e *ExporterProto) Start() {
	// for _, g := range e.gauges {
	// 	// go e.handleGauge(g)
	// }
	babel.Dial(e.confs.ImporterAddr, e.ID(), stream.NewUDPDialer())
}

func (e *ExporterProto) DialFailed(p peer.Peer) {
	e.failedDials++

	if e.failedDials > e.confs.MaxRedials {
		e.logger.Panicln("Could not dial importer")
	}

	babel.RegisterTimer(e.ID(), NewRedialTimer(e.confs.RedialTimeout))
}

func (e *ExporterProto) DialSuccess(sourceProto protocol.ID, peer peer.Peer) bool {
	return sourceProto == e.ID()
}

func (e *ExporterProto) InConnRequested(dialerProto protocol.ID, peer peer.Peer) bool {
	return false
}

func (e *ExporterProto) OutConnDown(peer peer.Peer) {
	e.logger.Panic("local conn down")
}
