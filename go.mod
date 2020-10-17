module github.com/nm-morais/deMMon-exporter

go 1.13

require (
	github.com/nm-morais/demmon-common v0.0.0-20201015082106-9efd45327054
	github.com/nm-morais/go-babel v1.0.0
	github.com/sirupsen/logrus v1.7.0
)

replace github.com/nm-morais/go-babel => ../go-babel

replace github.com/nm-morais/demmon-common => ../deMMon-common
