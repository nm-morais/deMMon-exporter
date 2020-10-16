package protocoltypes

import (
	"github.com/nm-morais/go-babel/pkg/notification"
)

const metricNotificationID = 100

type MetricNotification struct {
}

func NewMetricNotification() MetricNotification {
	return MetricNotification{}
}

func (m MetricNotification) ID() notification.ID {
	return metricNotificationID
}
