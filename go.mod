module github.com/nm-morais/deMMon-exporter

go 1.13

require (
	github.com/VividCortex/gohistogram v1.0.0
	github.com/influxdata/influxdb v1.8.3
	github.com/influxdata/line-protocol v0.0.0-20180522152040-32c6aa80de5e
	github.com/nm-morais/go-babel v1.0.0
	github.com/sirupsen/logrus v1.7.0
)

replace github.com/nm-morais/go-babel => ../go-babel
