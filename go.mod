module github.com/nm-morais/demmon-exporter

go 1.13

require (
	github.com/VividCortex/gohistogram v1.0.0
	github.com/go-kit/kit v0.10.0
	github.com/influxdata/influxdb v1.8.3
	github.com/influxdata/influxdb1-client v0.0.0-20200827194710-b269163b24ab
	github.com/influxdata/line-protocol v0.0.0-20200327222509-2487e7298839
	github.com/nm-morais/demmon-metrics-client v0.0.0-20201011123844-55ca4bcfbea9 // indirect
	github.com/nm-morais/go-babel v1.0.0
	github.com/sirupsen/logrus v1.7.0
)

replace github.com/nm-morais/go-babel => ../go-babel
