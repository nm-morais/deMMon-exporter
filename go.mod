module github.com/nm-morais/demmon-exporter

go 1.13

require (
	github.com/nm-morais/demmon-common v0.0.0-20201017151530-55e09669041d
	github.com/nm-morais/go-babel v1.0.0
	github.com/sirupsen/logrus v1.7.0
)

replace github.com/nm-morais/go-babel => ../go-babel

replace github.com/nm-morais/demmon-common => ../demmon-common
