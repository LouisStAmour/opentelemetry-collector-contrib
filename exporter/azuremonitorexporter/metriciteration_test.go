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

	mock "github.com/stretchr/testify/mock"
	"go.opentelemetry.io/collector/consumer/pdata"
)

type mockMetricVisitor struct {
	mock.Mock
}

func (v *mockMetricVisitor) visit(resource pdata.Resource, instrumentationLibrary pdata.InstrumentationLibrary, metricRecord pdata.MetricRecord) (ok bool) {
	args := v.Called(resource, instrumentationLibrary, metricRecord)
	return args.Bool(0)
}

// Tests the iteration metricic over a pdata.Metrics type when no ResourceMetrics are provided
func TestMetricDataIterationNoResourceMetrics(t *testing.T) {
	metrics := pdata.NewMetrics()

	visitor := getMockMetricVisitor(true)

	AcceptMetrics(metrics, visitor)

	visitor.AssertNumberOfCalls(t, "visit", 0)
}

// Tests the iteration metricic over a pdata.Metrics type when a ResourceMetrics is nil
func TestMetricDataIterationResourceMetricsIsNil(t *testing.T) {
	metrics := pdata.NewMetrics()
	resourceMetrics := pdata.NewResourceMetrics()
	metrics.ResourceMetrics().Append(resourceMetrics)

	visitor := getMockMetricVisitor(true)

	AcceptMetrics(metrics, visitor)

	visitor.AssertNumberOfCalls(t, "visit", 0)
}

// Tests the iteration metricic over a pdata.Metrics type when a Resource is nil
func TestMetricDataIterationResourceIsNil(t *testing.T) {
	metrics := pdata.NewMetrics()
	metrics.ResourceMetrics().Resize(1)

	visitor := getMockMetricVisitor(true)

	AcceptMetrics(metrics, visitor)

	visitor.AssertNumberOfCalls(t, "visit", 0)
}

// Tests the iteration metricic over a pdata.Metrics type when InstrumentationLibraryMetrics is nil
func TestMetricDataIterationInstrumentationLibraryMetricsIsNil(t *testing.T) {
	metrics := pdata.NewMetrics()
	metrics.ResourceMetrics().Resize(1)
	rl := metrics.ResourceMetrics().At(0)
	r := rl.Resource()
	r.InitEmpty()
	instrumentationLibraryMetrics := pdata.NewInstrumentationLibraryMetrics()
	rl.InstrumentationLibraryMetrics().Append(instrumentationLibraryMetrics)

	visitor := getMockMetricVisitor(true)

	AcceptMetrics(metrics, visitor)

	visitor.AssertNumberOfCalls(t, "visit", 0)
}

// Tests the iteration metricic over a pdata.Metrics type when there are no Metrics
func TestMetricDataIterationNoMetrics(t *testing.T) {
	metrics := pdata.NewMetrics()
	metrics.ResourceMetrics().Resize(1)
	rl := metrics.ResourceMetrics().At(0)
	r := rl.Resource()
	r.InitEmpty()
	rl.InstrumentationLibraryMetrics().Resize(1)

	visitor := getMockMetricVisitor(true)

	AcceptMetrics(metrics, visitor)

	visitor.AssertNumberOfCalls(t, "visit", 0)
}

// Tests the iteration metricic over a pdata.Metrics type when the MetricRecord is nil
func TestMetricDataIterationMetricRecordIsNil(t *testing.T) {
	metrics := pdata.NewMetrics()
	metrics.ResourceMetrics().Resize(1)
	rl := metrics.ResourceMetrics().At(0)
	r := rl.Resource()
	r.InitEmpty()
	rl.InstrumentationLibraryMetrics().Resize(1)
	illr := rl.InstrumentationLibraryMetrics().At(0)
	metricRecord := pdata.NewMetricRecord()
	illr.Metrics().Append(metricRecord)

	visitor := getMockMetricVisitor(true)

	AcceptMetrics(metrics, visitor)

	visitor.AssertNumberOfCalls(t, "visit", 0)
}

// Tests the iteration metricic if the visitor returns true
func TestMetricDataIterationNoShortCircuit(t *testing.T) {
	metrics := pdata.NewMetrics()
	metrics.ResourceMetrics().Resize(1)
	rl := metrics.ResourceMetrics().At(0)
	r := rl.Resource()
	r.InitEmpty()
	rl.InstrumentationLibraryMetrics().Resize(1)
	illr := rl.InstrumentationLibraryMetrics().At(0)
	illr.Metrics().Resize(2)

	visitor := getMockMetricVisitor(true)

	AcceptMetrics(metrics, visitor)

	visitor.AssertNumberOfCalls(t, "visit", 2)
}

// Tests the iteration metricic short circuit if the visitor returns false
func TestMetricDataIterationShortCircuit(t *testing.T) {
	metrics := pdata.NewMetrics()
	metrics.ResourceMetrics().Resize(1)
	rl := metrics.ResourceMetrics().At(0)
	r := rl.Resource()
	r.InitEmpty()
	rl.InstrumentationLibraryMetrics().Resize(1)
	illr := rl.InstrumentationLibraryMetrics().At(0)
	illr.Metrics().Resize(2)

	visitor := getMockMetricVisitor(false)

	AcceptMetrics(metrics, visitor)

	visitor.AssertNumberOfCalls(t, "visit", 1)
}

func getMockMetricVisitor(returns bool) *mockMetricVisitor {
	visitor := new(mockMetricVisitor)
	visitor.On("visit", mock.Anything, mock.Anything, mock.Anything).Return(returns)
	return visitor
}
