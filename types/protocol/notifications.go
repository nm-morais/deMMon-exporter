package prototypes

import (
	influxdb "github.com/influxdata/influxdb/client/v2"
	"github.com/nm-morais/go-babel/pkg/notification"
)

const metricNotificationID = 100

type MetricNotification struct {
	Points influxdb.BatchPoints
}

func NewMetricNotification(points influxdb.BatchPoints) MetricNotification {
	return MetricNotification{
		Points: points,
	}
}

func (m MetricNotification) ID() notification.ID {
	return metricNotificationID
}
