package exporter

import (
	"bytes"

	lineProto "github.com/influxdata/line-protocol"
	"github.com/nm-morais/go-babel/pkg/message"
)

// -------------- Metric --------------

const metricMessageID = 100

type metricMessage struct {
	Metrics []lineProto.Metric
}

func NewMetricMessage(metrics []lineProto.Metric) metricMessage {
	return metricMessage{
		Metrics: metrics,
	}
}

func (metricMessage) Type() message.ID {
	return metricMessageID
}

func (metricMessage) Serializer() message.Serializer {
	return metricMsgSerializer
}

func (metricMessage) Deserializer() message.Deserializer {
	return metricMsgSerializer
}

var metricMsgSerializer = MetricMsgSerializer{}

type MetricMsgSerializer struct {
}

func (MetricMsgSerializer) Serialize(m message.Message) []byte { // can be optimized to spend less memory
	buf := &bytes.Buffer{}
	serializer := lineProto.NewEncoder(buf)
	serializer.SetFieldTypeSupport(lineProto.UintSupport)
	metricMsg := m.(metricMessage)
	for _, m := range metricMsg.Metrics {
		_, err := serializer.Encode(m)
		if err != nil {
			panic(err)
		}
	}
	return buf.Bytes()
}

func (MetricMsgSerializer) Deserialize(msgBytes []byte) message.Message {
	deserializer := lineProto.NewParser(lineProto.NewMetricHandler())
	metrics, err := deserializer.Parse(msgBytes)
	if err != nil {
		panic(err)
	}
	return metricMessage{
		Metrics: metrics,
	}
}
