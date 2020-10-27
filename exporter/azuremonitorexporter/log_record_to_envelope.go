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
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
	"regexp"
	"strconv"
	"time"

	"github.com/microsoft/ApplicationInsights-Go/appinsights/contracts"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
	"go.uber.org/zap"
)

var mapStringToAppInsightsSeverity = map[contracts.SeverityLevel]*regexp.Regexp{
	contracts.Verbose: regexp.MustCompile("(?i)^(?:TRACE|FINEST|DEBUG|Verbose|FINER|FINE|CONFIG)[2-4]*$"),
	contracts.Information: regexp.MustCompile("(?i)^(?:INFO|Informational|Information|Notice)[2-4]*$"),
	contracts.Warning: regexp.MustCompile("(?i)^(?:WARN|Warning)[2-4]*$"),
	contracts.Error: regexp.MustCompile("(?i)^(?:ERROR|SEVERE)[2-4]*$"),
	contracts.Critical: regexp.MustCompile("(?i)^(?:Critical|Dpanic|Emergency|Panic|FATAL|Alert)[2-4]*$"),
}

func getSeverityLevel(logRecord pdata.LogRecord) (bool, contracts.SeverityLevel) {
	if logRecord.SeverityNumber() != pdata.SeverityNumberUNDEFINED {
		switch sev := logRecord.SeverityNumber(); {
		case sev <= pdata.SeverityNumberDEBUG4:
			return true, contracts.Verbose
		case sev <= pdata.SeverityNumberINFO4:
			return true, contracts.Information
		case sev <= pdata.SeverityNumberWARN4:
			return true, contracts.Warning
		case sev <= pdata.SeverityNumberERROR4:
			return true, contracts.Error
		default:
			return true, contracts.Critical
		}
	} else {
		sev := logRecord.SeverityText()
		for level, r := range mapStringToAppInsightsSeverity {
			if r.MatchString(sev) {
				return true, level
			}
		}
	}
	return false, contracts.Information
}

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

	// Application Insights Messages can have severity but not metrics,
	// Application Insights Events can have metrics but not severity...
	// Since Application Insights messages are more limited than events in terms of structured data,
	// we only use them in certain scenarios...
	sevFound, sevLevel := getSeverityLevel(logRecord)
	if logRecord.Body().Type() == pdata.AttributeValueSTRING && sevFound {
		data := contracts.NewMessageData()
		data.Message = logRecord.Body().StringVal()
		data.SeverityLevel = sevLevel
		logRecord.Attributes().ForEach(func(k string, v pdata.AttributeValue) {
			data.Properties[k] = tracetranslator.AttributeValueToString(v, false)
		})
		envelope.Data = data
	} else {
		data := contracts.NewEventData()
		copyAttributesWithoutMapping(logRecord.Attributes(), data.Properties, data.Measurements)
		data.Name = logRecord.Name()
		switch logRecord.Body().Type() {
		case pdata.AttributeValueMAP:
			copyAttributesWithoutMapping(logRecord.Body().MapVal(), data.Properties, data.Measurements)
		default:
			data.Properties["Message"] = tracetranslator.AttributeValueToString(logRecord.Body(), false)
		}
		data.Properties["SeverityText"] = logRecord.SeverityText()
		data.Properties["SeverityNumber"] = strconv.FormatInt(int64(logRecord.SeverityNumber()), 10)
		envelope.Data = data
	}

	var dataSanitizeFunc func() []string
	var dataProperties map[string]string

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