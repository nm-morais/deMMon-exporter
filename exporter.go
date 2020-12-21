package exporter

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	client "github.com/nm-morais/demmon-client/pkg"
	"github.com/nm-morais/demmon-common/body_types"
	"github.com/nm-morais/demmon-common/exporters"
	"github.com/nm-morais/demmon-exporter/internal/generic"
	"github.com/nm-morais/demmon-exporter/internal/lv"
	"github.com/nm-morais/demmon-exporter/internal/metrics"
	"github.com/sirupsen/logrus"
)

const (
	ContentTypeJSON = "application/json"
	ContentTypeText = "text/plain; charset=utf-8"
)

type Conf struct {
	Silent       bool
	LogFolder    string
	LogFile      string
	ImporterPort int
	ImporterHost string

	DialAttempts    int
	DialBackoffTime time.Duration

	DialTimeout    time.Duration
	RequestTimeout time.Duration
}

type Exporter struct {
	counters   *lv.Space
	gauges     *lv.Space
	histBounds map[string][]float64
	histograms *lv.Space
	tags       map[string]string

	bucketGranularities map[string]int

	client *client.DemmonClient
	logger *logrus.Logger
	conf   *Conf
}

func New(confs *Conf, host, service string, tags map[string]string) (*Exporter, error) {
	clientConf := client.DemmonClientConf{
		DemmonPort:     confs.ImporterPort,
		DemmonHostAddr: confs.ImporterHost,
		RequestTimeout: confs.RequestTimeout,
	}

	if tags == nil {
		tags = map[string]string{}
	}

	tags["service"] = service
	tags["host"] = host

	e := &Exporter{
		counters:            lv.NewSpace(),
		gauges:              lv.NewSpace(),
		histograms:          lv.NewSpace(),
		tags:                tags,
		logger:              logrus.New(),
		conf:                confs,
		bucketGranularities: make(map[string]int),
	}

	c := client.New(clientConf)
	e.client = c

	var connectErr error

	for i := 0; i < confs.DialAttempts; i++ {
		connectErr = c.ConnectTimeout(confs.DialTimeout)
		if connectErr != nil {
			time.Sleep(confs.DialBackoffTime) // sleep and retry
			continue
		}

		break
	}

	if connectErr != nil {
		return nil, connectErr
	}

	setupLogger(e.logger, e.conf.LogFolder, e.conf.LogFile, e.conf.Silent)

	return e, nil
}

// NewCounter returns an Influx counter.
func (e *Exporter) NewCounter(name string, nrSamplesToStore int) *Counter {
	e.bucketGranularities[name] = nrSamplesToStore

	return &Counter{
		name: name,
		obs:  e.counters.Observe,
	}
}

// NewGauge returns an Influx gauge.
func (e *Exporter) NewGauge(name string, nrSamplesToStore int) *Gauge {
	e.bucketGranularities[name] = nrSamplesToStore

	return &Gauge{
		name: name,
		obs:  e.gauges.Observe,
		add:  e.gauges.Add,
	}
}

func (e *Exporter) NewHistogram(name string, nrSamplesToStore int, upperBucketBounds []float64) *Histogram {
	e.bucketGranularities[name] = nrSamplesToStore

	return &Histogram{
		name: name,
		obs:  e.histograms.Observe,
	}
}

func (e *Exporter) ExportLoop(ctx context.Context, interval time.Duration) {
	e.logger.Info("Starting export loop")

	t := time.NewTicker(interval)

	for bName, bSampleCount := range e.bucketGranularities {
		e.logger.Info("installing buckets...")
		err := e.client.InstallBucket(bName, interval, bSampleCount)

		if err != nil {
			e.logger.Panic(err)
		}
	}

	for {
		select {
		case <-t.C:
			if err := e.Export(); err != nil {
				e.logger.Errorf("Error exporting: %s", err)
				continue
			}

			e.logger.Info("Exported metrics successfully")
		case <-ctx.Done():
			e.logger.Error("Context is done")
			return
		}
	}
}

func (e *Exporter) Export() (err error) {
	now := time.Now()
	bp := []*body_types.TimeseriesDTO{}

	e.logger.Infof("exporting metrics...")

	e.counters.Reset().Walk(
		func(name string, lvs lv.LabelValues, values []float64) bool {
			tags := mergeTags(e.tags, lvs)
			v := sum(values)
			fields := map[string]interface{}{"count": v}
			bp = append(bp, body_types.NewTimeseriesDTO(name, tags, body_types.NewObservable(fields, now)))
			return true
		},
	)

	e.gauges.Reset().Walk(
		func(name string, lvs lv.LabelValues, values []float64) bool {
			tags := mergeTags(e.tags, lvs)
			fields := map[string]interface{}{"value": last(values)}
			bp = append(bp, body_types.NewTimeseriesDTO(name, tags, body_types.NewObservable(fields, now)))
			return true
		},
	)

	e.histograms.Reset().Walk(
		func(name string, lvs lv.LabelValues, values []float64) bool {
			histBounds, ok := e.histBounds[name]
			if !ok {
				e.logger.Panicf("No bounds fot histogram %s", name)
			}
			histogram := generic.NewHistogram(name, histBounds)
			tags := mergeTags(e.tags, lvs)
			for _, v := range values {
				histogram.Observe(v)
			}
			fields := histogram.Value()
			bp = append(bp, body_types.NewTimeseriesDTO(name, tags, body_types.NewObservable(fields, now)))
			return true
		},
	)

	for _, ts := range bp {
		e.logger.Infof("%s:%s:%+v", ts.MeasurementName, ts.TSTags, ts.Values)
	}

	return e.client.PushMetricBlob(bp)
}

type formatter struct {
	owner string
	lf    logrus.Formatter
}

func (f *formatter) Format(e *logrus.Entry) ([]byte, error) {
	e.Message = fmt.Sprintf("[%s] %s", f.owner, e.Message)
	return f.lf.Format(e)
}

func setupLogger(logger *logrus.Logger, logFolder, logFile string, silent bool) {
	logger.SetFormatter(
		&formatter{
			owner: "demmon_exporter",
			lf: &logrus.TextFormatter{
				DisableColors:   true,
				ForceColors:     false,
				FullTimestamp:   true,
				TimestampFormat: time.StampMilli,
			},
		},
	)

	if logFolder == "" {
		logger.Panicf("Invalid logFolder '%s'", logFolder)
	}

	if logFile == "" {
		logger.Panicf("Invalid logFile '%s'", logFile)
	}

	filePath := fmt.Sprintf("%s/%s", logFolder, logFile)
	err := os.MkdirAll(logFolder, 0777)

	if err != nil {
		logger.Panic(err)
	}

	file, err := os.Create(filePath)
	if os.IsExist(err) {
		var err = os.Remove(filePath)
		if err != nil {
			logger.Panic(err)
		}

		file, err = os.Create(filePath)
		if err != nil {
			logger.Panic(err)
		}
	}

	var out io.Writer = file

	if silent {
		logger.SetOutput(out)
		fmt.Println("Setting exporter silently")

		return
	}

	out = io.MultiWriter(os.Stdout, file)
	logger.SetOutput(out)
}

type observeFunc func(name string, lvs lv.LabelValues, value float64)

type Counter struct {
	name string
	lvs  lv.LabelValues
	obs  observeFunc
}

func (c *Counter) With(labelValues ...string) exporters.Counter {
	return &Counter{
		name: c.name,
		lvs:  c.lvs.With(labelValues...),
		obs:  c.obs,
	}
}

func (c *Counter) Add(delta float64) {
	c.obs(c.name, c.lvs, delta)
}

type Gauge struct {
	name string
	lvs  lv.LabelValues
	obs  observeFunc
	add  observeFunc
}

// With implements exporters.Gauge.
func (g *Gauge) With(labelValues ...string) exporters.Gauge {
	return &Gauge{
		name: g.name,
		lvs:  g.lvs.With(labelValues...),
		obs:  g.obs,
		add:  g.add,
	}
}

// Set implements exporters.Gauge.
func (g *Gauge) Set(value float64) {
	g.obs(g.name, g.lvs, value)
}

// Add implements exporters.Gauge.
func (g *Gauge) Add(delta float64) {
	g.add(g.name, g.lvs, delta)
}

// Histogram is an Influx histrogram. Observations are aggregated into a
// generic.Histogram and emitted as per-quantile gauges to the Influx server.
type Histogram struct {
	name string
	lvs  lv.LabelValues
	obs  observeFunc
}

// With implements metrics.Histogram.
func (h *Histogram) With(labelValues ...string) metrics.Histogram {
	return &Histogram{
		name: h.name,
		lvs:  h.lvs.With(labelValues...),
		obs:  h.obs,
	}
}

func (h *Histogram) Observe(value float64) {
	h.obs(h.name, h.lvs, value)
}

func mergeTags(tags map[string]string, labelValues []string) map[string]string {
	if len(labelValues)%2 != 0 {
		panic("mergeTags received a labelValues with an odd number of strings")
	}

	ret := make(map[string]string, len(tags)+len(labelValues)/2)

	for k, v := range tags {
		ret[k] = v
	}

	for i := 0; i < len(labelValues); i += 2 {
		ret[labelValues[i]] = labelValues[i+1]
	}

	return ret
}

func sum(a []float64) float64 {
	var v float64
	for _, f := range a {
		v += f
	}

	return v
}

func last(a []float64) float64 {
	return a[len(a)-1]
}

// type exporterState struct {
// 	finishOnce *sync.Once
// 	finishChan chan interface{}

// 	emitFrequency time.Duration
// 	granularity   []timeseries.Granularity
// 	batchMetric   bool
// 	metricPlugin  string
// }

// func (e *Exporter) Start() error {
// 	connected := false
// 	for i := 0; i < e.conf.DialAttempts; i++ {
// 		err := e.client.ConnectTimeout(e.conf.DialTimeout)
// 		if err != nil {
// 			e.logger.Warn(err)
// 			time.Sleep(e.conf.DialBackoffTime)
// 			continue
// 		}
// 		connected = true
// 	}
// 	if !connected {
// 		return errors.New("could not connect to demmon")
// 	}
// 	e.logger.Info("Dialed demmon successfully")

// 	for pluginName, pluginPath := range e.plugins {
// 		err := e.client.AddPlugin(pluginPath, pluginName)
// 		if err != nil {
// 			e.logger.Error(err)
// 			e.logger.Panic(err)
// 		}
// 	}

// 	var i int
// 	for i = 0; i < e.conf.MaxRegisterMetricsRetries; i++ {
// 		if i != 0 {
// 			time.Sleep(e.conf.RegisterMetricsBackoffTime)
// 		}
// 		toRegister := make([]body_types.MetricMetadata, 0, len(e.m))
// 		for _, m := range e.m {
// 			toRegister = append(toRegister, body_types.MetricMetadata{
// 				Name:          m.exporter.Name(),
// 				Granularities: m.granularity,
// 				Plugin:        m.metricPlugin,
// 				Service:       e.conf.ServiceName,
// 				Origin:        e.conf.SenderId,
// 			})
// 		}
// 		err := e.client.RegisterMetrics(toRegister)
// 		if err != nil {
// 			e.logger.Error(err)
// 			continue
// 		}
// 		break
// 	}

// 	if i == e.conf.MaxRegisterMetricsRetries {
// 		return errors.New("could not register metrics with importer")
// 	}

// 	for _, m := range e.m {
// 		if !m.batchMetric {
// 			go e.handleNonBatchMetricExport(m)
// 		}
// 	}

// 	t := time.NewTicker(e.conf.ExportFrequency)
// 	for {
// 		<-t.C
// 		e.batchMu.Lock()
// 		aux := make([]string, 0, len(e.currBatch))
// 		aux = append(aux, e.currBatch...)
// 		e.batchMu.Unlock()
// 		// for _, p := range aux {
// 		// 	e.logger.Infof("Exporting entry: %s", p)
// 		// }
// 		err := e.client.PushMetricBlob(e.conf.ServiceName, e.conf.SenderId, aux)
// 		e.currBatch = []string{}
// 		if err != nil {
// 			e.logger.Error(err)
// 		}
// 	}
// }

// func (e *Exporter) RegisterMetric(
// 	m exporters.Exporter,
// 	plugin string,
// 	storageGranularity []timeseries.Granularity,
// 	batchMetric bool,
// 	exportFrequency time.Duration) {

// 	e.mu.Lock()
// 	// defer will unlock in case of panic
// 	// checks in test
// 	defer e.mu.Unlock()
// 	e.mustRegisterLocked(m, plugin, storageGranularity, batchMetric, exportFrequency)
// }

// // UnregisterMetric removes metric with the given name from the exporter.
// // True is returned if the metric has been removed.
// // False is returned if the given metric is missing in s.
// func (e *Exporter) UnregisterMetric(name string) bool {
// 	e.mu.Lock()
// 	defer e.mu.Unlock()
// 	m, ok := e.m[name]
// 	if !ok {
// 		return false
// 	}
// 	m.finishOnce.Do(func() {
// 		close(m.finishChan)
// 	})
// 	delete(e.m, name)
// 	return true
// }

// // ListMetricNames returns a list of all the metrics in s.
// func (e *Exporter) ListMetricNames() []string {
// 	var list []string
// 	for name := range e.m {
// 		list = append(list, name)
// 	}
// 	return list
// }

// // mustRegisterLocked registers given metric with
// // the given name. Panics if the given name was
// // already registered before.
// func (e *Exporter) mustRegisterLocked(m exporters.Exporter,
// 	plugin string,
// 	storageGranularity []timeseries.Granularity,
// 	batchMetric bool,
// 	exportFrequency time.Duration) {

// 	_, ok := e.m[m.Name()]
// 	if !ok {
// 		nm := &exporterState{
// 			finishChan:    make(chan interface{}),
// 			finishOnce:    &sync.Once{},
// 			exporter:      m,
// 			batchMetric:   batchMetric,
// 			emitFrequency: exportFrequency,
// 			metricPlugin:  plugin,
// 			granularity:   storageGranularity,
// 		}
// 		e.m[m.Name()] = nm
// 	}
// 	if ok {
// 		panic(fmt.Errorf("BUG: metric %q is already registered", m.Name()))
// 	}
// }

// func (e *Exporter) handleNonBatchMetricExport(metricState *exporterState) {
// 	e.logger.Infof("Starting metric export for metric %s", metricState.exporter.Name())
// 	t := time.NewTicker(metricState.emitFrequency)
// 	for {
// 		select {
// 		case <-t.C:
// 			// TODO
// 			// toSend := &bytes.Buffer{}
// 			// toSend.WriteString(marshalMetricAsTextLine(metricState.metric))
// 			// _, err := e.makeRequestToImporter(routes.AddMetricsMethod, routes.AddMetricsPath, ContentTypeText, toSend)
// 			v := metricState.exporter.Get()
// 			ts := time.Now()
// 			pv := &body_types.Metric{TS: ts, Value: v}
// 			marshaledVal, err := metrics.MarshalPVAsTextLine(e.conf.ServiceName, metricState.exporter.Name(), e.conf.SenderId, pv)
// 			if err != nil {
// 				e.logger.Panicf("Could not marshal metric %s", metricState.exporter.Name())
// 			}
// 			// e.logger.Infof("Exporting: %s", marshaledVal)
// 			err = e.client.PushMetricBlob(e.conf.ServiceName, e.conf.SenderId, []string{marshaledVal})
// 			if err != nil {
// 				e.logger.Errorf("Error flushing metric %s : %s", metricState.exporter.Name(), err)
// 			}

// 		case <-metricState.finishChan:
// 			return
// 		}
// 	}
// }
