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
	"go.opentelemetry.io/collector/consumer/pdata"
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

func getMetricsExporter(config *Config, transportChannel transportChannel) *metricExporter {
	return &metricExporter{
		config,
		transportChannel,
		zap.NewNop(),
	}
}
