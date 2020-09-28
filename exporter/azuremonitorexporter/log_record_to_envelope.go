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
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/microsoft/ApplicationInsights-Go/appinsights/contracts"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

var (
	errUnsupportedLogRecordType          = errors.New("unsupported LogRecord type")
)

// Used to identify the type of a received LogRecord
type logRecordType int8

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

	if logRecordKind == pdata.LogRecordKindSERVER || logRecordKind == pdata.LogRecordKindCONSUMER {
		requestData := logRecordToRequestData(logRecord, incomingLogRecordType)
		dataProperties = requestData.Properties
		dataSanitizeFunc = requestData.Sanitize
		envelope.Name = requestData.EnvelopeName("")
		envelope.Tags[contracts.OperationName] = requestData.Name
		data.BaseData = requestData
		data.BaseType = requestData.BaseType()
	} else if logRecordKind == pdata.LogRecordKindCLIENT || logRecordKind == pdata.LogRecordKindPRODUCER || logRecordKind == pdata.LogRecordKindINTERNAL {
		remoteDependencyData := logRecordToRemoteDependencyData(logRecord, incomingLogRecordType)

		// Regardless of the detected LogRecord type, if the LogRecordKind is Internal we need to set data.Type to InProc
		if logRecordKind == pdata.LogRecordKindINTERNAL {
			remoteDependencyData.Type = "InProc"
		}

		dataProperties = remoteDependencyData.Properties
		dataSanitizeFunc = remoteDependencyData.Sanitize
		envelope.Name = remoteDependencyData.EnvelopeName("")
		data.BaseData = remoteDependencyData
		data.BaseType = remoteDependencyData.BaseType()
	}

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

// Maps Server/Consumer LogRecord to AppInsights RequestData
func logRecordToRequestData(logRecord pdata.LogRecord, incomingLogRecordType logRecordType) *contracts.RequestData {
	// See https://github.com/microsoft/ApplicationInsights-Go/blob/master/appinsights/contracts/requestdata.go
	// Start with some reasonable default for server logRecords.
	data := contracts.NewRequestData()
	data.Id = idToHex(logRecord.SpanID())
	data.Name = logRecord.Name()
	data.Duration = formatLogRecordDuration(logRecord)
	data.Properties = make(map[string]string)
	data.Measurements = make(map[string]float64)
	data.ResponseCode, data.Success = getDefaultFormattedLogRecordStatus(logRecord.Status())

	switch incomingLogRecordType {
	case httpLogRecordType:
		fillRequestDataHTTP(logRecord, data)
	case rpcLogRecordType:
		fillRequestDataRPC(logRecord, data)
	case messagingLogRecordType:
		fillRequestDataMessaging(logRecord, data)
	case unknownLogRecordType:
		copyAttributesWithoutMapping(logRecord.Attributes(), data.Properties, data.Measurements)
	}

	return data
}

// Maps LogRecord to AppInsights RemoteDependencyData
func logRecordToRemoteDependencyData(logRecord pdata.LogRecord, incomingLogRecordType logRecordType) *contracts.RemoteDependencyData {
	// https://github.com/microsoft/ApplicationInsights-Go/blob/master/appinsights/contracts/remotedependencydata.go
	// Start with some reasonable default for dependent logRecords.
	data := contracts.NewRemoteDependencyData()
	data.Id = idToHex(logRecord.SpanID())
	data.Name = logRecord.Name()
	data.ResultCode, data.Success = getDefaultFormattedLogRecordStatus(logRecord.Status())
	data.Duration = formatLogRecordDuration(logRecord)
	data.Properties = make(map[string]string)
	data.Measurements = make(map[string]float64)

	switch incomingLogRecordType {
	case httpLogRecordType:
		fillRemoteDependencyDataHTTP(logRecord, data)
	case rpcLogRecordType:
		fillRemoteDependencyDataRPC(logRecord, data)
	case databaseLogRecordType:
		fillRemoteDependencyDataDatabase(logRecord, data)
	case messagingLogRecordType:
		fillRemoteDependencyDataMessaging(logRecord, data)
	case unknownLogRecordType:
		copyAttributesWithoutMapping(logRecord.Attributes(), data.Properties, data.Measurements)
	}

	return data
}

func getFormattedHTTPStatusValues(statusCode int64) (statusAsString string, success bool) {
	// see https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/http.md#status
	return strconv.FormatInt(statusCode, 10), statusCode >= 100 && statusCode <= 399
}

// Maps HTTP Server LogRecord to AppInsights RequestData
// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/http.md#semantic-conventions-for-http-logRecords
func fillRequestDataHTTP(logRecord pdata.LogRecord, data *contracts.RequestData) {
	attrs := copyAndExtractHTTPAttributes(logRecord.Attributes(), data.Properties, data.Measurements)

	if attrs.HTTPStatusCode != 0 {
		data.ResponseCode, data.Success = getFormattedHTTPStatusValues(attrs.HTTPStatusCode)
	}

	var sb strings.Builder

	// Construct data.Name
	// The data.Name should be {HTTP METHOD} {HTTP SERVER ROUTE TEMPLATE}
	// https://github.com/microsoft/ApplicationInsights-Home/blob/f1f9f619d74557c8db3dbde4b49c4193e10d8a81/EndpointSpecs/Schemas/Bond/RequestData.bond#L32
	sb.WriteString(attrs.HTTPMethod)
	sb.WriteString(" ")

	// Use httpRoute if available otherwise fallback to the logRecord name
	// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/http.md#name
	if attrs.HTTPRoute != "" {
		sb.WriteString(prefixIfNecessary(attrs.HTTPRoute, "/"))
	} else {
		sb.WriteString(logRecord.Name())
	}

	data.Name = sb.String()
	sb.Reset()

	/*
		To construct the value for data.Url we will use the following sets of attributes as defined by the otel spec
		Order of preference is:
		http.scheme, http.host, http.target
		http.scheme, http.server_name, net.host.port, http.target
		http.scheme, net.host.name, net.host.port, http.target
		http.url
	*/

	if attrs.HTTPTarget != "" {
		attrs.HTTPTarget = prefixIfNecessary(attrs.HTTPTarget, "/")
	}

	netHostPortAsString := ""
	if attrs.NetworkAttributes.NetHostPort != 0 {
		netHostPortAsString = strconv.FormatInt(attrs.NetworkAttributes.NetHostPort, 10)
	}

	if attrs.HTTPScheme != "" && attrs.HTTPHost != "" && attrs.HTTPTarget != "" {
		sb.WriteString(attrs.HTTPScheme)
		sb.WriteString("://")
		sb.WriteString(attrs.HTTPHost)
		sb.WriteString(attrs.HTTPTarget)
		data.Url = sb.String()
	} else if attrs.HTTPScheme != "" && attrs.HTTPServerName != "" && netHostPortAsString != "" && attrs.HTTPTarget != "" {
		sb.WriteString(attrs.HTTPScheme)
		sb.WriteString("://")
		sb.WriteString(attrs.HTTPServerName)
		sb.WriteString(":")
		sb.WriteString(netHostPortAsString)
		sb.WriteString(attrs.HTTPTarget)
		data.Url = sb.String()
	} else if attrs.HTTPScheme != "" && attrs.NetworkAttributes.NetHostName != "" && netHostPortAsString != "" && attrs.HTTPTarget != "" {
		sb.WriteString(attrs.HTTPScheme)
		sb.WriteString("://")
		sb.WriteString(attrs.NetworkAttributes.NetHostName)
		sb.WriteString(":")
		sb.WriteString(netHostPortAsString)
		sb.WriteString(attrs.HTTPTarget)
		data.Url = sb.String()
	} else if attrs.HTTPURL != "" {
		if _, err := url.Parse(attrs.HTTPURL); err == nil {
			data.Url = attrs.HTTPURL
		}
	}

	sb.Reset()

	// data.Source should be the client ip if available or fallback to net.peer.ip
	// https://github.com/microsoft/ApplicationInsights-Home/blob/f1f9f619d74557c8db3dbde4b49c4193e10d8a81/EndpointSpecs/Schemas/Bond/RequestData.bond#L28
	// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/http.md#http-server-semantic-conventions
	if attrs.HTTPClientIP != "" {
		data.Source = attrs.HTTPClientIP
	} else if attrs.NetworkAttributes.NetPeerIP != "" {
		data.Source = attrs.NetworkAttributes.NetPeerIP
	}
}

// Maps HTTP Client LogRecord to AppInsights RemoteDependencyData
// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/http.md
func fillRemoteDependencyDataHTTP(logRecord pdata.LogRecord, data *contracts.RemoteDependencyData) {
	attrs := copyAndExtractHTTPAttributes(logRecord.Attributes(), data.Properties, data.Measurements)

	data.Type = "HTTP"
	if attrs.HTTPStatusCode != 0 {
		data.ResultCode, data.Success = getFormattedHTTPStatusValues(attrs.HTTPStatusCode)
	}

	var sb strings.Builder

	// Construct data.Name
	// The data.Name should default to {HTTP METHOD} and include {HTTP ROUTE TEMPLATE} (if available)
	sb.WriteString(attrs.HTTPMethod)

	// Use httpRoute if available otherwise fallback to the HTTP method
	// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/http.md#name
	if attrs.HTTPRoute != "" {
		sb.WriteString(" ")
		sb.WriteString(attrs.HTTPRoute)
	}

	data.Name = sb.String()
	sb.Reset()

	/*
		Order of preference is:
		http.url
		http.scheme, http.host, http.target
		http.scheme, net.peer.name, net.peer.port, http.target
		http.scheme, net.peer.ip, net.peer.port, http.target
	*/

	// prefix httpTarget, if specified
	if attrs.HTTPTarget != "" {
		attrs.HTTPTarget = prefixIfNecessary(attrs.HTTPTarget, "/")
	}

	netPeerPortAsString := ""
	if attrs.NetworkAttributes.NetPeerPort != 0 {
		netPeerPortAsString = strconv.FormatInt(attrs.NetworkAttributes.NetPeerPort, 10)
	}

	if attrs.HTTPURL != "" {
		if u, err := url.Parse(attrs.HTTPURL); err == nil {
			data.Data = attrs.HTTPURL
			data.Target = u.Host
		}
	} else if attrs.HTTPScheme != "" && attrs.HTTPHost != "" && attrs.HTTPTarget != "" {
		sb.WriteString(attrs.HTTPScheme)
		sb.WriteString("://")
		sb.WriteString(attrs.HTTPHost)
		sb.WriteString(attrs.HTTPTarget)
		data.Data = sb.String()
		data.Target = attrs.HTTPHost
	} else if attrs.HTTPScheme != "" && attrs.NetworkAttributes.NetPeerName != "" && netPeerPortAsString != "" && attrs.HTTPTarget != "" {
		sb.WriteString(attrs.HTTPScheme)
		sb.WriteString("://")
		sb.WriteString(attrs.NetworkAttributes.NetPeerName)
		sb.WriteString(":")
		sb.WriteString(netPeerPortAsString)
		sb.WriteString(attrs.HTTPTarget)
		data.Data = sb.String()

		sb.Reset()
		sb.WriteString(attrs.NetworkAttributes.NetPeerName)
		sb.WriteString(":")
		sb.WriteString(netPeerPortAsString)
		data.Target = sb.String()
	} else if attrs.HTTPScheme != "" && attrs.NetworkAttributes.NetPeerIP != "" && netPeerPortAsString != "" && attrs.HTTPTarget != "" {
		sb.WriteString(attrs.HTTPScheme)
		sb.WriteString("://")
		sb.WriteString(attrs.NetworkAttributes.NetPeerIP)
		sb.WriteString(":")
		sb.WriteString(netPeerPortAsString)
		sb.WriteString(attrs.HTTPTarget)
		data.Data = sb.String()

		sb.Reset()
		sb.WriteString(attrs.NetworkAttributes.NetPeerIP)
		sb.WriteString(":")
		sb.WriteString(netPeerPortAsString)
		data.Target = sb.String()
	}
}

// Maps RPC Server LogRecord to AppInsights RequestData
// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/rpc.md
func fillRequestDataRPC(logRecord pdata.LogRecord, data *contracts.RequestData) {
	attrs := copyAndExtractRPCAttributes(logRecord.Attributes(), data.Properties, data.Measurements)

	var sb strings.Builder

	sb.WriteString(attrs.RPCSystem)
	sb.WriteString(" ")
	sb.WriteString(data.Name)

	// Prefix the name with the type of RPC
	data.Name = sb.String()

	// Set the .Data property to .Name which contain the full RPC method
	data.Url = data.Name

	sb.Reset()

	writeFormattedPeerAddressFromNetworkAttributes(&attrs.NetworkAttributes, &sb)

	data.Source = sb.String()
}

// Maps RPC Client LogRecord to AppInsights RemoteDependencyData
// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/rpc.md
func fillRemoteDependencyDataRPC(logRecord pdata.LogRecord, data *contracts.RemoteDependencyData) {
	attrs := copyAndExtractRPCAttributes(logRecord.Attributes(), data.Properties, data.Measurements)

	// Set the .Data property to .Name which contain the full RPC method
	data.Data = data.Name

	data.Type = attrs.RPCSystem

	var sb strings.Builder
	writeFormattedPeerAddressFromNetworkAttributes(&attrs.NetworkAttributes, &sb)
	data.Target = sb.String()
}

// Maps Database Client LogRecord to AppInsights RemoteDependencyData
// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/database.md
func fillRemoteDependencyDataDatabase(logRecord pdata.LogRecord, data *contracts.RemoteDependencyData) {
	attrs := copyAndExtractDatabaseAttributes(logRecord.Attributes(), data.Properties, data.Measurements)

	data.Type = attrs.DBSystem

	if attrs.DBStatement != "" {
		data.Data = attrs.DBStatement
	} else if attrs.DBOperation != "" {
		data.Data = attrs.DBOperation
	}

	var sb strings.Builder
	writeFormattedPeerAddressFromNetworkAttributes(&attrs.NetworkAttributes, &sb)
	data.Target = sb.String()
}

// Maps Messaging Consumer/Server LogRecord to AppInsights RequestData
// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/messaging.md
func fillRequestDataMessaging(logRecord pdata.LogRecord, data *contracts.RequestData) {
	attrs := copyAndExtractMessagingAttributes(logRecord.Attributes(), data.Properties, data.Measurements)

	// TODO Understand how to map attributes to RequestData fields
	if attrs.MessagingURL != "" {
		data.Source = attrs.MessagingURL
	} else {
		var sb strings.Builder
		writeFormattedPeerAddressFromNetworkAttributes(&attrs.NetworkAttributes, &sb)
		data.Source = sb.String()
	}
}

// Maps Messaging Producer/Client LogRecord to AppInsights RemoteDependencyData
// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/messaging.md
func fillRemoteDependencyDataMessaging(logRecord pdata.LogRecord, data *contracts.RemoteDependencyData) {
	attrs := copyAndExtractMessagingAttributes(logRecord.Attributes(), data.Properties, data.Measurements)

	// TODO Understand how to map attributes to RemoteDependencyData fields
	data.Data = attrs.MessagingURL
	data.Type = attrs.MessagingSystem

	if attrs.MessagingURL != "" {
		data.Target = attrs.MessagingURL
	} else {
		var sb strings.Builder
		writeFormattedPeerAddressFromNetworkAttributes(&attrs.NetworkAttributes, &sb)
		data.Target = sb.String()
	}
}

// Copies all attributes to either properties or measurements and passes the key/value to another mapping function
func copyAndMapAttributes(
	attributeMap pdata.AttributeMap,
	properties map[string]string,
	measurements map[string]float64,
	mappingFunc func(k string, v pdata.AttributeValue)) {

	attributeMap.ForEach(
		func(k string, v pdata.AttributeValue) {
			setAttributeValueAsPropertyOrMeasurement(k, v, properties, measurements)

			if mappingFunc != nil {
				mappingFunc(k, v)
			}
		})
}

// Copies all attributes to either properties or measurements without any kind of mapping to a known set of attributes
func copyAttributesWithoutMapping(
	attributeMap pdata.AttributeMap,
	properties map[string]string,
	measurements map[string]float64) {

	copyAndMapAttributes(attributeMap, properties, measurements, nil)
}

// Attribute extraction logic for HTTP LogRecord attributes
func copyAndExtractHTTPAttributes(
	attributeMap pdata.AttributeMap,
	properties map[string]string,
	measurements map[string]float64) *HTTPAttributes {

	attrs := &HTTPAttributes{}
	copyAndMapAttributes(
		attributeMap,
		properties,
		measurements,
		func(k string, v pdata.AttributeValue) { attrs.MapAttribute(k, v) })

	return attrs
}

// Attribute extraction logic for RPC LogRecord attributes
func copyAndExtractRPCAttributes(
	attributeMap pdata.AttributeMap,
	properties map[string]string,
	measurements map[string]float64) *RPCAttributes {

	attrs := &RPCAttributes{}
	copyAndMapAttributes(
		attributeMap,
		properties,
		measurements,
		func(k string, v pdata.AttributeValue) { attrs.MapAttribute(k, v) })

	return attrs
}

// Attribute extraction logic for Database LogRecord attributes
func copyAndExtractDatabaseAttributes(
	attributeMap pdata.AttributeMap,
	properties map[string]string,
	measurements map[string]float64) *DatabaseAttributes {

	attrs := &DatabaseAttributes{}
	copyAndMapAttributes(
		attributeMap,
		properties,
		measurements,
		func(k string, v pdata.AttributeValue) { attrs.MapAttribute(k, v) })

	return attrs
}

// Attribute extraction logic for Messaging LogRecord attributes
func copyAndExtractMessagingAttributes(
	attributeMap pdata.AttributeMap,
	properties map[string]string,
	measurements map[string]float64) *MessagingAttributes {

	attrs := &MessagingAttributes{}
	copyAndMapAttributes(
		attributeMap,
		properties,
		measurements,
		func(k string, v pdata.AttributeValue) { attrs.MapAttribute(k, v) })

	return attrs
}

// Maps incoming LogRecord to a type defined in the specification
func mapIncomingLogRecordToType(attributeMap pdata.AttributeMap) logRecordType {
	// No attributes
	if attributeMap.Len() == 0 {
		return unknownLogRecordType
	}

	// HTTP
	if _, exists := attributeMap.Get(conventions.AttributeHTTPMethod); exists {
		return httpLogRecordType
	}

	// RPC
	if _, exists := attributeMap.Get(conventions.AttributeRPCSystem); exists {
		return rpcLogRecordType
	}

	// Database
	if _, exists := attributeMap.Get(attributeDBSystem); exists {
		return databaseLogRecordType
	}

	// Messaging
	if _, exists := attributeMap.Get(conventions.AttributeMessagingSystem); exists {
		return messagingLogRecordType
	}

	if _, exists := attributeMap.Get(conventions.AttributeFaaSTrigger); exists {
		return faasLogRecordType
	}

	return unknownLogRecordType
}

func writeFormattedPeerAddressFromNetworkAttributes(networkAttributes *NetworkAttributes, sb *strings.Builder) {
	// Favor name over IP for
	if networkAttributes.NetPeerName != "" {
		sb.WriteString(networkAttributes.NetPeerName)
	} else if networkAttributes.NetPeerIP != "" {
		sb.WriteString(networkAttributes.NetPeerIP)
	}

	if networkAttributes.NetPeerPort != 0 {
		sb.WriteString(":")
		sb.WriteString(strconv.FormatInt(networkAttributes.NetPeerPort, 10))
	}
}

func setAttributeValueAsPropertyOrMeasurement(
	key string,
	attributeValue pdata.AttributeValue,
	properties map[string]string,
	measurements map[string]float64) {

	switch attributeValue.Type() {
	case pdata.AttributeValueBOOL:
		properties[key] = strconv.FormatBool(attributeValue.BoolVal())

	case pdata.AttributeValueSTRING:
		properties[key] = attributeValue.StringVal()

	case pdata.AttributeValueINT:
		measurements[key] = float64(attributeValue.IntVal())

	case pdata.AttributeValueDOUBLE:
		measurements[key] = float64(attributeValue.DoubleVal())
	}
}

func prefixIfNecessary(s string, prefix string) string {
	if strings.HasPrefix(s, prefix) {
		return s
	}

	return prefix + s
}

func sanitize(sanitizeFunc func() []string, logger *zap.Logger) {
	sanitizeWithCallback(sanitizeFunc, nil, logger)
}

func sanitizeWithCallback(sanitizeFunc func() []string, warningCallback func(string), logger *zap.Logger) {
	sanitizeWarnings := sanitizeFunc()
	for _, warning := range sanitizeWarnings {
		if warningCallback == nil {
			// TODO error handling
			logger.Warn(warning)
		} else {
			warningCallback(warning)
		}
	}
}
