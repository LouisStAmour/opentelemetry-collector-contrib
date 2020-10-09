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
	"go.opentelemetry.io/collector/consumer/pdata"
)


const (
	defaultMessageDataEnvelopeName          = "Microsoft.ApplicationInsights.Message"
	defaultExceptionDataEnvelopeName 		= "Microsoft.ApplicationInsights.Exception"
	defaultEventDataEnvelopeName			= "Microsoft.ApplicationInsights.Event"
	defaultExceptionDataBaseType			= "ExceptionData"
	defaultMessageDataBaseType				= "MessageData"
	defaultEventDataBaseType				= "EventData"
	defaultLogTimestamp          			= pdata.TimestampUnixNano(0)
)

const (
	DefaultFlag = 0b00000000
	SampledFlag = 0b00000001
)

/*
	The remainder of these methods are for building up test assets
*/
func getLogRecord(name string,
	body string,
	droppedAttributesCount uint32,
	flags uint32,
	severityNumber pdata.SeverityNumber,
	severityText string,
	initialAttributes map[string]pdata.AttributeValue) pdata.LogRecord {
	logRecord := pdata.NewLogRecord()
	logRecord.InitEmpty()
	logRecord.Attributes().InitFromMap(initialAttributes)
	logRecord.Body().SetStringVal(body)
	logRecord.SetDroppedAttributesCount(droppedAttributesCount)
	logRecord.SetFlags(flags)
	logRecord.SetName(name)
	logRecord.SetSeverityNumber(severityNumber)
	logRecord.SetSeverityText(severityText)
	logRecord.SetSpanID(pdata.NewSpanID(defaultSpanID))
	logRecord.SetTimestamp(defaultLogTimestamp)
	logRecord.SetTraceID(pdata.NewTraceID(defaultTraceID))
	return logRecord
}

func getDefaultHTTPServerLog() pdata.LogRecord {
	lr := pdata.NewLogRecord()
	lr.InitEmpty()
	lr.SetName("logName")
	lr.Body().SetStringVal("mylog")
	lr.SetSeverityNumber(pdata.SeverityNumberINFO)
	lr.SetSeverityText("INFO")
	lr.SetFlags(SampledFlag)
	lr.Attributes().InitFromMap(map[string]pdata.AttributeValue{
		"keyString": pdata.NewAttributeValueString("arithmetic"),
		"keyInt":    pdata.NewAttributeValueInt(123),
		"keyDouble": pdata.NewAttributeValueDouble(3245.6),
		"keyBool":   pdata.NewAttributeValueBool(true),
		"keyExists": pdata.NewAttributeValueString("present"),
	})
	lr.SetSpanID(pdata.NewSpanID(defaultSpanID))
	lr.SetTimestamp(defaultLogTimestamp)
	lr.SetTraceID(pdata.NewTraceID(defaultTraceID))
	return lr
}
