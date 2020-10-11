package exporter

import (
	lineProto "github.com/influxdata/line-protocol"
	"github.com/nm-morais/go-babel/pkg/notification"
)

const metricNotificationID = 100

type metricNotification struct {
	Metric lineProto.Metric
	Batch  bool
}

func NewMetricNotification(m lineProto.Metric, batch bool) metricNotification {
	return metricNotification{
		Metric: m,
		Batch:  batch,
	}
}

func (metricNotification) ID() notification.ID {
	return metricNotificationID
}
