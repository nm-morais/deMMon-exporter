package protocolTypes

import (
	"github.com/nm-morais/go-babel/pkg/notification"
)

const metricNotificationID = 3000

type MetricNotification struct {
	Metrics []byte
}

func NewMetricNotification(metrics []byte) MetricNotification {
	return MetricNotification{
		Metrics: metrics,
	}
}

func (m MetricNotification) ID() notification.ID {
	return metricNotificationID
}
