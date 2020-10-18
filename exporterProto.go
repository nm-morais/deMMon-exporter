package exporter

import (
	"github.com/nm-morais/demmon-exporter/types/protocolTypes"
	"github.com/nm-morais/go-babel/pkg/errors"
	"github.com/nm-morais/go-babel/pkg/logs"
	"github.com/nm-morais/go-babel/pkg/message"
	"github.com/nm-morais/go-babel/pkg/peer"
	"github.com/nm-morais/go-babel/pkg/protocol"
	"github.com/nm-morais/go-babel/pkg/protocolManager"
	"github.com/nm-morais/go-babel/pkg/timer"
	"github.com/sirupsen/logrus"
)

const (
	name            = "Exporter"
	exporterProtoID = 100
	importerProtoID = 101
)

type ExporterProto struct {
	babel       protocolManager.ProtocolManager
	exporter    *Exporter
	confs       ExporterConf
	logger      *logrus.Logger
	failedDials int
}

func NewExporterProto(confs ExporterConf, exporter *Exporter, protocolManager protocolManager.ProtocolManager) *ExporterProto {
	return &ExporterProto{
		babel:       protocolManager,
		exporter:    exporter,
		confs:       confs,
		failedDials: 0,
		logger:      logs.NewLogger(name),
	}
}

func (e *ExporterProto) MessageDelivered(message message.Message, peer peer.Peer) {}

func (e *ExporterProto) MessageDeliveryErr(message message.Message, peer peer.Peer, error errors.Error) {
}

func (e *ExporterProto) handleRedialTimer(timer timer.Timer) {
	e.babel.Dial(e.ID(), e.confs.ImporterAddr, e.confs.ImporterAddr.ToUDPAddr())
}

func (e *ExporterProto) handleFlushTimer(timer timer.Timer) {
	e.babel.RegisterTimer(e.ID(), protocolTypes.NewFlushTimer(e.confs.ExportFrequency))
	e.logger.Info("Exporting metrics")
	if err != nil {
		e.logger.Error(err)
		return
	}
	e.logger.Info("Exported metrics successfully")
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
	e.babel.RegisterNotificationHandler(e.ID(), protocolTypes.MetricNotification{}, e.handleMetricNotification)
	e.babel.RegisterTimerHandler(e.ID(), protocolTypes.NewFlushTimer(0).ID(), e.handleFlushTimer)

	// babel.RegisterTimerHandler(e.ID(), NewRedialTimer(0).ID(), e.handleRedialTimer)
}

func (e *ExporterProto) Start() {
	// for _, g := range e.gauges {
	// 	// go e.handleGauge(g)
	// }
	// babel.Dial(e.confs.ImporterAddr, e.ID(), stream.NewUDPDialer())
	e.babel.RegisterTimer(e.ID(), protocolTypes.NewFlushTimer(0))
}

func (e *ExporterProto) DialFailed(p peer.Peer) {
	e.failedDials++

	if e.failedDials > e.confs.MaxRedials {
		e.logger.Panicln("Could not dial importer")
	}

	e.babel.RegisterTimer(e.ID(), protocolTypes.NewRedialTimer(e.confs.RedialTimeout))
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
