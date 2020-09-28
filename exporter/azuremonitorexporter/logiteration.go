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

import "go.opentelemetry.io/collector/consumer/pdata"

/*
	Encapsulates iteration over the LogRecords inside pdata.Logs from the underlying representation.
	Everyone is doing the same kind of iteration and checking over a set traces.
*/

// LogVisitor interface defines a iteration callback when walking through traces
type LogVisitor interface {
	// Called for each tuple of Resource, InstrumentationLibrary, and LogRecord
	// If Visit returns false, the iteration is short-circuited
	visit(resource pdata.Resource, instrumentationLibrary pdata.InstrumentationLibrary, logRecord pdata.LogRecord) (ok bool)
}

// Accept method is called to start the iteration process
func AcceptLogs(logs pdata.Logs, v LogVisitor) {
	resourceLogRecords := logs.ResourceLogs()

	// Walk each ResourceLogRecords instance
	for i := 0; i < resourceLogRecords.Len(); i++ {
		rl := resourceLogRecords.At(i)
		if rl.IsNil() {
			continue
		}

		resource := rl.Resource()
		instrumentationLibraryLogRecordsSlice := rl.InstrumentationLibraryLogs()

		if resource.IsNil() {
			// resource is required
			continue
		}

		for i := 0; i < instrumentationLibraryLogRecordsSlice.Len(); i++ {
			instrumentationLibraryLogRecords := instrumentationLibraryLogRecordsSlice.At(i)

			if instrumentationLibraryLogRecords.IsNil() {
				continue
			}

			// instrumentation library is optional
			instrumentationLibrary := instrumentationLibraryLogRecords.InstrumentationLibrary()
			logsSlice := instrumentationLibraryLogRecords.Logs()
			if logsSlice.Len() == 0 {
				continue
			}

			for i := 0; i < logsSlice.Len(); i++ {
				span := logsSlice.At(i)
				if span.IsNil() {
					continue
				}

				if ok := v.visit(resource, instrumentationLibrary, span); !ok {
					return
				}
			}
		}
	}
}
