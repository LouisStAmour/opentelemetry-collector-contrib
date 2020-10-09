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

// Contains code common to both trace and metrics exporters
import (
	"time"

	"github.com/microsoft/ApplicationInsights-Go/appinsights/contracts"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
	"go.uber.org/zap"
)

// Transforms a tuple of pdata.Resource, pdata.InstrumentationLibrary, pdata.LogRecord into an AppInsights contracts.Envelope
// This is the only method that should be targeted in the unit tests
func logRecordToEnvelope(
	resource pdata.Resource,
	instrumentationLibrary pdata.InstrumentationLibrary,
	logRecord pdata.LogRecord,
	logger *zap.Logger) (*contracts.Envelope, error) {

	envelope := contracts.NewEnvelope()
	envelope.Tags = make(map[string]string)
	envelope.Time = toTime(logRecord.Timestamp()).Format(time.RFC3339Nano)
	traceIDHexString := idToHex(logRecord.TraceID().Bytes())
	if len(traceIDHexString) == 0 {
		traceIDHexString = "00000000000000000000000000000000"
	}
	envelope.Tags[contracts.OperationId] = traceIDHexString
	spanIDHexString := idToHex(logRecord.SpanID().Bytes())
	if len(spanIDHexString) == 0 {
		spanIDHexString = "0000000000000000"
	}
	envelope.Tags[contracts.OperationParentId] = "|" + traceIDHexString + "." + spanIDHexString

	data := contracts.NewData()
	var dataSanitizeFunc func() []string
	var dataProperties map[string]string

	// TODO: fill in data

	envelope.Data = data
	resourceAttributes := resource.Attributes()

	// Copy all the resource labels into the base data properties. Resource values are always strings
	resourceAttributes.ForEach(func(k string, v pdata.AttributeValue) { dataProperties[k] = v.StringVal() })

	// Copy the instrumentation properties
	if !instrumentationLibrary.IsNil() {
		if instrumentationLibrary.Name() != "" {
			dataProperties[instrumentationLibraryName] = instrumentationLibrary.Name()
		}

		if instrumentationLibrary.Version() != "" {
			dataProperties[instrumentationLibraryVersion] = instrumentationLibrary.Version()
		}
	}

	// Extract key service.* labels from the Resource labels and construct CloudRole and CloudRoleInstance envelope tags
	// https://github.com/open-telemetry/opentelemetry-specification/tree/master/specification/resource/semantic_conventions
	if serviceName, serviceNameExists := resourceAttributes.Get(conventions.AttributeServiceName); serviceNameExists {
		cloudRole := serviceName.StringVal()

		if serviceNamespace, serviceNamespaceExists := resourceAttributes.Get(conventions.AttributeServiceNamespace); serviceNamespaceExists {
			cloudRole = serviceNamespace.StringVal() + "." + cloudRole
		}

		envelope.Tags[contracts.CloudRole] = cloudRole
	}

	if serviceInstance, exists := resourceAttributes.Get(conventions.AttributeServiceInstance); exists {
		envelope.Tags[contracts.CloudRoleInstance] = serviceInstance.StringVal()
	}

	// Sanitize the base data, the envelope and envelope tags
	sanitize(dataSanitizeFunc, logger)
	sanitize(func() []string { return envelope.Sanitize() }, logger)
	sanitize(func() []string { return contracts.SanitizeTags(envelope.Tags) }, logger)

	return envelope, nil
}