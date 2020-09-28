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
	Encapsulates iteration over the Metrics inside pdata.Metrics from the underlying representation.
	Everyone is doing the same kind of iteration and checking over a set traces.
*/

// MetricVisitor interface defines a iteration callback when walking through traces
type MetricVisitor interface {
	// Called for each tuple of Resource, InstrumentationLibrary, and MetricRecord
	// If Visit returns false, the iteration is short-circuited
	visit(resource pdata.Resource, instrumentationLibrary pdata.InstrumentationLibrary, metric pdata.Metric) (ok bool)
}

// Accept method is called to start the iteration process
func AcceptMetrics(metrics pdata.Metrics, v MetricVisitor) {
	resourceMetrics := metrics.ResourceMetrics()

	// Walk each ResourceMetrics instance
	for i := 0; i < resourceMetrics.Len(); i++ {
		rl := resourceMetrics.At(i)
		if rl.IsNil() {
			continue
		}

		resource := rl.Resource()
		instrumentationLibraryMetricsSlice := rl.InstrumentationLibraryMetrics()

		if resource.IsNil() {
			// resource is required
			continue
		}

		for i := 0; i < instrumentationLibraryMetricsSlice.Len(); i++ {
			instrumentationLibraryMetrics := instrumentationLibraryMetricsSlice.At(i)

			if instrumentationLibraryMetrics.IsNil() {
				continue
			}

			// instrumentation library is optional
			instrumentationLibrary := instrumentationLibraryMetrics.InstrumentationLibrary()
			metricsSlice := instrumentationLibraryMetrics.Metrics()
			if metricsSlice.Len() == 0 {
				continue
			}

			for i := 0; i < metricsSlice.Len(); i++ {
				span := metricsSlice.At(i)
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
