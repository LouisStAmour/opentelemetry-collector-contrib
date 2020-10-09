// Copyright OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package azuremonitorexporter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.uber.org/zap"
)

type metricExporter struct {
	config           *Config
	transportChannel transportChannel
	logger           *zap.Logger
}

type metricVisitor struct {
	processed int
	err       error
	exporter  *metricExporter
}

// Called for each tuple of Resource, InstrumentationLibrary, and Metric
func (v *metricVisitor) visit(
	resource pdata.Resource,
	instrumentationLibrary pdata.InstrumentationLibrary, metric pdata.Metric) (ok bool) {

	envelope, err := metricToEnvelope(resource, instrumentationLibrary, metric, v.exporter.logger)
	if err != nil {
		// record the error and short-circuit
		v.err = consumererror.Permanent(err)
		return false
	}

	// apply the instrumentation key to the envelope
	envelope.IKey = v.exporter.config.InstrumentationKey

	// This is a fire and forget operation
	v.exporter.transportChannel.Send(envelope)
	v.processed++

	return true
}

func (exporter *metricExporter) onMetricData(context context.Context, metricData pdata.Metrics) (droppedSpans int, err error) {
	metricCount := metricData.MetricCount()
	if metricCount == 0 {
		return 0, nil
	}

	visitor := &metricVisitor{exporter: exporter}
	AcceptMetrics(metricData, visitor)
	return metricCount - visitor.processed, visitor.err
}

// Returns a new instance of the metric exporter
func newMetricsExporter(config *Config, transportChannel transportChannel, logger *zap.Logger) (component.MetricsExporter, error) {

	exporter := &metricExporter{
		config:           config,
		transportChannel: transportChannel,
		logger:           logger,
	}

	return exporterhelper.NewMetricsExporter(config, exporter.onMetricData)
}
