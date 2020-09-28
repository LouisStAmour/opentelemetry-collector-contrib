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

// Tests the export onLogData callback with no Logs
func TestExporterLogsDataCallbackNoLogs(t *testing.T) {
	mockTransportChannel := getMockTransportChannel()
	exporter := getLogsExporter(defaultConfig, mockTransportChannel)

	logs := pdata.NewLogs()

	droppedLogs, err := exporter.onLogData(context.Background(), logs)
	assert.Nil(t, err)
	assert.Equal(t, 0, droppedLogs)

	mockTransportChannel.AssertNumberOfCalls(t, "Send", 0)
}

// Tests the export onLogData callback with a single Log
func TestExporterLogsDataCallbackSingleLog(t *testing.T) {
	mockTransportChannel := getMockTransportChannel()
	exporter := getLogsExporter(defaultConfig, mockTransportChannel)

	// re-use some test generation method(s) from trace_to_envelope_test
	resource := getResource()
	instrumentationLibrary := getInstrumentationLibrary()
	log := getDefaultHTTPServerLog()

	logs := pdata.NewLogs()
	logs.ResourceLogs().Resize(1)
	rl := logs.ResourceLogs().At(0)
	r := rl.Resource()
	r.InitEmpty()
	resource.CopyTo(r)
	rl.InstrumentationLibraryLogs().Resize(1)
	ilss := rl.InstrumentationLibraryLogs().At(0)
	instrumentationLibrary.CopyTo(ilss.InstrumentationLibrary())
	ilss.Logs().Resize(1)
	log.CopyTo(ilss.Logs().At(0))

	droppedLogs, err := exporter.onLogData(context.Background(), logs)
	assert.Nil(t, err)
	assert.Equal(t, 0, droppedLogs)

	mockTransportChannel.AssertNumberOfCalls(t, "Send", 1)
}

// Tests the export onLogData callback with a single Log that fails to produce an envelope
func TestExporterLogsDataCallbackSingleLogNoEnvelope(t *testing.T) {
	mockTransportChannel := getMockTransportChannel()
	exporter := getLogsExporter(defaultConfig, mockTransportChannel)

	// re-use some test generation method(s) from trace_to_envelope_test
	resource := getResource()
	instrumentationLibrary := getInstrumentationLibrary()
	log := getDefaultInternalLog()

	// Make this a FaaS span, which will trigger an error, because conversion
	// of them is currently not supported.
	log.Attributes().InsertString(conventions.AttributeFaaSTrigger, "http")

	logs := pdata.NewLogs()
	logs.ResourceLogs().Resize(1)
	rl := logs.ResourceLogs().At(0)
	r := rl.Resource()
	r.InitEmpty()
	resource.CopyTo(r)
	rl.InstrumentationLibraryLogs().Resize(1)
	ilss := rl.InstrumentationLibraryLogs().At(0)
	instrumentationLibrary.CopyTo(ilss.InstrumentationLibrary())
	ilss.Logs().Resize(1)
	log.CopyTo(ilss.Logs().At(0))

	droppedLogs, err := exporter.onLogData(context.Background(), logs)
	assert.NotNil(t, err)
	assert.True(t, consumererror.IsPermanent(err), "error should be permanent")
	assert.Equal(t, 1, droppedLogs)

	mockTransportChannel.AssertNumberOfCalls(t, "Send", 0)
}

func getLogsExporter(config *Config, transportChannel transportChannel) *logExporter {
	return &logExporter{
		config,
		transportChannel,
		zap.NewNop(),
	}
}
