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
	"testing"

	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

// Tests the export onMetricData callback with no Spans
func TestExporterMetricsDataCallbackNoSpans(t *testing.T) {
	mockTransportChannel := getMockTransportChannel()
	exporter := getMetricsExporter(defaultConfig, mockTransportChannel)

	metrics := pdata.NewMetrics()

	droppedMetrics, err := exporter.onMetricData(context.Background(), metrics)
	assert.Nil(t, err)
	assert.Equal(t, 0, droppedMetrics)

	mockTransportChannel.AssertNumberOfCalls(t, "Send", 0)
}

// Tests the export onMetricData callback with a single Span
func TestExporterMetricsDataCallbackSingleSpan(t *testing.T) {
	mockTransportChannel := getMockTransportChannel()
	exporter := getMetricsExporter(defaultConfig, mockTransportChannel)

	// re-use some test generation method(s) from trace_to_envelope_test
	resource := getResource()
	instrumentationLibrary := getInstrumentationLibrary()
	metric := getDefaultHTTPServerMetric()

	metrics := pdata.NewMetrics()
	metrics.ResourceMetrics().Resize(1)
	rm := metrics.ResourceMetrics().At(0)
	r := rm.Resource()
	r.InitEmpty()
	resource.CopyTo(r)
	rm.InstrumentationLibraryMetrics().Resize(1)
	ilss := rm.InstrumentationLibraryMetrics().At(0)
	instrumentationLibrary.CopyTo(ilss.InstrumentationLibrary())
	ilss.Metrics().Resize(1)
	metric.CopyTo(ilss.Metrics().At(0))

	droppedMetrics, err := exporter.onMetricData(context.Background(), metrics)
	assert.Nil(t, err)
	assert.Equal(t, 0, droppedMetrics)

	mockTransportChannel.AssertNumberOfCalls(t, "Send", 1)
}

// Tests the export onMetricData callback with a single Span that fails to produce an envelope
func TestExporterMetricsDataCallbackSingleSpanNoEnvelope(t *testing.T) {
	mockTransportChannel := getMockTransportChannel()
	exporter := getMetricsExporter(defaultConfig, mockTransportChannel)

	// re-use some test generation method(s) from trace_to_envelope_test
	resource := getResource()
	instrumentationLibrary := getInstrumentationLibrary()
	metric := getDefaultInternalMetric()

	// Make this a FaaS span, which will trigger an error, because conversion
	// of them is currently not supported.
	metric.Attributes().InsertString(conventions.AttributeFaaSTrigger, "http")

	metrics := pdata.NewMetrics()
	metrics.ResourceMetrics().Resize(1)
	rm := metrics.ResourceMetrics().At(0)
	r := rm.Resource()
	r.InitEmpty()
	resource.CopyTo(r)
	rm.InstrumentationLibraryMetrics().Resize(1)
	ilss := rm.InstrumentationLibraryMetrics().At(0)
	instrumentationLibrary.CopyTo(ilss.InstrumentationLibrary())
	ilss.Metrics().Resize(1)
	metric.CopyTo(ilss.Metrics().At(0))

	droppedMetrics, err := exporter.onMetricData(context.Background(), metrics)
	assert.NotNil(t, err)
	assert.True(t, consumererror.IsPermanent(err), "error should be permanent")
	assert.Equal(t, 1, droppedMetrics)

	mockTransportChannel.AssertNumberOfCalls(t, "Send", 0)
}

func getMetricsExporter(config *Config, transportChannel transportChannel) *metricExporter {
	return &metricExporter{
		config,
		transportChannel,
		zap.NewNop(),
	}
}
