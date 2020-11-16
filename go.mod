module github.com/nm-morais/demmon-exporter

go 1.15

require (
	github.com/nm-morais/demmon-client v1.0.0
	github.com/nm-morais/demmon-common v1.0.0
	github.com/sirupsen/logrus v1.7.0
	go.uber.org/atomic v1.7.0
)

replace github.com/nm-morais/demmon-client => ../demmon-client

replace github.com/nm-morais/demmon-common => ../demmon-common

replace github.com/nm-morais/go-babel => ../go-babel
